package db

import (
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"

	"github.com/steven754/dockermanage/backend/migrations"
)

// createMigrationsTable 建迁移记录表
const createMigrationsTable = `
CREATE TABLE IF NOT EXISTS migrations (
    version     TEXT PRIMARY KEY,
    applied_at  INTEGER NOT NULL
)`

// Migrate 执行所有未应用的迁移，按文件名顺序执行，幂等。
func Migrate(sqlDB *sql.DB) error {
	if _, err := sqlDB.Exec(createMigrationsTable); err != nil {
		return fmt.Errorf("创建 migrations 表失败: %w", err)
	}

	applied, err := appliedVersions(sqlDB)
	if err != nil {
		return err
	}

	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("读取迁移目录失败: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		if applied[name] {
			slog.Debug("迁移已跳过", "version", name)
			continue
		}
		body, err := migrations.FS.ReadFile(name)
		if err != nil {
			return fmt.Errorf("读取迁移文件失败 %s: %w", name, err)
		}
		tx, err := sqlDB.Begin()
		if err != nil {
			return fmt.Errorf("开启事务失败: %w", err)
		}
		if _, err := tx.Exec(string(body)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("执行迁移失败 %s: %w", name, err)
		}
		if _, err := tx.Exec("INSERT INTO migrations (version, applied_at) VALUES (?, ?)", name, nowUnix()); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("记录迁移版本失败 %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("提交迁移失败 %s: %w", name, err)
		}
		slog.Info("迁移已应用", "version", name)
	}
	return nil
}

// appliedVersions 查询已执行的迁移版本
func appliedVersions(sqlDB *sql.DB) (map[string]bool, error) {
	rows, err := sqlDB.Query("SELECT version FROM migrations")
	if err != nil {
		return nil, fmt.Errorf("查询迁移记录失败: %w", err)
	}
	defer func() { _ = rows.Close() }()

	applied := make(map[string]bool)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("扫描迁移记录失败: %w", err)
		}
		applied[v] = true
	}
	return applied, rows.Err()
}
