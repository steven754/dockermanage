package auth_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/steven754/dockermanage/backend/internal/db"
	"github.com/steven754/dockermanage/backend/internal/repo"
	"github.com/steven754/dockermanage/backend/internal/service/auth"
)

// newTestService 构造带内存库的认证服务
func newTestService(t *testing.T) (*auth.Service, *repo.Store, string) {
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
	dir := t.TempDir()

	return auth.New(store.Users, tokens, dir), store, dir
}

func TestBootstrapCreatesUserAndPasswordFile(t *testing.T) {
	svc, _, dir := newTestService(t)

	password, err := svc.Bootstrap(t.Context())
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}
	if password == "" {
		t.Fatal("首次启动应返回初始密码")
	}
	if len(password) != auth.DefaultPasswordLength {
		t.Fatalf("期望密码长度 %d，实际 %d", auth.DefaultPasswordLength, len(password))
	}

	// 明文密码文件应存在且权限为 0600
	path := filepath.Join(dir, auth.PasswordFileName)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("初始密码文件不存在: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("期望文件权限 0600，实际 %o", info.Mode().Perm())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取初始密码文件失败: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("初始密码文件不应为空")
	}

	// 重复调用不应再生成密码
	second, err := svc.Bootstrap(t.Context())
	if err != nil {
		t.Fatalf("重复初始化失败: %v", err)
	}
	if second != "" {
		t.Fatal("重复初始化不应再生成密码")
	}
}

func TestLogin(t *testing.T) {
	svc, _, _ := newTestService(t)

	password, err := svc.Bootstrap(t.Context())
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}

	// 错误密码
	if _, err := svc.Login(t.Context(), auth.DefaultUsername, "wrong"); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("期望 ErrInvalidCredentials，实际 %v", err)
	}

	// 不存在的用户
	if _, err := svc.Login(t.Context(), "nobody", password); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("期望 ErrInvalidCredentials，实际 %v", err)
	}

	// 正确密码
	resp, err := svc.Login(t.Context(), auth.DefaultUsername, password)
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatal("登录应返回双 token")
	}
	if !resp.MustChangePassword {
		t.Fatal("首启后 mustChangePassword 应为 true")
	}
	if resp.ExpiresIn != 3600 {
		t.Fatalf("期望 expiresIn 3600，实际 %d", resp.ExpiresIn)
	}
}

func TestChangePassword(t *testing.T) {
	svc, _, dir := newTestService(t)

	password, err := svc.Bootstrap(t.Context())
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}

	// 旧密码错误
	err = svc.ChangePassword(t.Context(), 1, "wrong", "new-strong-password")
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("期望 ErrInvalidCredentials，实际 %v", err)
	}

	// 新密码过短
	err = svc.ChangePassword(t.Context(), 1, password, "short")
	if !errors.Is(err, auth.ErrPasswordTooWeak) {
		t.Fatalf("期望 ErrPasswordTooWeak，实际 %v", err)
	}

	// 用户不存在
	err = svc.ChangePassword(t.Context(), 999, password, "new-strong-password")
	if !errors.Is(err, auth.ErrUserNotFound) {
		t.Fatalf("期望 ErrUserNotFound，实际 %v", err)
	}

	// 正常改密
	if err := svc.ChangePassword(t.Context(), 1, password, "new-strong-password"); err != nil {
		t.Fatalf("改密失败: %v", err)
	}

	// 旧密码不再可用
	if _, err := svc.Login(t.Context(), auth.DefaultUsername, password); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("旧密码应失效，实际 %v", err)
	}
	// 新密码可用
	resp, err := svc.Login(t.Context(), auth.DefaultUsername, "new-strong-password")
	if err != nil {
		t.Fatalf("新密码登录失败: %v", err)
	}
	if resp.MustChangePassword {
		t.Fatal("改密后 mustChangePassword 应为 false")
	}

	// 初始密码文件应被清理
	if _, err := os.Stat(filepath.Join(dir, auth.PasswordFileName)); !os.IsNotExist(err) {
		t.Fatal("改密后初始密码文件应被删除")
	}
}

func TestMe(t *testing.T) {
	svc, _, _ := newTestService(t)

	if _, err := svc.Bootstrap(t.Context()); err != nil {
		t.Fatalf("初始化失败: %v", err)
	}

	me, err := svc.Me(t.Context(), 1)
	if err != nil {
		t.Fatalf("查询用户信息失败: %v", err)
	}
	if me.Username != auth.DefaultUsername {
		t.Fatalf("期望用户名 %s，实际 %s", auth.DefaultUsername, me.Username)
	}
	if !me.MustChangePassword {
		t.Fatal("首启后 mustChangePassword 应为 true")
	}

	if _, err := svc.Me(t.Context(), 999); !errors.Is(err, auth.ErrUserNotFound) {
		t.Fatalf("期望 ErrUserNotFound，实际 %v", err)
	}
}

func TestLoadOrCreateSecret(t *testing.T) {
	dir := t.TempDir()

	secret, err := auth.LoadOrCreateSecret(dir)
	if err != nil {
		t.Fatalf("生成密钥失败: %v", err)
	}
	if len(secret) != 32 {
		t.Fatalf("期望密钥 32 字节，实际 %d", len(secret))
	}

	path := filepath.Join(dir, auth.SecretFileName)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("密钥文件不存在: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("期望密钥文件权限 0600，实际 %o", info.Mode().Perm())
	}

	// 再次读取应返回同一份密钥
	again, err := auth.LoadOrCreateSecret(dir)
	if err != nil {
		t.Fatalf("读取密钥失败: %v", err)
	}
	if string(again) != string(secret) {
		t.Fatal("重复读取应返回相同密钥")
	}
}
