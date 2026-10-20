// Package httpx 提供 HTTP 层的通用响应与编解码辅助。
package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/steven754/dockermanage/backend/internal/model"
)

// ErrorBody 错误响应体
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse 统一错误响应
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// 错误码常量
const (
	CodeUnauthorized     = "UNAUTHORIZED"
	CodeForbidden        = "FORBIDDEN"
	CodeNotFound         = "NOT_FOUND"
	CodeConflict         = "CONFLICT"
	CodeValidation       = "VALIDATION_ERROR"
	CodeDockerAPIError   = "DOCKER_API_ERROR"
	CodeInternal         = "INTERNAL_ERROR"
	CodeContainerMissing = "CONTAINER_NOT_FOUND"
	CodeImageMissing     = "IMAGE_NOT_FOUND"
	// CodeMustChangePassword 尚未修改初始密码，除改密相关接口外一律拒绝
	CodeMustChangePassword = "MUST_CHANGE_PASSWORD"
)

// WriteJSON 写出 JSON 响应
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("写出 JSON 响应失败", "error", err)
	}
}

// WriteError 写出错误响应
//
// 业务错误用 Warn 级别记录，内部错误用 Error 级别记录。
func WriteError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	if status >= http.StatusInternalServerError {
		slog.Error("请求处理失败", "method", r.Method, "path", r.URL.Path, "code", code, "message", message)
	} else {
		slog.Warn("请求被拒绝", "method", r.Method, "path", r.URL.Path, "code", code, "message", message)
	}
	WriteJSON(w, status, ErrorResponse{Error: ErrorBody{Code: code, Message: message}})
}

// DecodeJSON 解析请求体
func DecodeJSON(r *http.Request, target any) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if err != nil {
		return errors.New("读取请求体失败")
	}
	if len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, target); err != nil {
		return errors.New("请求体不是合法 JSON")
	}
	return nil
}

// OK 写出 ok=true 响应
func OK(w http.ResponseWriter) {
	WriteJSON(w, http.StatusOK, model.Ok{OK: true})
}

// List 写出列表响应
func List[T any](w http.ResponseWriter, items []T) {
	WriteJSON(w, http.StatusOK, model.NewList(items))
}
