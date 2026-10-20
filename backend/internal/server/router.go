package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/steven754/dockermanage/backend/internal/handler"
	"github.com/steven754/dockermanage/backend/internal/repo"
	"github.com/steven754/dockermanage/backend/internal/service/auth"
)

// Deps 路由装配依赖
type Deps struct {
	// Handler 业务处理器
	Handler *handler.Handler
	// Tokens JWT 管理器
	Tokens *auth.TokenManager
	// Users 用户仓储，用于校验令牌版本与强制改密状态
	Users *repo.UserRepo
	// WebDir 前端静态资源目录，留空使用内嵌资源
	WebDir string
	// TrustProxy 是否采信 X-Forwarded-For / X-Real-IP
	TrustProxy bool
	// LoginRateLimit 每分钟允许的登录尝试次数
	LoginRateLimit int
}

// NewRouter 构建 HTTP 路由
//
// 所有接口都直接注册在根路由上（不使用 r.Route 挂载子路由），
// 这样未匹配的 /api/v1/** 路径会落到静态资源处理器并返回统一的 JSON 404。
func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(RequestLogger)
	r.Use(Recoverer)

	h := d.Handler

	// 健康检查（无需鉴权）
	r.Get("/healthz", h.Healthz)
	r.Get("/readyz", h.Readyz)

	// 无需鉴权的接口
	r.Get("/api/v1/version", h.Version)
	loginLimit := d.LoginRateLimit
	if loginLimit <= 0 {
		loginLimit = 5
	}
	r.With(LimitLogin(NewRateLimiter(loginLimit, time.Minute), d.TrustProxy)).Post("/api/v1/auth/login", h.Login)

	// 需要鉴权的接口
	authed := func(method, pattern string, fn http.HandlerFunc) {
		r.With(Authenticate(d.Tokens, d.Users)).Method(method, pattern, fn)
	}

	// 认证
	authed(http.MethodPost, "/api/v1/auth/logout", h.Logout)
	authed(http.MethodPost, "/api/v1/auth/password", h.ChangePassword)
	authed(http.MethodGet, "/api/v1/auth/me", h.Me)

	// Dashboard
	authed(http.MethodGet, "/api/v1/dashboard", h.GetDashboard)

	// 容器
	authed(http.MethodGet, "/api/v1/containers", h.ListContainers)
	authed(http.MethodGet, "/api/v1/containers/{id}", h.GetContainer)
	authed(http.MethodGet, "/api/v1/containers/{id}/json", h.InspectContainerJSON)
	authed(http.MethodGet, "/api/v1/containers/{id}/stats", h.ContainerStats)
	authed(http.MethodPost, "/api/v1/containers/{id}/update", h.UpdateContainer)
	authed(http.MethodPost, "/api/v1/containers/prune", h.PruneContainers)
	authed(http.MethodPost, "/api/v1/containers/{id}/{action}", h.ContainerAction)
	authed(http.MethodDelete, "/api/v1/containers/{id}", h.DeleteContainer)
	authed(http.MethodGet, "/api/v1/containers/{id}/logs", h.ContainerLogs)
	authed(http.MethodGet, "/api/v1/containers/{id}/exec", h.ContainerExec)

	// 镜像
	authed(http.MethodGet, "/api/v1/images", h.ListImages)
	authed(http.MethodPost, "/api/v1/images/pull", h.PullImage)
	authed(http.MethodPost, "/api/v1/images/prune", h.PruneImages)
	authed(http.MethodPost, "/api/v1/images/{id}/tag", h.TagImage)
	// 镜像 / 网络的引用可能带斜杠（如 myrepo/nginx:1.27），用通配路由整段匹配
	authed(http.MethodDelete, "/api/v1/images/*", h.DeleteImage)

	// 卷
	authed(http.MethodGet, "/api/v1/volumes", h.ListVolumes)
	authed(http.MethodPost, "/api/v1/volumes/prune", h.PruneVolumes)
	authed(http.MethodDelete, "/api/v1/volumes/{name}", h.DeleteVolume)

	// 网络
	authed(http.MethodGet, "/api/v1/networks", h.ListNetworks)
	authed(http.MethodDelete, "/api/v1/networks/*", h.DeleteNetwork)

	// 事件
	authed(http.MethodGet, "/api/v1/events", h.ListEvents)
	authed(http.MethodGet, "/api/v1/events/history", h.ListEvents)
	authed(http.MethodGet, "/api/v1/events/stream", h.EventsStream)

	// 指标
	authed(http.MethodGet, "/api/v1/stats/stream", h.StatsStream)
	authed(http.MethodGet, "/api/v1/stats/history", h.StatsHistory)
	authed(http.MethodGet, "/api/v1/system/stats", h.SystemStatsStream)

	// Compose
	authed(http.MethodGet, "/api/v1/compose/projects", h.ListComposeProjects)
	authed(http.MethodGet, "/api/v1/compose/projects/{name}", h.GetComposeProject)
	authed(http.MethodPut, "/api/v1/compose/projects/{name}/files/{filename}", h.SaveComposeFile)
	authed(http.MethodPut, "/api/v1/compose/projects/{name}/file", h.SaveComposeDefaultFile)
	authed(http.MethodPost, "/api/v1/compose/projects/{name}/up", h.ComposeUp)
	authed(http.MethodPost, "/api/v1/compose/projects/{name}/down", h.ComposeDown)
	authed(http.MethodGet, "/api/v1/compose/projects/{name}/ps", h.ComposePs)

	// 设置
	authed(http.MethodGet, "/api/v1/settings", h.GetSettings)
	authed(http.MethodPut, "/api/v1/settings", h.UpdateSettings)

	// 前端静态资源（含 SPA 回退与 API 404）
	r.Handle("/*", newStaticHandler(d.WebDir))

	return r
}
