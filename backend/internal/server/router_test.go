package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/steven754/dockermanage/backend/internal/db"
	"github.com/steven754/dockermanage/backend/internal/handler"
	"github.com/steven754/dockermanage/backend/internal/repo"
	"github.com/steven754/dockermanage/backend/internal/server"
	"github.com/steven754/dockermanage/backend/internal/service/auth"
)

// testEnv 测试环境
type testEnv struct {
	router   http.Handler
	password string
	username string
}

// newTestEnv 构造带鉴权的测试路由
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	sqlDB, err := db.Open("", true)
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := db.Migrate(sqlDB); err != nil {
		t.Fatalf("执行迁移失败: %v", err)
	}
	store := repo.New(sqlDB)

	secret := []byte("0123456789abcdef0123456789abcdef")
	tokens := auth.NewTokenManager(secret)
	authSvc := auth.New(store.Users, tokens, t.TempDir())

	password, err := authSvc.Bootstrap(t.Context())
	if err != nil {
		t.Fatalf("初始化用户失败: %v", err)
	}

	h := handler.New(handler.Deps{
		Auth:     authSvc,
		Settings: nil,
		Store:    store,
	})

	return &testEnv{
		router:   server.NewRouter(server.Deps{Handler: h, Tokens: tokens}),
		password: password,
		username: auth.DefaultUsername,
	}
}

