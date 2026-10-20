// Package handler 实现各资源的 HTTP 接口。
package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/steven754/dockermanage/backend/internal/httpx"
	"github.com/steven754/dockermanage/backend/internal/repo"
	"github.com/steven754/dockermanage/backend/internal/service/auth"
	"github.com/steven754/dockermanage/backend/internal/service/compose"
	"github.com/steven754/dockermanage/backend/internal/service/docker"
	"github.com/steven754/dockermanage/backend/internal/service/events"
	"github.com/steven754/dockermanage/backend/internal/service/settings"
	"github.com/steven754/dockermanage/backend/internal/service/stats"
)

// Handler 聚合各资源的处理器
type Handler struct {
	auth       *auth.Service
	containers *docker.ContainerService
	images     *docker.ImageService
	volumes    *docker.VolumeService
	networks   *docker.NetworkService
	system     *docker.SystemService
	sampler    *stats.Sampler
	events     *events.Service
	compose    *compose.Service
	settings   *settings.Service
	store      *repo.Store

	// collector 即时采集宿主机资源（Dashboard 首次访问时兜底）
	collector stats.Collector
	// diskMountpoint 磁盘统计的挂载点
	diskMountpoint string
}

// Deps 处理器依赖
type Deps struct {
	Auth           *auth.Service
	Containers     *docker.ContainerService
	Images         *docker.ImageService
	Volumes        *docker.VolumeService
	Networks       *docker.NetworkService
	System         *docker.SystemService
	Sampler        *stats.Sampler
	Events         *events.Service
	Compose        *compose.Service
	Settings       *settings.Service
	Store          *repo.Store
	Collector      stats.Collector
	DiskMountpoint string
}

// New 创建处理器
func New(d Deps) *Handler {
	mountpoint := d.DiskMountpoint
	if mountpoint == "" {
		mountpoint = "/"
	}
	return &Handler{
		auth:           d.Auth,
		containers:     d.Containers,
		images:         d.Images,
		volumes:        d.Volumes,
		networks:       d.Networks,
		system:         d.System,
		sampler:        d.Sampler,
		events:         d.Events,
		compose:        d.Compose,
		settings:       d.Settings,
		store:          d.Store,
		collector:      d.Collector,
		diskMountpoint: mountpoint,
	}
}

// logInput 记录请求输入
func logInput(r *http.Request, args ...any) {
	slog.Info("收到请求", append([]any{"method", r.Method, "path", r.URL.Path}, args...)...)
}

// writeInternalError 写出内部错误
func writeInternalError(w http.ResponseWriter, r *http.Request, message string) {
	httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, message)
}

// writeDockerError 写出 Docker 上游错误
//
// 错误信息面向用户，故使用中文描述作为主句，Docker 原始错误作为补充细节。
func writeDockerError(w http.ResponseWriter, r *http.Request, err error) {
	writeDockerErrorWithCode(w, r, err, httpx.CodeNotFound)
}

// writeDockerErrorWithCode 写出 Docker 上游错误，可指定 404 时的具体错误码
func writeDockerErrorWithCode(w http.ResponseWriter, r *http.Request, err error, notFoundCode string) {
	switch {
	case docker.IsNotFound(err):
		httpx.WriteError(w, r, http.StatusNotFound, notFoundCode, "资源不存在或已被删除："+detailOf(err))
	case docker.IsConflict(err):
		httpx.WriteError(w, r, http.StatusConflict, httpx.CodeConflict, "当前状态不允许该操作："+detailOf(err))
	default:
		httpx.WriteError(w, r, http.StatusBadGateway, httpx.CodeDockerAPIError, detailOf(err))
	}
}

// detailOf 提取错误细节，去掉重复的中文操作前缀
func detailOf(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if idx := strings.Index(msg, ": "); idx >= 0 && idx+2 < len(msg) {
		return msg[idx+2:]
	}
	return msg
}

// writeValidationError 写出参数校验错误
func writeValidationError(w http.ResponseWriter, r *http.Request, message string) {
	httpx.WriteError(w, r, http.StatusBadRequest, httpx.CodeValidation, message)
}
