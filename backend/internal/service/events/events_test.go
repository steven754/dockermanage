package events_test

import (
	"testing"

	dockerevents "github.com/docker/docker/api/types/events"

	"github.com/steven754/dockermanage/backend/internal/service/events"
)

func TestConvertContainerEvent(t *testing.T) {
	msg := dockerevents.Message{
		Type:   dockerevents.Type("container"),
		Action: dockerevents.Action("die"),
		Time:   1700000000,
		Actor: dockerevents.Actor{
			ID: "abc123",
			Attributes: map[string]string{
				"name":     "nginx-proxy",
				"image":    "nginx:1.27",
				"exitCode": "137",
			},
		},
	}

	e, ok := events.Convert(msg)
	if !ok {
		t.Fatal("容器事件应转换成功")
	}
	if e.Type != "container" {
		t.Fatalf("期望类型 container，实际 %s", e.Type)
	}
	if e.Action != "die" {
		t.Fatalf("期望动作 die，实际 %s", e.Action)
	}
	if e.ActorID != "abc123" {
		t.Fatalf("期望 actorId abc123，实际 %s", e.ActorID)
	}
	if e.ActorName != "nginx-proxy" {
		t.Fatalf("期望 actorName nginx-proxy，实际 %s", e.ActorName)
	}
	if e.ExitCode == nil || *e.ExitCode != 137 {
		t.Fatalf("期望退出码 137，实际 %v", e.ExitCode)
	}
	if e.Details["image"] != "nginx:1.27" {
		t.Fatalf("详情中缺少 image: %v", e.Details)
	}
}

func TestConvertImageEvent(t *testing.T) {
	msg := dockerevents.Message{
		Type:   dockerevents.Type("image"),
		Action: dockerevents.Action("pull"),
		Actor: dockerevents.Actor{
			ID: "sha256:abc",
			Attributes: map[string]string{
				"name": "nginx:1.27",
			},
		},
	}

	e, ok := events.Convert(msg)
	if !ok {
		t.Fatal("镜像事件应转换成功")
	}
	if e.Type != "image" || e.Action != "pull" {
		t.Fatalf("类型或动作不正确: %+v", e)
	}
	if e.ExitCode != nil {
		t.Fatal("镜像事件不应有退出码")
	}
}

func TestConvertRejectsIncompleteMessage(t *testing.T) {
	if _, ok := events.Convert(dockerevents.Message{Action: "start"}); ok {
		t.Fatal("缺少 type 的事件应被拒绝")
	}
	if _, ok := events.Convert(dockerevents.Message{Type: dockerevents.Type("container")}); ok {
		t.Fatal("缺少 action 的事件应被拒绝")
	}
}

func TestParseSince(t *testing.T) {
	// RFC3339
	ts, err := events.ParseSinceForTest("2026-10-20T15:30:00Z")
	if err != nil {
		t.Fatalf("解析 RFC3339 失败: %v", err)
	}
	if ts != 1792510200 {
		t.Fatalf("时间戳不正确: %d", ts)
	}

	// unix 秒
	ts, err = events.ParseSinceForTest("1700000000")
	if err != nil {
		t.Fatalf("解析 unix 时间戳失败: %v", err)
	}
	if ts != 1700000000 {
		t.Fatalf("时间戳不正确: %d", ts)
	}

	// 空值
	if _, err := events.ParseSinceForTest("  "); err != nil {
		t.Fatalf("空值不应报错: %v", err)
	}

	// 非法值
	if _, err := events.ParseSinceForTest("not-a-time"); err == nil {
		t.Fatal("非法时间应返回错误")
	}
}
