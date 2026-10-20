package auth_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/steven754/dockermanage/backend/internal/service/auth"
)

func TestGeneratePassword(t *testing.T) {
	pwd, err := auth.GeneratePassword(16)
	if err != nil {
		t.Fatalf("生成密码失败: %v", err)
	}
	if len(pwd) != 16 {
		t.Fatalf("期望密码长度 16，实际 %d", len(pwd))
	}
	if strings.ContainsAny(pwd, " \t\n") {
		t.Fatalf("密码不应包含空白字符: %q", pwd)
	}

	// 两次生成应不同
	other, err := auth.GeneratePassword(16)
	if err != nil {
		t.Fatalf("生成密码失败: %v", err)
	}
	if pwd == other {
		t.Fatal("两次生成的随机密码不应相同")
	}
}

func TestGeneratePasswordDefaultLength(t *testing.T) {
	pwd, err := auth.GeneratePassword(0)
	if err != nil {
		t.Fatalf("生成密码失败: %v", err)
	}
	if len(pwd) != auth.DefaultPasswordLength {
		t.Fatalf("期望默认长度 %d，实际 %d", auth.DefaultPasswordLength, len(pwd))
	}
}

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := auth.HashPassword("s3cret-pass")
	if err != nil {
		t.Fatalf("生成哈希失败: %v", err)
	}
	if hash == "s3cret-pass" {
		t.Fatal("哈希不应等于明文")
	}
	if !auth.CheckPassword(hash, "s3cret-pass") {
		t.Fatal("正确密码应校验通过")
	}
	if auth.CheckPassword(hash, "wrong-pass") {
		t.Fatal("错误密码不应校验通过")
	}
}

func TestTokenManagerIssueAndParse(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef")
	tm := auth.NewTokenManager(secret)

	pair, err := tm.Issue(1, "admin", true)
	if err != nil {
		t.Fatalf("签发 token 失败: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("access / refresh token 不应为空")
	}
	if pair.ExpiresIn != int64(auth.AccessTokenTTL.Seconds()) {
		t.Fatalf("期望 expiresIn %d，实际 %d", int64(auth.AccessTokenTTL.Seconds()), pair.ExpiresIn)
	}

	claims, err := tm.ParseAccess(pair.AccessToken)
	if err != nil {
		t.Fatalf("解析 access token 失败: %v", err)
	}
	if claims.Username != "admin" {
		t.Fatalf("期望用户名 admin，实际 %s", claims.Username)
	}
	if !claims.MustChangePassword {
		t.Fatal("mustChangePassword 应为 true")
	}
	uid, err := claims.UserID()
	if err != nil {
		t.Fatalf("解析用户 ID 失败: %v", err)
	}
	if uid != 1 {
		t.Fatalf("期望用户 ID 1，实际 %d", uid)
	}

	// refresh token 不能当作 access token 使用
	if _, err := tm.ParseAccess(pair.RefreshToken); err == nil {
		t.Fatal("refresh token 不应通过 access token 校验")
	}
	if _, err := tm.ParseRefresh(pair.RefreshToken); err != nil {
		t.Fatalf("解析 refresh token 失败: %v", err)
	}
}

func TestTokenManagerRejectsWrongSecret(t *testing.T) {
	tm := auth.NewTokenManager([]byte("secret-one-secret-one-secret-one-"))
	pair, err := tm.Issue(1, "admin", false)
	if err != nil {
		t.Fatalf("签发 token 失败: %v", err)
	}

	other := auth.NewTokenManager([]byte("secret-two-secret-two-secret-two-"))
	if _, err := other.Parse(pair.AccessToken); err == nil {
		t.Fatal("使用其他密钥签发的 token 不应通过校验")
	} else if !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("期望 ErrInvalidToken，实际 %v", err)
	}
}

func TestTokenManagerExpiredToken(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef")
	base := time.Now()
	tm := auth.NewTokenManager(secret).WithClock(func() time.Time { return base })

	pair, err := tm.Issue(1, "admin", false)
	if err != nil {
		t.Fatalf("签发 token 失败: %v", err)
	}

	// 把时钟拨到过期之后
	tm.WithClock(func() time.Time { return base.Add(auth.AccessTokenTTL + time.Minute) })
	if _, err := tm.ParseAccess(pair.AccessToken); err == nil {
		t.Fatal("过期 token 不应通过校验")
	}
}

func TestTokenManagerRejectsGarbage(t *testing.T) {
	tm := auth.NewTokenManager([]byte("0123456789abcdef0123456789abcdef"))
	if _, err := tm.ParseAccess("not-a-token"); err == nil {
		t.Fatal("非法 token 不应通过校验")
	}
}
