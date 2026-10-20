package repo

import (
	"database/sql"
	"errors"
	"fmt"
)

// SettingRepo 设置仓储（key-value，value 为 JSON）
type SettingRepo struct {
	db *sql.DB
}

// SettingKV 单条设置
type SettingKV struct {
	Key       string
	Value     string
	UpdatedAt int64
}

// Get 读取单个设置项，不存在返回 ErrNotFound
func (r *SettingRepo) Get(key string) (*SettingKV, error) {
	var s SettingKV
	err := r.db.QueryRow("SELECT key, value, updated_at FROM settings WHERE key = ?", key).Scan(&s.Key, &s.Value, &s.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("读取设置失败: %w", err)
	}
	return &s, nil
}

// Set 写入（覆盖）单个设置项
func (r *SettingRepo) Set(key, value string) error {
	_, err := r.db.Exec(
		"INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at",
		key, value, nowUnix(),
	)
	if err != nil {
		return fmt.Errorf("写入设置失败: %w", err)
	}
	return nil
}

// All 返回全部设置项
func (r *SettingRepo) All() ([]SettingKV, error) {
	rows, err := r.db.Query("SELECT key, value, updated_at FROM settings")
	if err != nil {
		return nil, fmt.Errorf("查询设置失败: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]SettingKV, 0)
	for rows.Next() {
		var s SettingKV
		if err := rows.Scan(&s.Key, &s.Value, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("扫描设置失败: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
