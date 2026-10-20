// Package repo 是数据访问层，隔离 SQL 与上层业务。
package repo

import (
	"database/sql"
	"time"
)

// Store 聚合各资源仓储
type Store struct {
	db *sql.DB

	Users          *UserRepo
	Settings       *SettingRepo
	SystemStats    *SystemStatsRepo
	ContainerStats *ContainerStatsRepo
	Events         *EventRepo
}

// New 创建 Store
func New(db *sql.DB) *Store {
	return &Store{
		db:             db,
		Users:          &UserRepo{db: db},
		Settings:       &SettingRepo{db: db},
		SystemStats:    &SystemStatsRepo{db: db},
		ContainerStats: &ContainerStatsRepo{db: db},
		Events:         &EventRepo{db: db},
	}
}

// DB 返回底层数据库句柄
func (s *Store) DB() *sql.DB { return s.db }

// nowUnix 返回当前 unix 秒
func nowUnix() int64 { return time.Now().Unix() }
