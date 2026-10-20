package docker_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"

	"github.com/steven754/dockermanage/backend/internal/service/docker"
)

// resolveDockerHost 解析 Docker daemon 地址。
//
// 优先使用 DOCKER_HOST 环境变量；未设置时按 docker CLI 的当前 context 解析，
// 以便在 Docker Desktop（macOS）环境下也能连上 daemon。
func resolveDockerHost(t *testing.T) string {
	t.Helper()

	if host := os.Getenv("DOCKER_HOST"); host != "" {
		return host
	}

	out, err := exec.Command("docker", "context", "inspect", "--format", "{{.Endpoints.docker.Host}}").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// liveClient 返回连接本机 Docker daemon 的客户端，daemon 不可用时跳过测试
func liveClient(t *testing.T) *docker.Client {
	t.Helper()

	cli, err := docker.New(resolveDockerHost(t))
	if err != nil {
		t.Skipf("创建 Docker 客户端失败，跳过集成测试: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := cli.Ping(ctx); err != nil {
		_ = cli.Close()
		t.Skipf("Docker daemon 不可用，跳过集成测试: %v", err)
	}
	t.Cleanup(func() { _ = cli.Close() })
	return cli
}

// localImage 找一个本地已有镜像用于生命周期测试，避免联网拉取
func localImage(t *testing.T, cli *docker.Client) string {
	t.Helper()

	images, err := cli.Raw().ImageList(context.Background(), image.ListOptions{All: false})
	if err != nil {
		t.Skipf("列出镜像失败，跳过生命周期测试: %v", err)
	}
	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag != "" && tag != "<none>:<none>" {
				return tag
			}
		}
	}
	t.Skip("本地无可用镜像，跳过生命周期测试")
	return ""
}

func TestLiveListAndInspectContainers(t *testing.T) {
	cli := liveClient(t)
	svc := docker.NewContainerService(cli)
	ctx := context.Background()

	list, err := svc.List(ctx, docker.ListFilter{})
	if err != nil {
		t.Fatalf("列出容器失败: %v", err)
	}
	t.Logf("本机共 %d 个容器", len(list))

	for _, c := range list {
		if c.ID == "" {
			t.Fatal("容器 ID 不应为空")
		}
		if c.State == "" {
			t.Fatal("容器状态不应为空")
		}
		if c.Labels == nil {
			t.Fatal("容器标签不应为 nil")
		}
	}

	// 状态过滤
	running, err := svc.List(ctx, docker.ListFilter{Status: []string{"running"}})
	if err != nil {
		t.Fatalf("按状态过滤失败: %v", err)
	}
	for _, c := range running {
		if c.State != "running" {
			t.Fatalf("状态过滤失效: %s", c.State)
		}
	}

	if len(list) == 0 {
		t.Log("本机无容器，跳过详情校验")
		return
	}

	// 单个容器详情
	target := list[0]
	detail, err := svc.Get(ctx, target.ID)
	if err != nil {
		t.Fatalf("查询容器详情失败: %v", err)
	}
	if detail.ID != target.ID {
		t.Fatalf("详情 ID 不匹配: %s != %s", detail.ID, target.ID)
	}

	// inspect 原始 JSON
	raw, err := svc.InspectRaw(ctx, target.ID)
	if err != nil {
		t.Fatalf("读取 inspect 失败: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("inspect 原始 JSON 不应为空")
	}
}

func TestLiveContainerLifecycle(t *testing.T) {
	cli := liveClient(t)
	svc := docker.NewContainerService(cli)
	ctx := context.Background()

	image := localImage(t, cli)
	// 使用带时间戳的唯一名，避免与上一轮残留容器（正在移除中）抢占同名
	name := fmt.Sprintf("dockermanage-selftest-%d", time.Now().UnixNano())

	created, err := cli.Raw().ContainerCreate(ctx,
		&container.Config{
			Image:      image,
			Entrypoint: []string{"/bin/sleep"},
			Cmd:        []string{"300"},
			Labels:     map[string]string{"com.dockermanage.selftest": "1"},
		}, nil, &network.NetworkingConfig{}, nil, name)
	if err != nil {
		t.Fatalf("创建测试容器失败: %v", err)
	}
	id := created.ID
	t.Cleanup(func() {
		_ = cli.Raw().ContainerRemove(context.Background(), id, container.RemoveOptions{Force: true})
	})

	// 启动
	if err := svc.Start(ctx, id); err != nil {
		t.Fatalf("启动容器失败: %v", err)
	}
	// 暂停
	if err := svc.Pause(ctx, id); err != nil {
		t.Fatalf("暂停容器失败: %v", err)
	}
	// 恢复
	if err := svc.Unpause(ctx, id); err != nil {
		t.Fatalf("恢复容器失败: %v", err)
	}
	// 重启
	if err := svc.Restart(ctx, id); err != nil {
		t.Fatalf("重启容器失败: %v", err)
	}
	// 停止
	if err := svc.Stop(ctx, id); err != nil {
		t.Fatalf("停止容器失败: %v", err)
	}
	// 重新启动后杀死
	if err := svc.Start(ctx, id); err != nil {
		t.Fatalf("重新启动容器失败: %v", err)
	}
	if err := svc.Kill(ctx, id); err != nil {
		t.Fatalf("杀死容器失败: %v", err)
	}
	// 删除
	if err := svc.Remove(ctx, id, true); err != nil {
		t.Fatalf("删除容器失败: %v", err)
	}
}

func TestLiveContainerStats(t *testing.T) {
	cli := liveClient(t)
	svc := docker.NewContainerService(cli)
	ctx := context.Background()

	list, err := svc.List(ctx, docker.ListFilter{Status: []string{"running"}})
	if err != nil {
		t.Fatalf("列出容器失败: %v", err)
	}
	if len(list) == 0 {
		t.Skip("本机无运行中的容器，跳过资源采样测试")
		return
	}

	stat, err := svc.Stats(ctx, list[0].ID)
	if err != nil {
		t.Fatalf("读取容器资源失败: %v", err)
	}
	if stat.MemLimitBytes <= 0 {
		t.Fatalf("内存上限应大于 0，实际 %d", stat.MemLimitBytes)
	}
	t.Logf("容器 %s CPU=%.2f%% 内存=%d 字节", list[0].Name, stat.CPUPct, stat.MemUsageBytes)
}

func TestLiveImageService(t *testing.T) {
	cli := liveClient(t)
	svc := docker.NewImageService(cli)
	ctx := context.Background()

	images, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("列出镜像失败: %v", err)
	}
	t.Logf("本机共 %d 个镜像", len(images))
	for _, img := range images {
		if img.ID == "" {
			t.Fatal("镜像 ID 不应为空")
		}
		if img.RepoTags == nil {
			t.Fatal("repoTags 不应为 nil")
		}
	}
}

