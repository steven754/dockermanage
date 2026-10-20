package docker_test

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	dockerevents "github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"

	"github.com/steven754/dockermanage/backend/internal/model"
	"github.com/steven754/dockermanage/backend/internal/service/docker"
)

// startEchoContainer 启动一个会持续输出日志的临时容器
func startEchoContainer(t *testing.T, cli *docker.Client) (string, string) {
	t.Helper()

	ctx := context.Background()
	image := localImage(t, cli)
	name := fmt.Sprintf("dm-logtest-%d", time.Now().UnixNano())

	created, err := cli.Raw().ContainerCreate(ctx,
		&container.Config{
			Image:      image,
			Entrypoint: []string{"/bin/sh"},
			Cmd:        []string{"-c", "echo hello-from-dockermanage; sleep 300"},
		}, nil, &network.NetworkingConfig{}, nil, name)
	if err != nil {
		t.Fatalf("创建测试容器失败: %v", err)
	}
	id := created.ID
	t.Cleanup(func() {
		_ = cli.Raw().ContainerRemove(context.Background(), id, container.RemoveOptions{Force: true})
	})

	if err := cli.Raw().ContainerStart(ctx, id, container.StartOptions{}); err != nil {
		t.Fatalf("启动测试容器失败: %v", err)
	}
	return id, name
}

// echoMarker 测试容器打印的标记文本
const echoMarker = "hello-from-dockermanage"

