package handler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/steven754/dockermanage/backend/internal/service/docker"
)

// acceptWebSocket 完成 WebSocket 握手
func acceptWebSocket(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// 面板为个人自用场景，Origin 校验交由鉴权与反向代理承担
		InsecureSkipVerify: true,
	})
	if err != nil {
		return nil, err
	}
	conn.SetReadLimit(1 << 20)
	return conn, nil
}

// ContainerLogs 处理 WS /api/v1/containers/:id/logs
//
// 后端不剥离 ANSI 颜色转义，终端由前端渲染。
func (h *Handler) ContainerLogs(w http.ResponseWriter, r *http.Request) {
	id := chiURLParam(r, "id")
	logInput(r, "containerId", id)

	conn, err := acceptWebSocket(w, r)
	if err != nil {
		slog.Warn("容器日志 WebSocket 握手失败", "containerId", id, "error", err)
		return
	}
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	q := r.URL.Query()
	opts := docker.LogOptions{
		Tail:       q.Get("tail"),
		Timestamps: queryBoolDefault(q.Get("timestamps"), true),
		Follow:     queryBoolDefault(q.Get("follow"), true),
	}
	if opts.Tail == "" {
		opts.Tail = "200"
	}

	logs, err := h.containers.Logs(r.Context(), id, opts)
	if err != nil {
		writeWebSocketError(conn, err.Error())
		return
	}
	defer func() { _ = logs.Close() }()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// 读取客户端消息（按契约仅支持 resize，忽略即可），用于感知断开
	go func() {
		for {
			if _, _, err := conn.Read(ctx); err != nil {
				cancel()
				return
			}
		}
	}()

	dataCh := make(chan []byte, 16)
	errCh := make(chan error, 1)

	go func() {
		defer close(dataCh)
		buf := make([]byte, 32*1024)
		for {
			n, readErr := logs.Read(buf)
			if n > 0 {
				chunk := make([]byte, n)
				copy(chunk, buf[:n])
				select {
				case dataCh <- chunk:
				case <-ctx.Done():
					return
				}
			}
			if readErr != nil {
				if !errors.Is(readErr, io.EOF) {
					slog.Debug("容器日志流结束", "containerId", id, "error", readErr)
				}
				errCh <- readErr
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case chunk, ok := <-dataCh:
			if !ok {
				return
			}
			if err := conn.Write(ctx, websocket.MessageText, chunk); err != nil {
				return
			}
		case <-errCh:
			return
		}
	}
}

// ContainerExec 处理 WS /api/v1/containers/:id/exec
func (h *Handler) ContainerExec(w http.ResponseWriter, r *http.Request) {
	id := chiURLParam(r, "id")
	logInput(r, "containerId", id)

	conn, err := acceptWebSocket(w, r)
	if err != nil {
		slog.Warn("容器终端 WebSocket 握手失败", "containerId", id, "error", err)
		return
	}
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	q := r.URL.Query()
	cmd := q.Get("cmd")
	if cmd == "" {
		cmd = "/bin/sh"
	}
	tty := queryBoolDefault(q.Get("tty"), true)
	user := q.Get("user")

	session, err := h.containers.ExecCreate(r.Context(), id, docker.ExecOptions{
		Cmd:  []string{cmd},
		TTY:  tty,
		User: user,
	})
	if err != nil {
		writeWebSocketError(conn, err.Error())
		return
	}

	stream, err := h.containers.ExecAttach(r.Context(), session.ID)
	if err != nil {
		writeWebSocketError(conn, err.Error())
		return
	}
	defer func() { stream.Close() }()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	var wg sync.WaitGroup

	// 客户端 -> 容器
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			typ, data, err := conn.Read(ctx)
			if err != nil {
				cancel()
				return
			}
			if typ != websocket.MessageText {
				continue
			}
			if _, err := stream.Conn.Write(data); err != nil {
				cancel()
				return
			}
		}
	}()

	// 容器 -> 客户端
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 32*1024)
		for {
			n, err := stream.Reader.Read(buf)
			if n > 0 {
				if werr := conn.Write(ctx, websocket.MessageText, buf[:n]); werr != nil {
					cancel()
					return
				}
			}
			if err != nil {
				cancel()
				return
			}
		}
	}()

	wg.Wait()
}

// writeWebSocketError 通过 WebSocket 发送错误信息
func writeWebSocketError(conn *websocket.Conn, message string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = conn.Write(ctx, websocket.MessageText, []byte(message))
}

// queryBoolDefault 解析布尔参数，缺省或非法时返回默认值
func queryBoolDefault(raw string, def bool) bool {
	if raw == "" {
		return def
	}
	v, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		return def
	}
	return v
}
