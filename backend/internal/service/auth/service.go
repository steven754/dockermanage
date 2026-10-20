package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"golang.org/x/crypto/bcrypt"

	"github.com/steven754/dockermanage/backend/internal/model"
	"github.com/steven754/dockermanage/backend/internal/repo"
)

// 文件名常量
const (
	// SecretFileName JWT 密钥文件名
	SecretFileName = ".jwt_secret"
	// PasswordFileName 初始密码文件名
	PasswordFileName = ".password"
	// DefaultUsername 默认用户名
	DefaultUsername = "admin"
	// MinPasswordLength 修改密码时的最小长度
	MinPasswordLength = 8
)

// 业务错误
var (
	// ErrInvalidCredentials 用户名或密码错误
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	// ErrUserNotFound 用户不存在
	ErrUserNotFound = errors.New("用户不存在")
	// ErrPasswordTooWeak 新密码强度不足
	ErrPasswordTooWeak = errors.New("新密码至少需要 8 个字符")
)

// Service 认证服务
type Service struct {
	users   *repo.UserRepo
	tokens  *TokenManager
	dataDir string
}

// New 创建认证服务
func New(users *repo.UserRepo, tokens *TokenManager, dataDir string) *Service {
	return &Service{users: users, tokens: tokens, dataDir: dataDir}
}

// LoadOrCreateSecret 读取 JWT 密钥，不存在则生成随机 32 字节并以 0600 权限写入。
func LoadOrCreateSecret(dataDir string) ([]byte, error) {
	path := filepath.Join(dataDir, SecretFileName)

	if data, err := os.ReadFile(path); err == nil {
		if len(data) == 0 {
			return nil, fmt.Errorf("JWT 密钥文件为空: %s", path)
		}
		return data, nil
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("读取 JWT 密钥失败: %w", err)
	}

	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("生成 JWT 密钥失败: %w", err)
	}
	if err := os.WriteFile(path, secret, 0o600); err != nil {
		return nil, fmt.Errorf("写入 JWT 密钥失败: %w", err)
	}
	slog.Info("已生成新的 JWT 密钥", "path", path)
	return secret, nil
}

// Bootstrap 首启流程：数据库无用户时生成初始密码。
//
// 返回初始密码（仅首启非空），密码同时写入 <data>/.password 且在日志中打印一次。
func (s *Service) Bootstrap(ctx context.Context) (string, error) {
	count, err := s.users.Count()
	if err != nil {
		return "", err
	}
	if count > 0 {
		return "", nil
	}

	password, err := GeneratePassword(DefaultPasswordLength)
	if err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("生成密码哈希失败: %w", err)
	}
	if _, err := s.users.Create(DefaultUsername, string(hash), true); err != nil {
		return "", err
	}

	// 明文密码落盘，方便容器化场景下从宿主机查看
	path := filepath.Join(s.dataDir, PasswordFileName)
	if err := os.WriteFile(path, []byte(password+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("写入初始密码文件失败: %w", err)
	}

	// 密码只在这一次打印
	slog.Warn("首次启动，已生成初始管理员密码，请登录后立即修改",
		"username", DefaultUsername,
		"password", password,
		"passwordFile", path)
	return password, nil
}

// Login 校验用户名密码并签发 token
func (s *Service) Login(ctx context.Context, username, password string) (*model.LoginResponse, error) {
	user, err := s.users.GetByUsername(username)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	pair, err := s.tokens.Issue(user.ID, user.Username, user.MustChangePassword)
	if err != nil {
		return nil, err
	}
	if err := s.users.UpdateLastLogin(user.ID); err != nil {
		slog.Warn("更新最后登录时间失败", "error", err)
	}

	return &model.LoginResponse{
		AccessToken:        pair.AccessToken,
		RefreshToken:       pair.RefreshToken,
		ExpiresIn:          pair.ExpiresIn,
		MustChangePassword: user.MustChangePassword,
	}, nil
}

// ChangePassword 修改密码
func (s *Service) ChangePassword(ctx context.Context, userID int64, oldPassword, newPassword string) error {
	user, err := s.users.GetByID(userID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return ErrInvalidCredentials
	}
	if len(newPassword) < MinPasswordLength {
		return ErrPasswordTooWeak
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("生成密码哈希失败: %w", err)
	}
	if err := s.users.UpdatePassword(userID, string(hash)); err != nil {
		return err
	}

	// 改密后移除初始密码文件
	path := filepath.Join(s.dataDir, PasswordFileName)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		slog.Warn("删除初始密码文件失败", "path", path, "error", err)
	}
	return nil
}

// Me 查询当前用户信息
func (s *Service) Me(ctx context.Context, userID int64) (*model.MeResponse, error) {
	user, err := s.users.GetByID(userID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &model.MeResponse{
		ID:                 user.ID,
		Username:           user.Username,
		MustChangePassword: user.MustChangePassword,
	}, nil
}

// HashPassword 计算密码哈希
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("生成密码哈希失败: %w", err)
	}
	return string(hash), nil
}

// CheckPassword 校验密码
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// TokenManager 暴露 token 管理器，供中间件校验 access token
func (s *Service) TokenManager() *TokenManager { return s.tokens }
