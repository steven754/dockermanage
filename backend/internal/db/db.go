// Package db 负责 SQLite 连接与迁移执行。
package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // 纯 Go SQLite 驱动，无需 CGO
)

// DBName 数据库文件名
const DBName = "dockermanage.db"

// openRetryTimes 数据库首次打不开时的重试次数
const openRetryTimes = 10

// openRetryInterval 重试间隔
const openRetryInterval = 300 * time.Millisecond

// Open 打开（必要时创建）SQLite 数据库，启用 WAL 模式与外键约束。
//
// memory=true 时使用内存库（测试用）。
//
// 容器化部署时数据目录是 bind mount，容器进程启动的瞬间挂载点可能还没完全就绪，
// 此时打开会得到 SQLITE_CANTOPEN（驱动会把 errno 14 误报成 "out of memory"）。
// 因此这里做有限次重试，避免因一次瞬时失败导致容器退出重启。
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

	var pingErr error
	for i := 0; i < openRetryTimes; i++ {
		if pingErr = sqlDB.Ping(); pingErr == nil {
			return sqlDB, nil
		}
		if i < openRetryTimes-1 {
			slog.Warn("数据库暂时不可用，稍后重试", "attempt", i+1, "error", pingErr)
			time.Sleep(openRetryInterval)
		}
	}
	_ = sqlDB.Close()
	return nil, fmt.Errorf("数据库连通性检查失败（重试 %d 次后仍不可用，数据目录 %s）: %w", openRetryTimes, dataDir, pingErr)
}