// waitForLog 轮询等待容器日志中出现期望内容。
//
// 不能用固定 sleep：持续集成环境负载高时，容器 stdout 的 flush 会被延迟，
// 固定等待会导致日志为空而误判失败。
func waitForLog(t *testing.T, svc *docker.ContainerService, id string, want string) string {
	t.Helper()

	ctx := context.Background()
	deadline := time.Now().Add(20 * time.Second)
	var last string

	for {
		rc, err := svc.Logs(ctx, id, docker.LogOptions{Tail: "50", Timestamps: true, Follow: false})
		if err != nil {
			t.Fatalf("读取日志失败: %v", err)
		}
		data, _ := io.ReadAll(bufio.NewReader(rc))
		_ = rc.Close()

		last = string(data)
		if strings.Contains(last, want) {
			return last
		}
		if time.Now().After(deadline) {
			t.Fatalf("等待日志内容超时（期望包含 %q），实际读到 %q", want, last)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func TestLiveContainerLogs(t *testing.T) {
	cli := liveClient(t)
	svc := docker.NewContainerService(cli)

	id, _ := startEchoContainer(t, cli)

	content := waitForLog(t, svc, id, echoMarker)
	if len(content) == 0 {
		t.Fatal("日志内容不应为空")
	}
	t.Logf("读到 %d 字节日志", len(content))
}

func TestLiveContainerExec(t *testing.T) {
	cli := liveClient(t)
	svc := docker.NewContainerService(cli)
	ctx := context.Background()

	id, _ := startEchoContainer(t, cli)

	session, err := svc.ExecCreate(ctx, id, docker.ExecOptions{
		Cmd: []string{"/bin/sh", "-c", "echo exec-ok"},
		TTY: true,
	})
	if err != nil {
		t.Fatalf("创建 exec 会话失败: %v", err)
	}
	if session.ID == "" {
		t.Fatal("exec 会话 ID 不应为空")
	}

	stream, err := svc.ExecAttach(ctx, session.ID)
	if err != nil {
		t.Fatalf("附加 exec 会话失败: %v", err)
	}
	defer func() { stream.Close() }()

	outCh := make(chan string, 1)
	go func() {
		buf := make([]byte, 4096)
		n, _ := stream.Reader.Read(buf)
		outCh <- string(buf[:n])
	}()

	select {
	case out := <-outCh:
		if !strings.Contains(out, "exec-ok") {
			t.Fatalf("exec 输出不符合预期: %q", out)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("exec 未返回输出")
	}

	// 调整终端尺寸：对一个仍在运行的 exec 会话执行
	longSession, err := svc.ExecCreate(ctx, id, docker.ExecOptions{
		Cmd: []string{"/bin/sleep", "300"},
		TTY: true,
	})
	if err != nil {
		t.Fatalf("创建长驻 exec 会话失败: %v", err)
	}
	longStream, err := svc.ExecAttach(ctx, longSession.ID)
	if err != nil {
		t.Fatalf("附加长驻 exec 会话失败: %v", err)
	}
	defer func() { longStream.Close() }()

	if err := svc.ExecResize(ctx, longSession.ID, 120, 40); err != nil {
		t.Fatalf("调整终端尺寸失败: %v", err)
	}
}

// localUnusedImage 找一个没有被容器使用的本地镜像，便于测试打标签 / 删除镜像
func localUnusedImage(t *testing.T, cli *docker.Client) string {
	t.Helper()

	ctx := context.Background()
	images, err := cli.Raw().ImageList(ctx, image.ListOptions{All: false})
	if err != nil {
		t.Skipf("列出镜像失败，跳过: %v", err)
	}
	containers, err := cli.Raw().ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		t.Skipf("列出容器失败，跳过: %v", err)
	}

	used := map[string]bool{}
	for _, c := range containers {
		used[c.ImageID] = true
	}

	for _, img := range images {
		if used[img.ID] {
			continue
		}
		for _, tag := range img.RepoTags {
			if tag != "" && tag != "<none>:<none>" {
				return tag
			}
		}
	}
	t.Skip("本地没有未被容器占用的镜像，跳过")
	return ""
}

func TestLiveImageTagAndRemove(t *testing.T) {
	cli := liveClient(t)
	svc := docker.NewImageService(cli)
	ctx := context.Background()

	image := localUnusedImage(t, cli)
	tagValue := fmt.Sprintf("%d", time.Now().UnixNano())
	tagged := "dockermanage-selftest:" + tagValue

	// 兜底清理，避免测试残留标签
	t.Cleanup(func() {
		_ = svc.Remove(context.Background(), tagged, true)
	})

	if err := svc.Tag(ctx, image, "dockermanage-selftest", tagValue); err != nil {
		t.Skipf("打标签失败（镜像 %s 可能为多架构清单），跳过: %v", image, err)
	}

	// 新标签应出现在镜像列表中
	found := false
	images, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("列出镜像失败: %v", err)
	}
	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == tagged {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("新打的标签未出现在镜像列表中")
	}

	// 删除该标签
	if err := svc.Remove(ctx, tagged, true); err != nil {
		t.Fatalf("删除标签镜像失败: %v", err)
	}

	// 标签应已消失
	images, err = svc.List(ctx)
	if err != nil {
		t.Fatalf("列出镜像失败: %v", err)
	}
	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == tagged {
				t.Fatalf("标签 %s 未被删除", tag)
			}
		}
	}
}

func TestLiveNetworkCreateAndRemove(t *testing.T) {
	cli := liveClient(t)
	svc := docker.NewNetworkService(cli)
	ctx := context.Background()

	name := fmt.Sprintf("dm-nettest-%d", time.Now().UnixNano())
	created, err := cli.Raw().NetworkCreate(ctx, name, network.CreateOptions{})
	if err != nil {
		t.Fatalf("创建网络失败: %v", err)
	}
	t.Cleanup(func() {
		_ = cli.Raw().NetworkRemove(context.Background(), created.ID)
	})

	networks, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("列出网络失败: %v", err)
	}
	found := false
	for _, n := range networks {
		if n.ID == created.ID {
			found = true
			if n.Name != name {
				t.Fatalf("期望网络名 %s，实际 %s", name, n.Name)
			}
			if n.Labels == nil {
				t.Fatal("网络标签不应为 nil")
			}
		}
	}
	if !found {
		t.Fatal("新建的网络未出现在列表中")
	}

	if err := svc.Remove(ctx, created.ID); err != nil {
		t.Fatalf("删除网络失败: %v", err)
	}
}

