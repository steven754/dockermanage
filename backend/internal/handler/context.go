package handler

import (
	"net/http"

	"github.com/steven754/dockermanage/backend/internal/httpx"
)

// UserIDFromContext 从请求上下文读取当前用户 ID
func UserIDFromContext(r *http.Request) (int64, bool) {
	return httpx.UserIDFromContext(r.Context())
}
