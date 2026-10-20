// Package server 组装 HTTP 路由与中间件。
package server

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/steven754/dockermanage/backend/internal/httpx"
	"github.com/steven754/dockermanage/backend/internal/repo"
	"github.com/steven754/dockermanage/backend/internal/service/auth"
)

// Authenticate JWT 鉴权中间件。
//
// token 来源优先级：
//  1. Authorization: Bearer <access_token>（契约要求，所有接口通用）
//  2. ?access_token=<access_token> / ?token=<access_token>
//     仅限流式端点：浏览器 EventSource / WebSocket 无法自定义请求头。
//     其余接口不接受查询参数携带令牌，避免令牌落入代理访问日志与浏览器历史。
//
// 除签名校验外，本中间件还承担两项安全职责：
//  1. 令牌版本校验：改密后用户令牌版本自增，改密前签发的令牌立即失效
//  2. 强制改密拦截：初始密码未修改时，除改密相关接口外一律拒绝，
//     避免「拿到初始口令就能直接调 exec / compose up」的安全约束只存在于前端
func Authenticate(tokens *auth.TokenManager, users *repo.UserRepo) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractToken(r)
			if tokenStr == "" {
				httpx.WriteError(w, r, http.StatusUnauthorized, httpx.CodeUnauthorized, "缺少访问令牌")
				return
			}

			claims, err := tokens.ParseAccess(tokenStr)
			if err != nil {
				httpx.WriteError(w, r, http.StatusUnauthorized, httpx.CodeUnauthorized, "访问令牌无效或已过期")
				return
			}
			userID, err := claims.UserID()
			if err != nil {
				httpx.WriteError(w, r, http.StatusUnauthorized, httpx.CodeUnauthorized, "访问令牌载荷无效")
				return
			}

			user, err := users.GetByID(userID)
			if err != nil {
				httpx.WriteError(w, r, http.StatusUnauthorized, httpx.CodeUnauthorized, "用户不存在或已被删除")
				return
			}

			// 令牌版本：改密后自增，旧令牌立即失效
			if claims.TokenVersion != user.TokenVersion {
				httpx.WriteError(w, r, http.StatusUnauthorized, httpx.CodeUnauthorized, "令牌已失效，请重新登录")
				return
			}

			// 强制改密：服务端强制，不依赖前端跳转
			if user.MustChangePassword && !isPasswordChangePath(r.URL.Path) {
				httpx.WriteError(w, r, http.StatusForbidden, httpx.CodeMustChangePassword,
					"初始密码尚未修改，请先修改密码")
				return
			}

			ctx := httpx.WithUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// isPasswordChangePath 判断该路径是否属于「改密流程」允许放行的接口。
//
// 改密前仅放行这三个接口：查询自身信息、修改密码、登出。
func isPasswordChangePath(path string) bool {
	switch path {
	case "/api/v1/auth/me", "/api/v1/auth/password", "/api/v1/auth/logout":
		return true
	}
	return false
}

// streamingPaths 流式端点：浏览器无法为这些请求设置请求头，
// 因此允许通过查询参数携带令牌。
var streamingPaths = []string{
	"/api/v1/stats/stream",
	"/api/v1/events/stream",
	"/api/v1/system/stats",
}

// isStreamingPath 判断是否为流式端点。
func isStreamingPath(path string) bool {
	for _, p := range streamingPaths {
		if path == p {
			return true
		}
	}
	// 容器下的流式子路径：/api/v1/containers/{id}/logs 与 /exec
	if strings.HasPrefix(path, "/api/v1/containers/") {
		rest := strings.TrimPrefix(path, "/api/v1/containers/")
		return strings.HasSuffix(rest, "/logs") || strings.HasSuffix(rest, "/exec")
	}
	return false
}

// extractToken 从请求中提取访问令牌
func extractToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if header != "" {
		parts := strings.SplitN(header, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}
	// EventSource / WebSocket 无法自定义请求头，只能走查询参数；
	// 限流在流式端点之外，避免令牌被代理访问日志记录
	if isStreamingPath(r.URL.Path) {
		q := r.URL.Query()
		if v := strings.TrimSpace(q.Get("access_token")); v != "" {
			return v
		}
		return strings.TrimSpace(q.Get("token"))
	}
	return ""
}

