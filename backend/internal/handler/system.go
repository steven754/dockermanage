package handler

import (
	"net/http"

	"github.com/steven754/dockermanage/backend/internal/httpx"
	"github.com/steven754/dockermanage/backend/internal/version"
)

// Version 处理 GET /api/v1/version
func (h *Handler) Version(w http.ResponseWriter, r *http.Request) {
	logInput(r)
	httpx.WriteJSON(w, http.StatusOK, version.Get())
}

// Healthz 处理 GET /healthz
func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// Readyz 处理 GET /readyz
func (h *Handler) Readyz(w http.ResponseWriter, r *http.Request) {
	dockerOK := h.system.Ping(r.Context()) == nil
	dbOK := h.store != nil && h.store.DB().PingContext(r.Context()) == nil

	status := http.StatusOK
	if !dockerOK || !dbOK {
		status = http.StatusServiceUnavailable
	}
	httpx.WriteJSON(w, status, map[string]any{
		"ok":       dockerOK && dbOK,
		"docker":   dockerOK,
		"database": dbOK,
	})
}