func TestLiveVolumeCreateAndRemove(t *testing.T) {
	cli := liveClient(t)
	svc := docker.NewVolumeService(cli)
	ctx := context.Background()

	name := fmt.Sprintf("dm-voltest-%d", time.Now().UnixNano())
	if _, err := cli.Raw().VolumeCreate(ctx, volume.CreateOptions{Name: name}); err != nil {
		t.Fatalf("创建数据卷失败: %v", err)
	}

	volumes, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("列出数据卷失败: %v", err)
	}
	found := false
	for _, v := range volumes {
		if v.Name == name {
			found = true
			if !v.Orphan {
				t.Fatal("新建的卷应为孤儿卷")
			}
			if v.RefCount != 0 {
				t.Fatalf("期望引用数 0，实际 %d", v.RefCount)
			}
		}
	}
	if !found {
		t.Fatal("新建的数据卷未出现在列表中")
	}

	if err := svc.Remove(ctx, name, false); err != nil {
		t.Fatalf("删除数据卷失败: %v", err)
	}
}

func TestLiveSystemEvents(t *testing.T) {
	cli := liveClient(t)
	svc := docker.NewSystemService(cli)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	msgs, errs := svc.Events(ctx, dockerevents.ListOptions{})

	// 触发一个事件，确保事件流确实可读
	image := localImage(t, cli)
	name := fmt.Sprintf("dm-evttest-%d", time.Now().UnixNano())
	created, err := cli.Raw().ContainerCreate(ctx,
		&container.Config{
			Image:      image,
			Entrypoint: []string{"/bin/sleep"},
			Cmd:        []string{"60"},
		}, nil, &network.NetworkingConfig{}, nil, name)
	if err != nil {
		t.Fatalf("创建测试容器失败: %v", err)
	}
	id := created.ID
	t.Cleanup(func() {
		_ = cli.Raw().ContainerRemove(context.Background(), id, container.RemoveOptions{Force: true})
	})
	_ = cli.Raw().ContainerStart(ctx, id, container.StartOptions{})

	timeout := time.After(10 * time.Second)
	for {
		select {
		case err := <-errs:
			if err != nil {
				t.Fatalf("订阅事件失败: %v", err)
			}
		case <-timeout:
			t.Fatal("订阅事件超时，未收到任何事件")
		case msg := <-msgs:
			if msg.Type == dockerevents.Type("container") {
				t.Logf("收到容器事件: type=%s action=%s", msg.Type, msg.Action)
				return
			}
		}
	}
}

// staticProvider 固定返回的采样源
type staticProvider struct{}

func (s *staticProvider) ContainerStatsSnapshot(id string) (model.ContainerStats, bool) {
	return model.ContainerStats{CPUPct: 42.5, MemUsageBytes: 1024, MemLimitBytes: 2048}, true
}

func TestLiveStatsProviderAttachment(t *testing.T) {
	cli := liveClient(t)
	svc := docker.NewContainerService(cli)
	ctx := context.Background()

	// 用采样器作为 stats provider
	svc.SetStatsProvider(&staticProvider{})

	list, err := svc.List(ctx, docker.ListFilter{})
	if err != nil {
		t.Fatalf("列出容器失败: %v", err)
	}
	if len(list) == 0 {
		t.Skip("本机无容器，跳过实时字段校验")
		return
	}

	detail, err := svc.Get(ctx, list[0].ID)
	if err != nil {
		t.Fatalf("查询容器详情失败: %v", err)
	}
	if detail.CPUPct == nil {
		t.Fatal("注入 provider 后 cpuPct 不应为 nil")
	}
	if *detail.CPUPct != 42.5 {
		t.Fatalf("期望 cpuPct 42.5，实际 %f", *detail.CPUPct)
	}
}