// RequestLogger 请求日志中间件
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		// SSE / WebSocket 是长连接，记录建立时刻即可
		if isStreaming(r.URL.Path) {
			slog.Info("请求完成",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"costMs", time.Since(start).Milliseconds())
			return
		}
		slog.Info("请求完成",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"costMs", time.Since(start).Milliseconds())
	})
}

// isStreaming 判断是否为流式接口
func isStreaming(path string) bool {
	return strings.HasSuffix(path, "/stream") ||
		strings.HasSuffix(path, "/logs") ||
		strings.HasSuffix(path, "/exec")
}

// Recoverer panic 恢复中间件
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("请求处理发生 panic",
					"method", r.Method,
					"path", r.URL.Path,
					"panic", rec)
				httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, "服务端内部错误")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// statusRecorder 记录响应状态码
type statusRecorder struct {
	http.ResponseWriter
	status int
}

// WriteHeader 记录状态码
func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// Flush 支持流式响应
func (s *statusRecorder) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack 支持 WebSocket 升级
func (s *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := s.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("响应对象不支持连接劫持")
	}
	return h.Hijack()
}

// Unwrap 交还底层 ResponseWriter，便于 http.ResponseController 找到原始实现
func (s *statusRecorder) Unwrap() http.ResponseWriter {
	return s.ResponseWriter
}

// RateLimiter 简单的内存限流器
type RateLimiter struct {
	mu       sync.Mutex
	entries  map[string]*rateEntry
	limit    int
	interval time.Duration
}

// rateEntry 单个来源的限流记录
type rateEntry struct {
	count int
	start time.Time
}

// NewRateLimiter 创建限流器，interval 内最多允许 limit 次请求
func NewRateLimiter(limit int, interval time.Duration) *RateLimiter {
	return &RateLimiter{
		entries:  map[string]*rateEntry{},
		limit:    limit,
		interval: interval,
	}
}

// Allow 判断是否放行
func (rl *RateLimiter) Allow(key string) bool {
	now := time.Now()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, ok := rl.entries[key]
	if !ok || now.Sub(entry.start) >= rl.interval {
		rl.entries[key] = &rateEntry{count: 1, start: now}
		return true
	}
	if entry.count >= rl.limit {
		return false
	}
	entry.count++
	return true
}

// LimitLogin 登录接口限流中间件（SPEC §8.2：5 次/分钟）
func LimitLogin(rl *RateLimiter, trustProxy bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := clientIP(r, trustProxy)
			if !rl.Allow(key) {
				slog.Warn("登录接口触发限流", "client", key)
				httpx.WriteError(w, r, http.StatusTooManyRequests, httpx.CodeUnauthorized, "尝试过于频繁，请稍后再试")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// clientIP 取客户端 IP。
//
// 默认**不采信** X-Forwarded-For / X-Real-IP：直连场景下攻击者每次请求
// 都能带一个随机的 X-Forwarded-For，把限流桶按新 key 分裂，登录限流形同虚设。
// 只有明确部署在反向代理之后（PANEL_TRUST_PROXY=true）才采信这些头。
func clientIP(r *http.Request, trustProxy bool) string {
	host := remoteHost(r.RemoteAddr)

	if trustProxy {
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			parts := strings.Split(fwd, ",")
			if ip := strings.TrimSpace(parts[0]); ip != "" {
				return ip
			}
		}
		if real := strings.TrimSpace(r.Header.Get("X-Real-IP")); real != "" {
			return real
		}
	}
	return host
}

// remoteHost 从 RemoteAddr 中剥出纯 IP
func remoteHost(remoteAddr string) string {
	host := remoteAddr
	if i := strings.LastIndex(host, ":"); i > 0 {
		host = host[:i]
	}
	return strings.Trim(host, "[]")
}