// doRequest 发起请求
func (e *testEnv) doRequest(t *testing.T, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var reader *bytes.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("序列化请求体失败: %v", err)
		}
		reader = bytes.NewReader(data)
	} else {
		reader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

func TestHealthzAndVersionNeedNoAuth(t *testing.T) {
	env := newTestEnv(t)

	rec := env.doRequest(t, http.MethodGet, "/healthz", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("期望 /healthz 返回 200，实际 %d", rec.Code)
	}

	rec = env.doRequest(t, http.MethodGet, "/api/v1/version", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("期望 /api/v1/version 返回 200，实际 %d", rec.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("解析版本响应失败: %v", err)
	}
	if _, ok := payload["version"]; !ok {
		t.Fatalf("版本响应缺少 version 字段: %s", rec.Body.String())
	}
}

func TestProtectedRouteReturns401(t *testing.T) {
	env := newTestEnv(t)

	// 无 Authorization 头
	rec := env.doRequest(t, http.MethodGet, "/api/v1/containers", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("期望未带 token 返回 401，实际 %d", rec.Code)
	}
	assertErrorCode(t, rec, "UNAUTHORIZED")

	// 非法 token
	rec = env.doRequest(t, http.MethodGet, "/api/v1/containers", "invalid-token", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("期望非法 token 返回 401，实际 %d", rec.Code)
	}
	assertErrorCode(t, rec, "UNAUTHORIZED")
}

func TestLoginFlow(t *testing.T) {
	env := newTestEnv(t)

	// 错误密码
	rec := env.doRequest(t, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"username": env.username,
		"password": "wrong-password",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("期望错误密码返回 401，实际 %d", rec.Code)
	}
	assertErrorCode(t, rec, "UNAUTHORIZED")

	// 正确密码
	rec = env.doRequest(t, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"username": env.username,
		"password": env.password,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("期望登录返回 200，实际 %d，响应 %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		AccessToken        string `json:"accessToken"`
		RefreshToken       string `json:"refreshToken"`
		ExpiresIn          int64  `json:"expiresIn"`
		MustChangePassword bool   `json:"mustChangePassword"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析登录响应失败: %v", err)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatal("登录响应缺少 token")
	}
	if resp.ExpiresIn != 3600 {
		t.Fatalf("期望 expiresIn 3600，实际 %d", resp.ExpiresIn)
	}
	if !resp.MustChangePassword {
		t.Fatal("首次启动登录后 mustChangePassword 应为 true")
	}

	// 带 token 访问受保护接口
	rec = env.doRequest(t, http.MethodGet, "/api/v1/auth/me", resp.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("期望 /api/v1/auth/me 返回 200，实际 %d", rec.Code)
	}
	var me struct {
		ID                 int64  `json:"id"`
		Username           string `json:"username"`
		MustChangePassword bool   `json:"mustChangePassword"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &me); err != nil {
		t.Fatalf("解析用户信息失败: %v", err)
	}
	if me.Username != env.username {
		t.Fatalf("期望用户名 %s，实际 %s", env.username, me.Username)
	}
	if !me.MustChangePassword {
		t.Fatal("mustChangePassword 应为 true")
	}
}

func TestChangePasswordFlow(t *testing.T) {
	env := newTestEnv(t)

	rec := env.doRequest(t, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"username": env.username,
		"password": env.password,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("登录失败: %d", rec.Code)
	}
	var login struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &login); err != nil {
		t.Fatalf("解析登录响应失败: %v", err)
	}

	// 旧密码错误
	rec = env.doRequest(t, http.MethodPost, "/api/v1/auth/password", login.AccessToken, map[string]string{
		"oldPassword": "not-the-old-one",
		"newPassword": "brand-new-password",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("期望旧密码错误返回 401，实际 %d", rec.Code)
	}

	// 新密码过短
	rec = env.doRequest(t, http.MethodPost, "/api/v1/auth/password", login.AccessToken, map[string]string{
		"oldPassword": env.password,
		"newPassword": "short",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("期望弱密码返回 400，实际 %d", rec.Code)
	}

	// 正常改密
	rec = env.doRequest(t, http.MethodPost, "/api/v1/auth/password", login.AccessToken, map[string]string{
		"oldPassword": env.password,
		"newPassword": "brand-new-password",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("期望改密返回 200，实际 %d，响应 %s", rec.Code, rec.Body.String())
	}
	var okResp struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &okResp); err != nil {
		t.Fatalf("解析改密响应失败: %v", err)
	}
	if !okResp.OK {
		t.Fatal("改密响应 ok 应为 true")
	}
}

func TestLogout(t *testing.T) {
	env := newTestEnv(t)

	rec := env.doRequest(t, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"username": env.username,
		"password": env.password,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("登录失败: %d", rec.Code)
	}
	var login struct {
		AccessToken string `json:"accessToken"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &login)

	rec = env.doRequest(t, http.MethodPost, "/api/v1/auth/logout", login.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("期望登出返回 200，实际 %d", rec.Code)
	}
	var okResp struct {
		OK bool `json:"ok"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &okResp)
	if !okResp.OK {
		t.Fatal("登出响应 ok 应为 true")
	}
}

func TestLoginValidationError(t *testing.T) {
	env := newTestEnv(t)

	rec := env.doRequest(t, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"username": "",
		"password": "",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("期望空用户名密码返回 400，实际 %d", rec.Code)
	}
	assertErrorCode(t, rec, "VALIDATION_ERROR")
}

// assertErrorCode 断言错误码
func assertErrorCode(t *testing.T, rec *httptest.ResponseRecorder, want string) {
	t.Helper()

	var resp struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析错误响应失败: %v，响应 %s", err, rec.Body.String())
	}
	if resp.Error.Code != want {
		t.Fatalf("期望错误码 %s，实际 %s", want, resp.Error.Code)
	}
	if resp.Error.Message == "" {
		t.Fatal("错误信息不应为空")
	}
}

// TestWildcardRouteCapturesSlashes 验证镜像 / 网络删除接口能捕获带斜杠的引用
func TestWildcardRouteCapturesSlashes(t *testing.T) {
	r := chi.NewRouter()

	var got string
	r.With(func(next http.Handler) http.Handler { return next }).
		Method(http.MethodDelete, "/api/v1/images/*", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			got = chi.URLParam(req, "*")
		}))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/images/dmtest/hello:v1", nil)
	r.ServeHTTP(httptest.NewRecorder(), req)

	if got != "dmtest/hello:v1" {
		t.Fatalf("通配路由未捕获完整引用，实际 %q", got)
	}
}
