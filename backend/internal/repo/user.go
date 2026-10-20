package repo

import (
	"database/sql"
	"errors"
	"fmt"
)

// ErrNotFound 记录不存在
var ErrNotFound = errors.New("记录不存在")

// User 用户记录
type User struct {
	ID                 int64
	Username           string
	PasswordHash       string
	MustChangePassword bool
	// TokenVersion 令牌版本，改密后自增；与 JWT 中的版本不一致的令牌视为已失效
	TokenVersion int
	CreatedAt    int64
	LastLoginAt  sql.NullInt64
}

// UserRepo 用户仓储
type UserRepo struct {
	db *sql.DB
}

// Count 返回用户总数
func (r *UserRepo) Count() (int, error) {
	var n int
	err := r.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("统计用户数失败: %w", err)
	}
	return n, nil
}

// Create 创建用户
func (r *UserRepo) Create(username, passwordHash string, mustChange bool) (*User, error) {
	res, err := r.db.Exec(
		"INSERT INTO users (username, password_hash, must_change_password, created_at) VALUES (?, ?, ?, ?)",
		username, passwordHash, boolToInt(mustChange), nowUnix(),
	)
	if err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("获取用户 ID 失败: %w", err)
	}
	return &User{
		ID:                 id,
		Username:           username,
		PasswordHash:       passwordHash,
		MustChangePassword: mustChange,
		CreatedAt:          nowUnix(),
	}, nil
}

// GetByUsername 按用户名查询
func (r *UserRepo) GetByUsername(username string) (*User, error) {
	return r.getOne("SELECT id, username, password_hash, must_change_password, token_version, created_at, last_login_at FROM users WHERE username = ?", username)
}

// GetByID 按 ID 查询
func (r *UserRepo) GetByID(id int64) (*User, error) {
	return r.getOne("SELECT id, username, password_hash, must_change_password, token_version, created_at, last_login_at FROM users WHERE id = ?", id)
}

func (r *UserRepo) getOne(query string, args ...any) (*User, error) {
	u := &User{}
	var lastLogin sql.NullInt64
	err := r.db.QueryRow(query, args...).Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.MustChangePassword, &u.TokenVersion, &u.CreatedAt, &lastLogin,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	u.LastLoginAt = lastLogin
	return u, nil
}

// UpdatePassword 更新密码哈希、清除强制改密标记并自增令牌版本。
//
// 返回自增后的令牌版本：改密前签发的令牌因版本不匹配而立即失效。
func (r *UserRepo) UpdatePassword(id int64, passwordHash string) (int, error) {
	var version int
	err := r.db.QueryRow(
		"UPDATE users SET password_hash = ?, must_change_password = 0, token_version = token_version + 1 WHERE id = ? RETURNING token_version",
		passwordHash, id,
	).Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("更新密码失败: %w", err)
	}
	return version, nil
}

// UpdateLastLogin 更新最后登录时间
func (r *UserRepo) UpdateLastLogin(id int64) error {
	_, err := r.db.Exec("UPDATE users SET last_login_at = ? WHERE id = ?", nowUnix(), id)
	if err != nil {
		return fmt.Errorf("更新登录时间失败: %w", err)
	}
	return nil
}

// boolToInt 布尔转整型
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
