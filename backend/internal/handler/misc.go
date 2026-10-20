package handler

import (
	"net/http"

	"github.com/steven754/dockermanage/backend/internal/httpx"
	"github.com/steven754/dockermanage/backend/internal/model"
)

// ListEvents 处理 GET /api/v1/events
func (h *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {
	logInput(r)

	limit, _ := parseLimitOffset(r, 100)
	events, err := h.events.List(r.Context(), limit, r.URL.Query().Get("since"), r.URL.Query().Get("type"))
	if err != nil {
		writeInternalError(w, r, "查询事件失败")
		return
	}
	httpx.List(w, events)
}

// StatsHistory 处理 GET /api/v1/stats/history
func (h *Handler) StatsHistory(w http.ResponseWriter, r *http.Request) {
	logInput(r)

	rangeKey := r.URL.Query().Get("range")
	if rangeKey == "" {
		rangeKey = "1h"
	}
	containerID := r.URL.Query().Get("containerId")

	var (
		history *model.StatsHistory
		err     error
	)
	if containerID != "" {
		history, err = h.sampler.ContainerHistory(r.Context(), rangeKey, containerID)
	} else {
		history, err = h.sampler.History(r.Context(), rangeKey)
	}
	if err != nil {
		writeValidationError(w, r, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, history)
}

// GetSettings 处理 GET /api/v1/settings
func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	logInput(r)
	httpx.WriteJSON(w, http.StatusOK, h.settings.Get())
}

// UpdateSettings 处理 PUT /api/v1/settings
func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	logInput(r)

	var next model.Settings
	if err := httpx.DecodeJSON(r, &next); err != nil {
		writeValidationError(w, r, err.Error())
		return
	}
	switch next.Theme {
	case "", "dark", "light", "system":
	default:
		writeValidationError(w, r, "theme 只能是 dark / light / system")
		return
	}

	if err := h.settings.Update(next); err != nil {
		writeInternalError(w, r, "保存设置失败")
		return
	}
	httpx.OK(w)
}
