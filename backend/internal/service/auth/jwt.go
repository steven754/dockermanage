package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenType token 用途
type TokenType string

const (
	// TokenTypeAccess 访问令牌
	TokenTypeAccess TokenType = "access"
	// TokenTypeRefresh 刷新令牌
	TokenTypeRefresh TokenType = "refresh"
)

// 默认有效期
const (
	// AccessTokenTTL 访问令牌有效期
	AccessTokenTTL = time.Hour
	// RefreshTokenTTL 刷新令牌有效期
	RefreshTokenTTL = 7 * 24 * time.Hour
)

// ErrInvalidToken token 无效
var ErrInvalidToken = errors.New("token 无效")

// Claims JWT 载荷
type Claims struct {
	Username           string    `json:"username"`
	MustChangePassword bool      `json:"mcp"`
	TokenType          TokenType `json:"typ"`
	jwt.RegisteredClaims
}

// TokenManager JWT 签发与校验
type TokenManager struct {
	secret []byte
	now    func() time.Time
}

// NewTokenManager 创建 token 管理器
func NewTokenManager(secret []byte) *TokenManager {
	return &TokenManager{secret: secret, now: time.Now}
}

// WithClock 替换时间源（测试用）
func (m *TokenManager) WithClock(now func() time.Time) *TokenManager {
	m.now = now
	return m
}

// TokenPair 签发结果
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

// Issue 为用户签发 access / refresh 双 token
func (m *TokenManager) Issue(userID int64, username string, mustChange bool) (*TokenPair, error) {
	now := m.now()

	access, err := m.sign(userID, username, mustChange, TokenTypeAccess, now, now.Add(AccessTokenTTL))
	if err != nil {
		return nil, err
	}
	refresh, err := m.sign(userID, username, mustChange, TokenTypeRefresh, now, now.Add(RefreshTokenTTL))
	if err != nil {
		return nil, err
	}
	return &TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int64(AccessTokenTTL.Seconds()),
	}, nil
}

// sign 签发单个 token
func (m *TokenManager) sign(userID int64, username string, mustChange bool, typ TokenType, issuedAt, expiresAt time.Time) (string, error) {
	claims := Claims{
		Username:           username,
		MustChangePassword: mustChange,
		TokenType:          typ,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(issuedAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("签发 token 失败: %w", err)
	}
	return signed, nil
}

// Parse 解析并校验 token，返回载荷
func (m *TokenManager) Parse(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("非预期的签名算法: %v", t.Header["alg"])
		}
		return m.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithTimeFunc(m.now),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// ParseAccess 解析 access token
func (m *TokenManager) ParseAccess(tokenStr string) (*Claims, error) {
	claims, err := m.Parse(tokenStr)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != TokenTypeAccess {
		return nil, fmt.Errorf("%w: token 类型不正确", ErrInvalidToken)
	}
	return claims, nil
}

// ParseRefresh 解析 refresh token
func (m *TokenManager) ParseRefresh(tokenStr string) (*Claims, error) {
	claims, err := m.Parse(tokenStr)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != TokenTypeRefresh {
		return nil, fmt.Errorf("%w: token 类型不正确", ErrInvalidToken)
	}
	return claims, nil
}

// UserID 从 claims 中取出用户 ID
func (c *Claims) UserID() (int64, error) {
	var id int64
	if _, err := fmt.Sscanf(c.Subject, "%d", &id); err != nil {
		return 0, fmt.Errorf("解析用户 ID 失败: %w", err)
	}
	return id, nil
}
