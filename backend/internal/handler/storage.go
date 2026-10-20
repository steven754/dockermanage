package handler

import (
	"net/http"

	"github.com/steven754/dockermanage/backend/internal/httpx"
)

// ListVolumes 处理 GET /api/v1/volumes
func (h *Handler) ListVolumes(w http.ResponseWriter, r *http.Request) {
	logInput(r)

	volumes, err := h.volumes.List(r.Context())
	if err != nil {
		writeDockerError(w, r, err)
		return
	}
	httpx.List(w, volumes)
}

// DeleteVolume 处理 DELETE /api/v1/volumes/:name
func (h *Handler) DeleteVolume(w http.ResponseWriter, r *http.Request) {
	name := chiURLParam(r, "name")
	force := queryBool(r, "force")
	logInput(r, "volume", name, "force", force)

	if err := h.volumes.Remove(r.Context(), name, force); err != nil {
		writeDockerError(w, r, err)
		return
	}
	httpx.OK(w)
}

// PruneVolumes 处理 POST /api/v1/volumes/prune
func (h *Handler) PruneVolumes(w http.ResponseWriter, r *http.Request) {
	logInput(r)

	result, err := h.volumes.Prune(r.Context())
	if err != nil {
		writeDockerError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

// ListNetworks 处理 GET /api/v1/networks
func (h *Handler) ListNetworks(w http.ResponseWriter, r *http.Request) {
	logInput(r)

	networks, err := h.networks.List(r.Context())
	if err != nil {
		writeDockerError(w, r, err)
		return
	}
	httpx.List(w, networks)
}

// DeleteNetwork 处理 DELETE /api/v1/networks/:id
func (h *Handler) DeleteNetwork(w http.ResponseWriter, r *http.Request) {
	id := chiURLParam(r, "id")
	logInput(r, "networkId", id)

	if err := h.networks.Remove(r.Context(), id); err != nil {
		writeDockerError(w, r, err)
		return
	}
	httpx.OK(w)
}
