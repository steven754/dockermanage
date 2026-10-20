package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/steven754/dockermanage/backend/internal/service/stats"
)

// keepaliveInterval SSE 心跳间隔
const keepaliveInterval = 15 * time.Second

// sseWriter SSE 响应写入器
type sseWriter struct {
	w       http.ResponseWriter
	flusher *http.ResponseController
}

// newSSEWriter 初始化 SSE 响应头
func newSSEWriter(w http.ResponseWriter) *sseWriter {
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	return &sseWriter{w: w, flusher: http.NewResponseController(w)}
}

// Send 发送一条 SSE 消息
func (s *sseWriter) Send(event string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化 SSE 消息失败: %w", err)
	}
	if event != "" {
		if _, err := fmt.Fprintf(s.w, "event: %s\n", event); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(s.w, "data: %s\n\n", data); err != nil {
		return err
	}
	return s.flush()
}

// Keepalive 发送心跳注释行
func (s *sseWriter) Keepalive() error {
	if _, err := fmt.Fprint(s.w, ": keepalive\n\n"); err != nil {
		return err
	}
	return s.flush()
}

// flush 刷新响应缓冲区
func (s *sseWriter) flush() error {
	if s.flusher != nil {
		return s.flusher.Flush()
	}
	return nil
}

// EventsStream 处理 GET /api/v1/events/stream（SSE）
func (h *Handler) EventsStream(w http.ResponseWriter, r *http.Request) {
	logInput(r)

	stream := newSSEWriter(w)
	ch, cancel := h.events.Broker().Subscribe(64)
	defer cancel()

	ticker := time.NewTicker(keepaliveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if err := stream.Keepalive(); err != nil {
				return
			}
		case event, ok := <-ch:
			if !ok {
				return
			}
			if err := stream.Send("event", event); err != nil {
				return
			}
		}
	}
}

// StatsStream 处理 GET /api/v1/stats/stream（SSE）
func (h *Handler) StatsStream(w http.ResponseWriter, r *http.Request) {
	logInput(r)

	stream := newSSEWriter(w)
	ch, cancel := h.sampler.Broker().Subscribe(256)
	defer cancel()

	ticker := time.NewTicker(keepaliveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if err := stream.Keepalive(); err != nil {
				return
			}
		case msg, ok := <-ch:
			if !ok {
				return
			}
			switch msg.Type {
			case stats.EventTypeSystem:
				if err := stream.Send("system", msg.System); err != nil {
					return
				}
			case stats.EventTypeContainer:
				if err := stream.Send("container", msg.Container); err != nil {
					return
				}
			}
		}
	}
}

// SystemStatsStream 处理 GET /api/v1/system/stats（SSE，仅系统级）
func (h *Handler) SystemStatsStream(w http.ResponseWriter, r *http.Request) {
	logInput(r)

	stream := newSSEWriter(w)
	ch, cancel := h.sampler.Broker().Subscribe(64)
	defer cancel()

	ticker := time.NewTicker(keepaliveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if err := stream.Keepalive(); err != nil {
				return
			}
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if msg.Type != stats.EventTypeSystem {
				continue
			}
			if err := stream.Send("system", msg.System); err != nil {
				return
			}
		}
	}
}

// logSSEError 记录 SSE 异常
func logSSEError(name string, err error) {
	slog.Debug("SSE 连接结束", "stream", name, "error", err)
}
