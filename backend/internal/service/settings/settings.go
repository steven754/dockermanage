// Package settings 管理面板设置的持久化。
package settings

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/steven754/dockermanage/backend/internal/model"
	"github.com/steven754/dockermanage/backend/internal/repo"
)

// settingsKey 设置在 settings 表中的键名
const settingsKey = "settings"

// Service 设置服务
type Service struct {
	repo *repo.SettingRepo

	mu      sync.RWMutex
	current model.Settings
}

// New 创建设置服务并从数据库加载初始值
func New(settingRepo *repo.SettingRepo) *Service {
	s := &Service{repo: settingRepo, current: model.DefaultSettings()}
	s.load()
	return s
}

// load 从数据库加载设置，失败时保留默认值
func (s *Service) load() {
	kv, err := s.repo.Get(settingsKey)
	if err != nil {
		if err != repo.ErrNotFound {
			slog.Error("读取设置失败，使用默认值", "error", err)
		}
		return
	}
	var loaded model.Settings
	if err := json.Unmarshal([]byte(kv.Value), &loaded); err != nil {
		slog.Error("解析设置失败，使用默认值", "error", err)
		return
	}
	// 补齐缺失字段
	defaults := model.DefaultSettings()
	if loaded.Theme == "" {
		loaded.Theme = defaults.Theme
	}
	if loaded.SampleIntervalSeconds <= 0 {
		loaded.SampleIntervalSeconds = defaults.SampleIntervalSeconds
	}
	if loaded.RetentionDays <= 0 {
		loaded.RetentionDays = defaults.RetentionDays
	}
	if loaded.DefaultPageSize <= 0 {
		loaded.DefaultPageSize = defaults.DefaultPageSize
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.current = loaded
}

// Get 返回当前设置
func (s *Service) Get() model.Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current
}

// Update 保存新设置
func (s *Service) Update(next model.Settings) error {
	defaults := model.DefaultSettings()
	if next.Theme == "" {
		next.Theme = defaults.Theme
	}
	if next.SampleIntervalSeconds <= 0 {
		next.SampleIntervalSeconds = defaults.SampleIntervalSeconds
	}
	if next.RetentionDays <= 0 {
		next.RetentionDays = defaults.RetentionDays
	}
	if next.DefaultPageSize <= 0 {
		next.DefaultPageSize = defaults.DefaultPageSize
	}

	data, err := json.Marshal(next)
	if err != nil {
		return fmt.Errorf("序列化设置失败: %w", err)
	}
	if err := s.repo.Set(settingsKey, string(data)); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.current = next
	return nil
}