func TestLiveVolumeService(t *testing.T) {
	cli := liveClient(t)
	svc := docker.NewVolumeService(cli)
	ctx := context.Background()

	volumes, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("列出数据卷失败: %v", err)
	}
	t.Logf("本机共 %d 个数据卷", len(volumes))
	for _, v := range volumes {
		if v.Name == "" {
			t.Fatal("卷名不应为空")
		}
		if v.Orphan != (v.RefCount == 0) {
			t.Fatalf("orphan 标记与 refCount 不一致: %+v", v)
		}
	}
}

func TestLiveNetworkService(t *testing.T) {
	cli := liveClient(t)
	svc := docker.NewNetworkService(cli)
	ctx := context.Background()

	networks, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("列出网络失败: %v", err)
	}
	t.Logf("本机共 %d 个网络", len(networks))
	for _, n := range networks {
		if n.Name == "" {
			t.Fatal("网络名不应为空")
		}
		if n.Labels == nil {
			t.Fatal("网络标签不应为 nil")
		}
	}
}

func TestLiveSystemService(t *testing.T) {
	cli := liveClient(t)
	svc := docker.NewSystemService(cli)
	ctx := context.Background()

	if err := svc.Ping(ctx); err != nil {
		t.Fatalf("ping 失败: %v", err)
	}

	info, err := svc.Info(ctx)
	if err != nil {
		t.Fatalf("读取 daemon 信息失败: %v", err)
	}
	if info.ServerVersion == "" {
		t.Fatal("daemon 版本不应为空")
	}
	t.Logf("Docker %s / %s / %d 核", info.ServerVersion, info.OperatingSystem, info.NCPU)

	version, err := svc.Version(ctx)
	if err != nil {
		t.Fatalf("读取版本失败: %v", err)
	}
	if version.Version == "" {
		t.Fatal("版本号不应为空")
	}

	du, err := svc.DiskUsage(ctx)
	if err != nil {
		t.Fatalf("读取磁盘占用失败: %v", err)
	}
	var imageBytes int64
	for _, img := range du.Images {
		imageBytes += img.Size
	}
	t.Logf("镜像占用 %d 字节", imageBytes)
}

func TestLivePullImageIsIdempotentForLocalImage(t *testing.T) {
	cli := liveClient(t)
	svc := docker.NewImageService(cli)
	ctx := context.Background()

	image := localImage(t, cli)

	// 拉取本地已存在的镜像不会产生网络流量
	id, err := svc.Pull(ctx, image)
	if err != nil {
		t.Skipf("拉取镜像 %s 失败（可能需要联网），跳过: %v", image, err)
	}
	if id == "" {
		t.Fatal("镜像 ID 不应为空")
	}
	t.Logf("镜像 %s ID=%s", image, id)
}
