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
	"github.com/steven754/dockermanage/backend/internal/service/auth"
)

// Authenticate JWT 鉴权中间件。
//
// token 来源优先级：
//  1. Authorization: Bearer <access_token>（契约要求，常规接口使用）
//  2. ?access_token=<access_token>（浏览器 EventSource / WebSocket 无法自定义
//     请求头时的兜底，前端约定用这个参数名）
//  3. ?token=<access_token>（同上的历史别名，保留兼容）
func Authenticate(tokens *auth.TokenManager) func(http.Handler) http.Handler {
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

			ctx := httpx.WithUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
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
	// EventSource / WebSocket 无法自定义请求头，只能走查询参数
	q := r.URL.Query()
	if v := strings.TrimSpace(q.Get("access_token")); v != "" {
		return v
	}
	return strings.TrimSpace(q.Get("token"))
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
func LimitLogin(rl *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := clientIP(r)
			if !rl.Allow(key) {
				slog.Warn("登录接口触发限流", "client", key)
				httpx.WriteError(w, r, http.StatusTooManyRequests, httpx.CodeUnauthorized, "尝试过于频繁，请稍后再试")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// clientIP 取客户端 IP
func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := strings.Split(fwd, ",")
		return strings.TrimSpace(parts[0])
	}
	if real := r.Header.Get("X-Real-IP"); real != "" {
		return strings.TrimSpace(real)
	}
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i > 0 {
		host = host[:i]
	}
	return host
}
