package handler

import (
	"net/http"

	"github.com/steven754/dockermanage/backend/internal/httpx"
	"github.com/steven754/dockermanage/backend/internal/model"
)

// ListImages 处理 GET /api/v1/images
func (h *Handler) ListImages(w http.ResponseWriter, r *http.Request) {
	logInput(r)

	images, err := h.images.List(r.Context())
	if err != nil {
		writeDockerErrorWithCode(w, r, err, httpx.CodeImageMissing)
		return
	}
	httpx.List(w, images)
}

// PullImage 处理 POST /api/v1/images/pull
func (h *Handler) PullImage(w http.ResponseWriter, r *http.Request) {
	logInput(r)

	var req model.PullImageRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		writeValidationError(w, r, err.Error())
		return
	}
	if req.Image == "" {
		writeValidationError(w, r, "镜像名不能为空")
		return
	}

	imageID, err := h.images.Pull(r.Context(), req.Image)
	if err != nil {
		writeDockerErrorWithCode(w, r, err, httpx.CodeImageMissing)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, model.PullImageResponse{OK: true, ImageID: imageID})
}

// TagImage 处理 POST /api/v1/images/:id/tag
func (h *Handler) TagImage(w http.ResponseWriter, r *http.Request) {
	id := chiURLParam(r, "id")
	logInput(r, "imageId", id)

	var req model.TagImageRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		writeValidationError(w, r, err.Error())
		return
	}
	if req.Repo == "" || req.Tag == "" {
		writeValidationError(w, r, "repo 与 tag 不能为空")
		return
	}

	if err := h.images.Tag(r.Context(), id, req.Repo, req.Tag); err != nil {
		writeDockerErrorWithCode(w, r, err, httpx.CodeImageMissing)
		return
	}
	httpx.OK(w)
}

// DeleteImage 处理 DELETE /api/v1/images/:id
func (h *Handler) DeleteImage(w http.ResponseWriter, r *http.Request) {
	id := chiURLParam(r, "id")
	force := queryBool(r, "force")
	logInput(r, "imageId", id, "force", force)

	if err := h.images.Remove(r.Context(), id, force); err != nil {
		writeDockerErrorWithCode(w, r, err, httpx.CodeImageMissing)
		return
	}
	httpx.OK(w)
}

// PruneImages 处理 POST /api/v1/images/prune
func (h *Handler) PruneImages(w http.ResponseWriter, r *http.Request) {
	logInput(r)

	var req struct {
		Dangling bool `json:"dangling"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		writeValidationError(w, r, err.Error())
		return
	}

	result, err := h.images.Prune(r.Context(), req.Dangling)
	if err != nil {
		writeDockerErrorWithCode(w, r, err, httpx.CodeImageMissing)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}
