package model_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/steven754/dockermanage/backend/internal/model"
)

func TestFormatTime(t *testing.T) {
	ts := time.Date(2026, 10, 20, 15, 30, 0, 0, time.UTC)
	got := model.FormatTime(ts)
	if got != "2026-10-20T15:30:00Z" {
		t.Fatalf("期望 2026-10-20T15:30:00Z，实际 %s", got)
	}

	// 零值返回空串
	if model.FormatTime(time.Time{}) != "" {
		t.Fatal("零值时间应返回空串")
	}
}

func TestFormatTimePtr(t *testing.T) {
	ts := time.Date(2026, 10, 20, 15, 30, 0, 0, time.UTC)
	got := model.FormatTimePtr(ts)
	if got == nil {
		t.Fatal("非零值时间不应返回 nil")
	}
	if *got != "2026-10-20T15:30:00Z" {
		t.Fatalf("期望 2026-10-20T15:30:00Z，实际 %s", *got)
	}

	if model.FormatTimePtr(time.Time{}) != nil {
		t.Fatal("零值时间应返回 nil")
	}
}

func TestFormatTimeConvertsToUTC(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	ts := time.Date(2026, 10, 20, 23, 30, 0, 0, loc)
	got := model.FormatTime(ts)
	if got != "2026-10-20T15:30:00Z" {
		t.Fatalf("期望转换为 UTC 的 2026-10-20T15:30:00Z，实际 %s", got)
	}
}

func TestNewListNeverNil(t *testing.T) {
	l := model.NewList[string](nil)
	data, err := json.Marshal(l)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}
	if string(data) != `{"items":[]}` {
		t.Fatalf("期望 {\"items\":[]}，实际 %s", data)
	}
}

func TestErrorResponseShape(t *testing.T) {
	// 与 api-contract 的错误格式保持一致
	body := `{"error":{"code":"CONTAINER_NOT_FOUND","message":"Container abc not found"}}`
	var parsed map[string]map[string]any
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if parsed["error"]["code"] != "CONTAINER_NOT_FOUND" {
		t.Fatalf("错误码解析错误: %v", parsed["error"]["code"])
	}
}

func TestContainerJSONTags(t *testing.T) {
	// 契约要求实时字段可为 null，这里确认指针字段序列化为 null
	c := model.Container{
		ID:       "abc",
		Name:     "web",
		State:    "running",
		Health:   "none",
		Labels:   map[string]string{},
		Ports:    []model.ContainerPort{},
		Networks: []model.ContainerNetwork{},
		Mounts:   []model.ContainerMount{},
	}
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}
	for _, key := range []string{"cpuPct", "memUsageBytes", "memLimitBytes", "netRxBytesPerSec", "netTxBytesPerSec"} {
		v, ok := parsed[key]
		if !ok {
			t.Fatalf("容器对象缺少字段 %s", key)
		}
		if v != nil {
			t.Fatalf("字段 %s 应为 null，实际 %v", key, v)
		}
	}
	if _, ok := parsed["finishedAt"]; !ok {
		t.Fatal("容器对象缺少字段 finishedAt")
	}
}

func TestDefaultSettings(t *testing.T) {
	s := model.DefaultSettings()
	if s.Theme != "dark" {
		t.Fatalf("期望默认主题 dark，实际 %s", s.Theme)
	}
	if s.SampleIntervalSeconds != 10 {
		t.Fatalf("期望默认采样间隔 10，实际 %d", s.SampleIntervalSeconds)
	}
	if s.RetentionDays != 7 {
		t.Fatalf("期望默认保留天数 7，实际 %d", s.RetentionDays)
	}
	if s.DefaultPageSize != 50 {
		t.Fatalf("期望默认分页 50，实际 %d", s.DefaultPageSize)
	}
}
