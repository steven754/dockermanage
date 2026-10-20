package handler

import (
	"errors"
	"net/http"

	"github.com/steven754/dockermanage/backend/internal/httpx"
	"github.com/steven754/dockermanage/backend/internal/model"
	"github.com/steven754/dockermanage/backend/internal/service/compose"
)

// ListComposeProjects 处理 GET /api/v1/compose/projects
func (h *Handler) ListComposeProjects(w http.ResponseWriter, r *http.Request) {
	logInput(r)

	projects, err := h.compose.ListProjects(r.Context())
	if err != nil {
		writeInternalError(w, r, "读取 compose 项目失败")
		return
	}
	httpx.List(w, projects)
}

// GetComposeProject 处理 GET /api/v1/compose/projects/:name
func (h *Handler) GetComposeProject(w http.ResponseWriter, r *http.Request) {
	name := chiURLParam(r, "name")
	logInput(r, "project", name)

	project, err := h.compose.GetProject(r.Context(), name)
	if err != nil {
		writeComposeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, project)
}

// SaveComposeFile 处理 PUT /api/v1/compose/projects/:name/files/:filename
func (h *Handler) SaveComposeFile(w http.ResponseWriter, r *http.Request) {
	name := chiURLParam(r, "name")
	fileName := chiURLParam(r, "filename")
	logInput(r, "project", name, "file", fileName)

	var req model.SaveComposeFileRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		writeValidationError(w, r, err.Error())
		return
	}

	size, err := h.compose.SaveFile(name, fileName, req.Content)
	if err != nil {
		writeComposeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, model.SaveComposeFileResponse{OK: true, SizeBytes: size})
}

// SaveComposeDefaultFile 处理 PUT /api/v1/compose/projects/:name/file
//
// 契约中的路径为 files/:filename，这里额外提供默认文件名（compose 文件名）的简写。
func (h *Handler) SaveComposeDefaultFile(w http.ResponseWriter, r *http.Request) {
	name := chiURLParam(r, "name")
	logInput(r, "project", name)

	var req model.SaveComposeFileRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		writeValidationError(w, r, err.Error())
		return
	}

	fileName := compose.DefaultComposeFileName
	size, err := h.compose.SaveFile(name, fileName, req.Content)
	if err != nil {
		writeComposeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, model.SaveComposeFileResponse{OK: true, SizeBytes: size})
}

// ComposeUp 处理 POST /api/v1/compose/projects/:name/up
func (h *Handler) ComposeUp(w http.ResponseWriter, r *http.Request) {
	name := chiURLParam(r, "name")
	logInput(r, "project", name)

	var req model.ComposeUpRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		writeValidationError(w, r, err.Error())
		return
	}

	started, err := h.compose.Up(r.Context(), name, req.Services)
	if err != nil {
		writeComposeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, model.ComposeUpResponse{OK: true, Started: started})
}

// ComposeDown 处理 POST /api/v1/compose/projects/:name/down
func (h *Handler) ComposeDown(w http.ResponseWriter, r *http.Request) {
	name := chiURLParam(r, "name")
	logInput(r, "project", name)

	var req model.ComposeDownRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		writeValidationError(w, r, err.Error())
		return
	}

	if err := h.compose.Down(r.Context(), name, req.RemoveVolumes); err != nil {
		writeComposeError(w, r, err)
		return
	}
	httpx.OK(w)
}

// ComposePs 处理 GET /api/v1/compose/projects/:name/ps
func (h *Handler) ComposePs(w http.ResponseWriter, r *http.Request) {
	name := chiURLParam(r, "name")
	logInput(r, "project", name)

	services, err := h.compose.Ps(r.Context(), name)
	if err != nil {
		writeComposeError(w, r, err)
		return
	}
	httpx.List(w, services)
}

// writeComposeError 写出 compose 相关错误
func writeComposeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, compose.ErrProjectNotFound):
		httpx.WriteError(w, r, http.StatusNotFound, httpx.CodeNotFound, "compose 项目不存在")
	case errors.Is(err, compose.ErrFileNotFound):
		httpx.WriteError(w, r, http.StatusNotFound, httpx.CodeNotFound, "未找到 compose 文件")
	case errors.Is(err, compose.ErrInvalidName):
		writeValidationError(w, r, "项目名或文件名非法")
	default:
		writeInternalError(w, r, err.Error())
	}
}
