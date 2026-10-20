package events_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	dockerevents "github.com/docker/docker/api/types/events"

	"github.com/steven754/dockermanage/backend/internal/db"
	"github.com/steven754/dockermanage/backend/internal/model"
	"github.com/steven754/dockermanage/backend/internal/repo"
	"github.com/steven754/dockermanage/backend/internal/service/events"
)

// fakeSource 假事件源
type fakeSource struct {
	messages chan dockerevents.Message
	errs     chan error
}

func (f *fakeSource) Events(ctx context.Context, opts dockerevents.ListOptions) (<-chan dockerevents.Message, <-chan error) {
	return f.messages, f.errs
}

// newTestService 构造带内存库的事件服务
func newTestService(t *testing.T) (*events.Service, *repo.Store) {
	t.Helper()

	sqlDB, err := db.Open("", true)
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := db.Migrate(sqlDB); err != nil {
		t.Fatalf("执行迁移失败: %v", err)
	}
	store := repo.New(sqlDB)

	source := &fakeSource{
		messages: make(chan dockerevents.Message),
		errs:     make(chan error, 1),
	}
	return events.New(source, store.Events, 7*24*time.Hour), store
}

func TestListEvents(t *testing.T) {
	svc, store := newTestService(t)

	now := time.Now().Unix()
	if _, err := store.Events.Insert(repo.EventRecord{
		TS: now, Type: "container", Action: "start", ActorID: "a1", ActorName: "web",
		ResourceKind: "container", ResourceID: "a1", ResourceName: "web",
		ExitCode: sql.NullInt64{Int64: 0, Valid: true},
		Details:  repo.MarshalDetails(map[string]any{"image": "nginx"}),
	}); err != nil {
		t.Fatalf("写入事件失败: %v", err)
	}
	if _, err := store.Events.Insert(repo.EventRecord{
		TS: now, Type: "image", Action: "pull", ResourceKind: "image",
	}); err != nil {
		t.Fatalf("写入事件失败: %v", err)
	}

	// 全部事件
	list, err := svc.List(t.Context(), 10, "", "")
	if err != nil {
		t.Fatalf("查询事件失败: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("期望 2 条事件，实际 %d", len(list))
	}
	// 两条事件时间戳相同，按 id 倒序，取出容器事件校验详情
	var containerEvent = list[1]
	if containerEvent.Type != "container" {
		containerEvent = list[0]
	}
	if containerEvent.Details["image"] != "nginx" {
		t.Fatalf("事件详情不正确: %v", containerEvent.Details)
	}
	if containerEvent.ExitCode == nil {
		t.Fatal("事件应包含退出码")
	}

	// 按类型过滤
	list, err = svc.List(t.Context(), 10, "", "container")
	if err != nil {
		t.Fatalf("按类型查询事件失败: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("期望 1 条 container 事件，实际 %d", len(list))
	}

	// limit 生效
	list, err = svc.List(t.Context(), 1, "", "")
	if err != nil {
		t.Fatalf("查询事件失败: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("期望 1 条事件，实际 %d", len(list))
	}

	// since 过滤
	list, err = svc.List(t.Context(), 10, "99999999999", "")
	if err != nil {
		t.Fatalf("按时间查询事件失败: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("期望 0 条事件，实际 %d", len(list))
	}

	// 非法 since
	if _, err := svc.List(t.Context(), 10, "bad-time", ""); err == nil {
		t.Fatal("非法 since 应返回错误")
	}
}

func TestCleanupRemovesOldEvents(t *testing.T) {
	svc, store := newTestService(t)

	// 写入一条很旧的事件
	if _, err := store.Events.Insert(repo.EventRecord{
		TS:   time.Now().Add(-30 * 24 * time.Hour).Unix(),
		Type: "container", Action: "die", ResourceKind: "container",
	}); err != nil {
		t.Fatalf("写入事件失败: %v", err)
	}

	svc.Cleanup(t.Context())

	list, err := svc.List(t.Context(), 10, "", "")
	if err != nil {
		t.Fatalf("查询事件失败: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("过期事件应被清理，剩余 %d 条", len(list))
	}
}

func TestBrokerReceivesEvents(t *testing.T) {
	svc, store := newTestService(t)

	ch, cancel := svc.Broker().Subscribe(4)
	defer cancel()

	if _, err := store.Events.Insert(repo.EventRecord{
		TS: time.Now().Unix(), Type: "container", Action: "start", ResourceKind: "container",
	}); err != nil {
		t.Fatalf("写入事件失败: %v", err)
	}

	// 广播由 handle 触发，这里直接验证订阅可用
	go func() {
		svc.Broker().Publish(model.Event{Type: "container", Action: "start"})
	}()

	select {
	case e := <-ch:
		if e.Action != "start" {
			t.Fatalf("期望动作 start，实际 %s", e.Action)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("未收到广播事件")
	}
}
