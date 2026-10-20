package events_test

import (
	"context"
	"testing"
	"time"

	dockerevents "github.com/docker/docker/api/types/events"

	"github.com/steven754/dockermanage/backend/internal/db"
	"github.com/steven754/dockermanage/backend/internal/repo"
	"github.com/steven754/dockermanage/backend/internal/service/events"
)

// scriptedSource 按脚本投递事件的可控事件源
type scriptedSource struct {
	messages chan dockerevents.Message
	errs     chan error
}

func (f *scriptedSource) Events(ctx context.Context, opts dockerevents.ListOptions) (<-chan dockerevents.Message, <-chan error) {
	return f.messages, f.errs
}

// newScriptedService 构造带脚本事件源的服务
func newScriptedService(t *testing.T) (*events.Service, *scriptedSource, *repo.Store) {
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

	source := &scriptedSource{
		messages: make(chan dockerevents.Message, 8),
		errs:     make(chan error, 1),
	}
	return events.New(source, store.Events, 7*24*time.Hour), source, store
}

func TestRunConsumesAndPersistsEvents(t *testing.T) {
	svc, source, store := newScriptedService(t)

	ch, cancel := svc.Broker().Subscribe(8)
	defer cancel()

	ctx, cancelRun := context.WithCancel(context.Background())
	defer cancelRun()
	done := make(chan struct{})
	go func() {
		svc.Run(ctx)
		close(done)
	}()

	source.messages <- dockerevents.Message{
		Type:   dockerevents.Type("container"),
		Action: dockerevents.Action("start"),
		Time:   time.Now().Unix(),
		Actor: dockerevents.Actor{
			ID:         "abc123",
			Attributes: map[string]string{"name": "web"},
		},
	}

	select {
	case e := <-ch:
		if e.Action != "start" {
			t.Fatalf("期望动作 start，实际 %s", e.Action)
		}
		if e.ActorName != "web" {
			t.Fatalf("期望 actorName web，实际 %s", e.ActorName)
		}
		if e.ID <= 0 {
			t.Fatal("事件应带有数据库自增 ID")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("未收到事件广播")
	}

	// 事件应已落库
	list, err := store.Events.List(repo.ListEventsQuery{Limit: 10})
	if err != nil {
		t.Fatalf("查询事件失败: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("期望 1 条落库事件，实际 %d", len(list))
	}

	cancelRun()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("事件监听未退出")
	}
}

func TestRunDeduplicatesRapidEvents(t *testing.T) {
	svc, source, store := newScriptedService(t)

	ctx, cancelRun := context.WithCancel(context.Background())
	defer cancelRun()
	go svc.Run(ctx)

	msg := dockerevents.Message{
		Type:   dockerevents.Type("container"),
		Action: dockerevents.Action("start"),
		Time:   time.Now().Unix(),
		Actor:  dockerevents.Actor{ID: "dedup-id"},
	}
	source.messages <- msg
	source.messages <- msg

	time.Sleep(500 * time.Millisecond)

	list, err := store.Events.List(repo.ListEventsQuery{Limit: 10})
	if err != nil {
		t.Fatalf("查询事件失败: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("1 秒内的重复事件应被去重，实际落库 %d 条", len(list))
	}
}

func TestRunReconnectsAfterStreamError(t *testing.T) {
	svc, source, _ := newScriptedService(t)

	ctx, cancelRun := context.WithCancel(context.Background())
	defer cancelRun()
	go svc.Run(ctx)

	// 断开事件流，服务应重连
	source.errs <- context.DeadlineExceeded

	// 重连后仍能收到事件
	time.Sleep(200 * time.Millisecond)
	source.messages <- dockerevents.Message{
		Type:   dockerevents.Type("container"),
		Action: dockerevents.Action("die"),
		Time:   time.Now().Unix(),
		Actor:  dockerevents.Actor{ID: "after-reconnect"},
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		list, err := svc.List(context.Background(), 10, "", "")
		if err != nil {
			t.Fatalf("查询事件失败: %v", err)
		}
		if len(list) > 0 && list[0].Action == "die" {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("重连后未收到事件")
}
