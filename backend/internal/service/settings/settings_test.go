package settings_test

import (
	"testing"

	"github.com/steven754/dockermanage/backend/internal/db"
	"github.com/steven754/dockermanage/backend/internal/model"
	"github.com/steven754/dockermanage/backend/internal/repo"
	"github.com/steven754/dockermanage/backend/internal/service/settings"
)

// newTestService 构造带内存库的设置服务
func newTestService(t *testing.T) (*settings.Service, *repo.SettingRepo) {
	t.Helper()

	sqlDB, err := db.Open("", true)
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := db.Migrate(sqlDB); err != nil {
		t.Fatalf("执行迁移失败: %v", err)
	}
	settingRepo := repo.New(sqlDB).Settings
	return settings.New(settingRepo), settingRepo
}

func TestDefaultSettingsOnEmptyStore(t *testing.T) {
	svc, _ := newTestService(t)

	got := svc.Get()
	if got != model.DefaultSettings() {
		t.Fatalf("期望默认设置，实际 %+v", got)
	}
}

func TestUpdateAndReload(t *testing.T) {
	svc, _ := newTestService(t)

	next := model.Settings{
		Theme:                 "light",
		RegistryMirror:        "https://docker.1ms.run",
		SampleIntervalSeconds: 30,
		RetentionDays:         3,
		DefaultPageSize:       100,
	}
	if err := svc.Update(next); err != nil {
		t.Fatalf("更新设置失败: %v", err)
	}

	got := svc.Get()
	if got.Theme != "light" || got.RegistryMirror != "https://docker.1ms.run" {
		t.Fatalf("设置未生效: %+v", got)
	}
	if got.SampleIntervalSeconds != 30 {
		t.Fatalf("期望采样间隔 30，实际 %d", got.SampleIntervalSeconds)
	}
}

func TestUpdateFillsMissingFields(t *testing.T) {
	svc, _ := newTestService(t)

	// 只传部分字段时应补齐默认值
	if err := svc.Update(model.Settings{Theme: "system"}); err != nil {
		t.Fatalf("更新设置失败: %v", err)
	}
	got := svc.Get()
	if got.Theme != "system" {
		t.Fatalf("期望主题 system，实际 %s", got.Theme)
	}
	if got.SampleIntervalSeconds != 10 {
		t.Fatalf("期望补齐采样间隔 10，实际 %d", got.SampleIntervalSeconds)
	}
	if got.RetentionDays != 7 {
		t.Fatalf("期望补齐保留天数 7，实际 %d", got.RetentionDays)
	}
	if got.DefaultPageSize != 50 {
		t.Fatalf("期望补齐分页 50，实际 %d", got.DefaultPageSize)
	}
}

func TestReloadFromStore(t *testing.T) {
	svc, settingRepo := newTestService(t)

	if err := svc.Update(model.Settings{Theme: "light", DefaultPageSize: 20}); err != nil {
		t.Fatalf("更新设置失败: %v", err)
	}

	// 重新构造服务应能从数据库读回
	reloaded := settings.New(settingRepo)
	got := reloaded.Get()
	if got.Theme != "light" {
		t.Fatalf("期望主题 light，实际 %s", got.Theme)
	}
	if got.DefaultPageSize != 20 {
		t.Fatalf("期望分页 20，实际 %d", got.DefaultPageSize)
	}
}
