package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/steven754/dockermanage/backend/internal/httpx"
	"github.com/steven754/dockermanage/backend/internal/service/docker"
)

// ListContainers 处理 GET /api/v1/containers
func (h *Handler) ListContainers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, offset := parseLimitOffset(r, 0)

	filter := docker.ListFilter{
		Status: splitCSV(q.Get("status")),
		Search: q.Get("search"),
		Label:  q.Get("label"),
		Limit:  limit,
		Offset: offset,
	}

	containers, err := h.containers.List(r.Context(), filter)
	if err != nil {
		writeDockerErrorWithCode(w, r, err, httpx.CodeContainerMissing)
		return
	}
	httpx.List(w, containers)
}

// GetContainer 处理 GET /api/v1/containers/:id
func (h *Handler) GetContainer(w http.ResponseWriter, r *http.Request) {
	id := chiURLParam(r, "id")
	logInput(r, "containerId", id)

	c, err := h.containers.Get(r.Context(), id)
	if err != nil {
		writeDockerErrorWithCode(w, r, err, httpx.CodeContainerMissing)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, c)
}

// InspectContainerJSON 处理 GET /api/v1/containers/:id/json
func (h *Handler) InspectContainerJSON(w http.ResponseWriter, r *http.Request) {
	id := chiURLParam(r, "id")
	logInput(r, "containerId", id)

	raw, err := h.containers.InspectRaw(r.Context(), id)
	if err != nil {
		writeDockerErrorWithCode(w, r, err, httpx.CodeContainerMissing)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(raw); err != nil {
		writeInternalError(w, r, "写出 inspect 结果失败")
	}
}

// ContainerAction 处理 POST /api/v1/containers/:id/{start|stop|restart|kill|pause|unpause}
func (h *Handler) ContainerAction(w http.ResponseWriter, r *http.Request) {
	id := chiURLParam(r, "id")
	action := chiURLParam(r, "action")
	logInput(r, "containerId", id, "action", action)

	var err error
	switch action {
	case "start":
		err = h.containers.Start(r.Context(), id)
	case "stop":
		err = h.containers.Stop(r.Context(), id)
	case "restart":
		err = h.containers.Restart(r.Context(), id)
	case "kill":
		err = h.containers.Kill(r.Context(), id)
	case "pause":
		err = h.containers.Pause(r.Context(), id)
	case "unpause":
		err = h.containers.Unpause(r.Context(), id)
	default:
		writeValidationError(w, r, "不支持的操作: "+action)
		return
	}
	if err != nil {
		writeDockerErrorWithCode(w, r, err, httpx.CodeContainerMissing)
		return
	}
	httpx.OK(w)
}

// DeleteContainer 处理 DELETE /api/v1/containers/:id
func (h *Handler) DeleteContainer(w http.ResponseWriter, r *http.Request) {
	id := chiURLParam(r, "id")
	force := queryBool(r, "force")
	logInput(r, "containerId", id, "force", force)

	if err := h.containers.Remove(r.Context(), id, force); err != nil {
		writeDockerErrorWithCode(w, r, err, httpx.CodeContainerMissing)
		return
	}
	httpx.OK(w)
}

// PruneContainers 处理 POST /api/v1/containers/prune
func (h *Handler) PruneContainers(w http.ResponseWriter, r *http.Request) {
	logInput(r)

	var req struct {
		All bool `json:"all"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		writeValidationError(w, r, err.Error())
		return
	}

	result, err := h.containers.Prune(r.Context(), req.All)
	if err != nil {
		writeDockerErrorWithCode(w, r, err, httpx.CodeContainerMissing)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

// UpdateContainer 处理 POST /api/v1/containers/:id/update
func (h *Handler) UpdateContainer(w http.ResponseWriter, r *http.Request) {
	id := chiURLParam(r, "id")
	logInput(r, "containerId", id)

	result, err := h.containers.Update(r.Context(), id)
	if err != nil {
		writeDockerErrorWithCode(w, r, err, httpx.CodeContainerMissing)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

// ContainerStats 处理 GET /api/v1/containers/:id/stats
func (h *Handler) ContainerStats(w http.ResponseWriter, r *http.Request) {
	id := chiURLParam(r, "id")
	logInput(r, "containerId", id)

	stat, err := h.containers.Stats(r.Context(), id)
	if err != nil {
		writeDockerErrorWithCode(w, r, err, httpx.CodeContainerMissing)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, stat)
}

// splitCSV 拆分逗号分隔的多值参数
func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// queryBool 解析布尔查询参数
func queryBool(r *http.Request, name string) bool {
	v := r.URL.Query().Get(name)
	if v == "" {
		return false
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false
	}
	return b
}
