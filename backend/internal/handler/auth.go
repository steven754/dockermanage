package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/steven754/dockermanage/backend/internal/httpx"
	"github.com/steven754/dockermanage/backend/internal/model"
	"github.com/steven754/dockermanage/backend/internal/service/auth"
)

// Login 处理 POST /api/v1/auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	logInput(r)

	var req model.LoginRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		writeValidationError(w, r, err.Error())
		return
	}
	if req.Username == "" || req.Password == "" {
		writeValidationError(w, r, "用户名与密码不能为空")
		return
	}

	resp, err := h.auth.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			httpx.WriteError(w, r, http.StatusUnauthorized, httpx.CodeUnauthorized, "用户名或密码错误")
			return
		}
		writeInternalError(w, r, "登录失败，请稍后重试")
		return
	}

	slog.Info("登录成功", "username", req.Username)
	httpx.WriteJSON(w, http.StatusOK, resp)
}

// ChangePassword 处理 POST /api/v1/auth/password
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	logInput(r)

	var req model.ChangePasswordRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		writeValidationError(w, r, err.Error())
		return
	}
	if req.OldPassword == "" || req.NewPassword == "" {
		writeValidationError(w, r, "旧密码与新密码不能为空")
		return
	}

	userID, ok := UserIDFromContext(r)
	if !ok {
		httpx.WriteError(w, r, http.StatusUnauthorized, httpx.CodeUnauthorized, "未登录")
		return
	}

	resp, err := h.auth.ChangePassword(r.Context(), userID, req.OldPassword, req.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidCredentials):
			httpx.WriteError(w, r, http.StatusUnauthorized, httpx.CodeUnauthorized, "旧密码不正确")
		case errors.Is(err, auth.ErrPasswordTooWeak):
			writeValidationError(w, r, err.Error())
		case errors.Is(err, auth.ErrUserNotFound):
			httpx.WriteError(w, r, http.StatusNotFound, httpx.CodeNotFound, "用户不存在")
		default:
			writeInternalError(w, r, "修改密码失败，请稍后重试")
		}
		return
	}

	slog.Info("密码已修改", "userId", userID)
	// 返回新令牌：改密前签发的令牌已失效，前端需替换本地令牌
	httpx.WriteJSON(w, http.StatusOK, resp)
}

// Logout 处理 POST /api/v1/auth/logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	logInput(r)
	httpx.OK(w)
}

// Me 处理 GET /api/v1/auth/me
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	logInput(r)

	userID, ok := UserIDFromContext(r)
	if !ok {
		httpx.WriteError(w, r, http.StatusUnauthorized, httpx.CodeUnauthorized, "未登录")
		return
	}
	me, err := h.auth.Me(r.Context(), userID)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			httpx.WriteError(w, r, http.StatusNotFound, httpx.CodeNotFound, "用户不存在")
			return
		}
		writeInternalError(w, r, "获取用户信息失败")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, me)
}

// parseLimitOffset 解析分页参数
func parseLimitOffset(r *http.Request, defaultLimit int) (int, int) {
	limit := defaultLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			limit = v
		}
	}
	offset := 0
	if raw := r.URL.Query().Get("offset"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v >= 0 {
			offset = v
		}
	}
	return limit, offset
}
