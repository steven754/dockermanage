// Package db 负责 SQLite 连接与迁移执行。
package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // 纯 Go SQLite 驱动，无需 CGO
)

// DBName 数据库文件名
const DBName = "dockermanage.db"

// Open 打开（必要时创建）SQLite 数据库，启用 WAL 模式与外键约束。
//
// memory=true 时使用内存库（测试用）。
func Open(dataDir string, memory bool) (*sql.DB, error) {
	dsn := ""
	if memory {
		// 每次调用使用独立库名，避免测试之间共享状态
		dsn = fmt.Sprintf("file:memdb_%d_%d?mode=memory&cache=shared", os.Getpid(), time.Now().UnixNano())
	} else {
		if err := os.MkdirAll(dataDir, 0o750); err != nil {
			return nil, fmt.Errorf("创建数据目录失败: %w", err)
		}
		path := filepath.Join(dataDir, DBName)
		dsn = "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	}

	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// SQLite 单写者模型，限制连接数避免写锁竞争
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("数据库连通性检查失败: %w", err)
	}
	return sqlDB, nil
}
