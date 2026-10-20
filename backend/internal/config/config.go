// Package config 负责解析环境变量与命令行 flag，产出运行期配置。
package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config 运行期配置
type Config struct {
	// Port HTTP 监听端口
	Port int
	// DataDir 数据目录（SQLite / .jwt_secret / .password）
	DataDir string
	// ComposeDir compose 项目根目录
	ComposeDir string
	// LogLevel 日志级别：debug/info/warn/error
	LogLevel string
	// WebDir 前端静态资源目录（为空则使用内嵌资源）
	WebDir string
	// DockerHost Docker daemon 地址，空则取环境变量 DOCKER_HOST / 默认 unix socket
	DockerHost string
	// Timezone 时区
	Timezone string
	// SampleIntervalSeconds stats 采样间隔（秒）
	SampleIntervalSeconds int
	// RetentionDays stats / events 保留天数
	RetentionDays int
	// InitialPassword 首启初始密码（仅数据库无用户时生效一次）
	InitialPassword string
}

// DefaultInitialPassword 默认初始密码。
//
// 沿用 Portainer 的体验：默认 admin / admin，登录后强制改密。
// 需要更强默认值时用环境变量 PANEL_INITIAL_PASSWORD 覆盖。
const DefaultInitialPassword = "admin"

// Default 返回默认配置
func Default() Config {
	return Config{
		Port:                  8080,
		DataDir:               "/data",
		ComposeDir:            "/compose",
		LogLevel:              "info",
		WebDir:                "",
		DockerHost:            "",
		Timezone:              "",
		SampleIntervalSeconds: 10,
		RetentionDays:         7,
		InitialPassword:       DefaultInitialPassword,
	}
}

// Load 解析 flag + 环境变量。flag 优先级高于环境变量，环境变量优先级高于默认值。
func Load(args []string) (Config, error) {
	cfg := Default()

	fs := flag.NewFlagSet("panel", flag.ContinueOnError)
	fs.IntVar(&cfg.Port, "port", cfg.Port, "HTTP 监听端口（环境变量 PANEL_PORT）")
	fs.StringVar(&cfg.DataDir, "data", cfg.DataDir, "数据目录，存放 SQLite 与密钥文件（环境变量 PANEL_DATA）")
	fs.StringVar(&cfg.ComposeDir, "compose-dir", cfg.ComposeDir, "compose 项目根目录（环境变量 PANEL_COMPOSE_DIR）")
	fs.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "日志级别 debug|info|warn|error（环境变量 PANEL_LOG_LEVEL）")
	fs.StringVar(&cfg.WebDir, "web-dir", cfg.WebDir, "前端静态资源目录，留空使用内嵌资源（环境变量 PANEL_WEB_DIR）")
	fs.StringVar(&cfg.DockerHost, "docker-host", cfg.DockerHost, "Docker daemon 地址，留空取 DOCKER_HOST 环境变量（环境变量 PANEL_DOCKER_HOST）")
	fs.StringVar(&cfg.Timezone, "tz", cfg.Timezone, "时区，如 Asia/Shanghai（环境变量 TZ）")
	fs.StringVar(&cfg.InitialPassword, "initial-password", cfg.InitialPassword, "首启初始密码，仅数据库无用户时生效一次（环境变量 PANEL_INITIAL_PASSWORD）")

	if err := fs.Parse(args); err != nil {
		return cfg, err
	}

	// 环境变量覆盖 flag 默认值（flag 显式指定时优先级最高，故仅在未被显式设置时覆盖）
	if !fsWasSet(fs, "port") {
		if v, ok := os.LookupEnv("PANEL_PORT"); ok {
			p, err := strconv.Atoi(strings.TrimSpace(v))
			if err != nil {
				return cfg, fmt.Errorf("PANEL_PORT 非法: %w", err)
			}
			cfg.Port = p
		}
	}
	if !fsWasSet(fs, "data") {
		if v, ok := os.LookupEnv("PANEL_DATA"); ok && v != "" {
			cfg.DataDir = v
		}
	}
	if !fsWasSet(fs, "compose-dir") {
		if v, ok := os.LookupEnv("PANEL_COMPOSE_DIR"); ok && v != "" {
			cfg.ComposeDir = v
		}
	}
	if !fsWasSet(fs, "log-level") {
		if v, ok := os.LookupEnv("PANEL_LOG_LEVEL"); ok && v != "" {
			cfg.LogLevel = v
		}
	}
	if !fsWasSet(fs, "web-dir") {
		if v, ok := os.LookupEnv("PANEL_WEB_DIR"); ok && v != "" {
			cfg.WebDir = v
		}
	}
	if !fsWasSet(fs, "docker-host") {
		if v, ok := os.LookupEnv("PANEL_DOCKER_HOST"); ok && v != "" {
			cfg.DockerHost = v
		}
	}
	if !fsWasSet(fs, "initial-password") {
		if v, ok := os.LookupEnv("PANEL_INITIAL_PASSWORD"); ok && v != "" {
			cfg.InitialPassword = v
		}
	}
	if !fsWasSet(fs, "tz") {
		if v, ok := os.LookupEnv("TZ"); ok && v != "" {
			cfg.Timezone = v
		}
	}

	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// fsWasSet 判断 flag 是否被显式指定
func fsWasSet(fs *flag.FlagSet, name string) bool {
	found := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

// Validate 校验配置合法性
func (c *Config) Validate() error {
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("端口非法: %d", c.Port)
	}
	if c.DataDir == "" {
		return fmt.Errorf("数据目录不能为空")
	}
	if c.ComposeDir == "" {
		return fmt.Errorf("compose 目录不能为空")
	}
	switch strings.ToLower(c.LogLevel) {
	case "debug", "info", "warn", "error":
		c.LogLevel = strings.ToLower(c.LogLevel)
	default:
		return fmt.Errorf("日志级别非法: %s", c.LogLevel)
	}
	c.DataDir = filepath.Clean(c.DataDir)
	c.ComposeDir = filepath.Clean(c.ComposeDir)
	return nil
}

// EnsureDirs 确保数据目录与 compose 目录存在
func (c *Config) EnsureDirs() error {
	if err := os.MkdirAll(c.DataDir, 0o750); err != nil {
		return fmt.Errorf("创建数据目录失败: %w", err)
	}
	if err := os.MkdirAll(c.ComposeDir, 0o750); err != nil {
		return fmt.Errorf("创建 compose 目录失败: %w", err)
	}
	return nil
}
