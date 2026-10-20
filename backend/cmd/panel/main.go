// Command panel 是 Dockermanage 后端服务入口。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/steven754/dockermanage/backend/internal/config"
	"github.com/steven754/dockermanage/backend/internal/db"
	"github.com/steven754/dockermanage/backend/internal/handler"
	"github.com/steven754/dockermanage/backend/internal/repo"
	"github.com/steven754/dockermanage/backend/internal/server"
	"github.com/steven754/dockermanage/backend/internal/service/auth"
	"github.com/steven754/dockermanage/backend/internal/service/compose"
	"github.com/steven754/dockermanage/backend/internal/service/docker"
	"github.com/steven754/dockermanage/backend/internal/service/events"
	"github.com/steven754/dockermanage/backend/internal/service/settings"
	"github.com/steven754/dockermanage/backend/internal/service/stats"
	"github.com/steven754/dockermanage/backend/internal/version"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		slog.Error("服务启动失败", "error", err)
		os.Exit(1)
	}
}

// run 执行服务主流程
func run(args []string) error {
	cfg, err := config.Load(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return fmt.Errorf("解析配置失败: %w", err)
	}

	initLogger(cfg.LogLevel)
	if cfg.Timezone != "" {
		if err := setTimezone(cfg.Timezone); err != nil {
			slog.Warn("设置时区失败", "timezone", cfg.Timezone, "error", err)
		}
	}

	slog.Info("Dockermanage 后端启动中",
		"version", version.Version,
		"commit", version.Commit,
		"buildTime", version.BuildTime,
		"port", cfg.Port,
		"dataDir", cfg.DataDir,
		"composeDir", cfg.ComposeDir)

	if err := cfg.EnsureDirs(); err != nil {
		return err
	}

	// 数据库
	sqlDB, err := db.Open(cfg.DataDir, false)
	if err != nil {
		return err
	}
	defer func() { _ = sqlDB.Close() }()

	if err := db.Migrate(sqlDB); err != nil {
		return err
	}
	store := repo.New(sqlDB)

	// Docker 客户端
	dockerClient, err := docker.New(cfg.DockerHost)
	if err != nil {
		return err
	}
	defer func() { _ = dockerClient.Close() }()

	containerSvc := docker.NewContainerService(dockerClient)
	imageSvc := docker.NewImageService(dockerClient)
	volumeSvc := docker.NewVolumeService(dockerClient)
	networkSvc := docker.NewNetworkService(dockerClient)
	systemSvc := docker.NewSystemService(dockerClient)

	// 认证
	secret, err := auth.LoadOrCreateSecret(cfg.DataDir)
	if err != nil {
		return err
	}
	tokenManager := auth.NewTokenManager(secret)
	authSvc := auth.New(store.Users, tokenManager, cfg.DataDir)

	// 采样器
	collector := stats.NewHostCollector(diskMountpoint())
	sampler := stats.New(
		store.SystemStats,
		store.ContainerStats,
		collector,
		time.Duration(cfg.SampleIntervalSeconds)*time.Second,
		time.Duration(cfg.RetentionDays)*24*time.Hour,
	)
	containerSvc.SetStatsProvider(sampler)
	containerSampler := stats.NewContainerSampler(containerSvc, sampler, time.Duration(cfg.SampleIntervalSeconds)*time.Second)

	// 事件
	eventSvc := events.New(systemSvc, store.Events, time.Duration(cfg.RetentionDays)*24*time.Hour)

	// 设置
	settingsSvc := settings.New(store.Settings)

	// Compose
	composeSvc := compose.New(cfg.ComposeDir, cfg.DockerHost)

	// 处理器
	h := handler.New(handler.Deps{
		Auth:           authSvc,
		Containers:     containerSvc,
		Images:         imageSvc,
		Volumes:        volumeSvc,
		Networks:       networkSvc,
		System:         systemSvc,
		Sampler:        sampler,
		Events:         eventSvc,
		Compose:        composeSvc,
		Settings:       settingsSvc,
		Store:          store,
		Collector:      collector,
		DiskMountpoint: diskMountpoint(),
	})

	router := server.NewRouter(server.Deps{
		Handler: h,
		Tokens:  tokenManager,
		WebDir:  cfg.WebDir,
	})

	// 首启：生成初始密码
	if _, err := authSvc.Bootstrap(context.Background()); err != nil {
		return fmt.Errorf("初始化用户失败: %w", err)
	}

	// 后台任务
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go sampler.Run(ctx)
	go containerSampler.Run(ctx)
	go eventSvc.Run(ctx)
	go runCleanup(ctx, sampler, eventSvc)

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           router,
		ReadHeaderTimeout: 15 * time.Second,
		// SSE / WebSocket 为长连接，不设置 WriteTimeout
		IdleTimeout: 120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("HTTP 服务已启动", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	// 等待退出信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		cancel()
		return fmt.Errorf("HTTP 服务异常退出: %w", err)
	case sig := <-sigCh:
		slog.Info("收到退出信号，开始优雅关闭", "signal", sig.String())
	}

	// 优雅退出：停采样器 -> 关 HTTP -> 关 DB
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("关闭 HTTP 服务失败", "error", err)
	}
	slog.Info("服务已退出")
	return nil
}

// runCleanup 定期清理过期数据
func runCleanup(ctx context.Context, sampler *stats.Sampler, eventSvc *events.Service) {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sampler.Cleanup(ctx)
			eventSvc.Cleanup(ctx)
		}
	}
}

// initLogger 初始化结构化日志
func initLogger(level string) {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	slog.SetDefault(slog.New(handler))
}

// setTimezone 设置进程时区
func setTimezone(tz string) error {
	return os.Setenv("TZ", tz)
}

// diskMountpoint 返回磁盘统计挂载点
func diskMountpoint() string {
	return "/"
}
