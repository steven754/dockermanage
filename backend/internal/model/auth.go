package model

// LoginRequest 登录请求体
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse 登录响应体
type LoginResponse struct {
	AccessToken        string `json:"accessToken"`
	RefreshToken       string `json:"refreshToken"`
	ExpiresIn          int64  `json:"expiresIn"`
	MustChangePassword bool   `json:"mustChangePassword"`
	// DefaultPasswordInUse 为 true 表示当前仍在使用默认初始密码，前端据此提示尽快修改
	DefaultPasswordInUse bool `json:"defaultPasswordInUse"`
}

// ChangePasswordRequest 修改密码请求体
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

// MeResponse 当前用户信息
type MeResponse struct {
	ID                 int64  `json:"id"`
	Username           string `json:"username"`
	MustChangePassword bool   `json:"mustChangePassword"`
}
