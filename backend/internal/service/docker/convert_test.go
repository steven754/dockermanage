package docker_test

import (
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"

	"github.com/steven754/dockermanage/backend/internal/model"
	"github.com/steven754/dockermanage/backend/internal/service/docker"
)

func TestNormalizeImageRef(t *testing.T) {
	cases := map[string]string{
		"nginx":               "nginx:latest",
		"nginx:1.27":          "nginx:1.27",
		"library/nginx":       "library/nginx:latest",
		"registry:5000/app":   "registry:5000/app:latest",
		"registry:5000/a:1":   "registry:5000/a:1",
		"nginx@sha256:abcdef": "nginx@sha256:abcdef",
		"  ":                  "",
	}
	for in, want := range cases {
		if got := docker.NormalizeImageRefForTest(in); got != want {
			t.Fatalf("normalize(%q) = %q，期望 %q", in, got, want)
		}
	}
}

func TestParseHostPort(t *testing.T) {
	if got := docker.ParseHostPortForTest("8080"); got != 8080 {
		t.Fatalf("期望 8080，实际 %d", got)
	}
	if got := docker.ParseHostPortForTest(""); got != 0 {
		t.Fatalf("期望 0，实际 %d", got)
	}
	if got := docker.ParseHostPortForTest("abc"); got != 0 {
		t.Fatalf("非法端口应返回 0，实际 %d", got)
	}
	if got := docker.ParseHostPortForTest("99999"); got != 0 {
		t.Fatalf("越界端口应返回 0，实际 %d", got)
	}
}

func TestParseDockerTime(t *testing.T) {
	got := docker.ParseDockerTimeForTest("2026-10-20T15:30:00.123456789Z")
	if got.IsZero() {
		t.Fatal("应能解析 RFC3339Nano 时间")
	}
	if got.Year() != 2026 {
		t.Fatalf("期望年份 2026，实际 %d", got.Year())
	}

	if !docker.ParseDockerTimeForTest("").IsZero() {
		t.Fatal("空字符串应返回零值")
	}
	if !docker.ParseDockerTimeForTest("not-a-time").IsZero() {
		t.Fatal("非法时间应返回零值")
	}
}

func TestContainerFromInspect(t *testing.T) {
	cj := types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			ID:           "abc123",
			Name:         "/nginx-proxy",
			Image:        "sha256:abc123",
			RestartCount: 3,
			Created:      "2026-10-17T08:00:00Z",
			State: &types.ContainerState{
				Status:     "running",
				ExitCode:   0,
				StartedAt:  "2026-10-17T08:00:01Z",
				FinishedAt: "0001-01-01T00:00:00Z",
				Health:     &types.Health{Status: "Healthy"},
			},
			HostConfig: &container.HostConfig{},
		},
		Config: &container.Config{
			Image: "nginx:1.27-alpine",
			Labels: map[string]string{
				"com.docker.compose.project": "media-server",
			},
		},
		NetworkSettings: &types.NetworkSettings{
			Networks: map[string]*network.EndpointSettings{
				"proxy": {IPAddress: "172.18.0.3", Aliases: []string{"nginx"}},
			},
		},
		Mounts: []types.MountPoint{
			docker.MountPointForTest(mount.TypeVolume, "nginx_data", "/var/lib/docker/volumes/nginx_data/_data", "/data", true),
		},
	}

	c := docker.ContainerFromInspectForTest(cj)
	if c.ID != "abc123" {
		t.Fatalf("期望 ID abc123，实际 %s", c.ID)
	}
	if c.Name != "nginx-proxy" {
		t.Fatalf("期望名称去掉前导斜杠，实际 %s", c.Name)
	}
	if c.Image != "nginx:1.27-alpine" {
		t.Fatalf("期望镜像 nginx:1.27-alpine，实际 %s", c.Image)
	}
	if c.ImageID != "sha256:abc123" {
		t.Fatalf("期望镜像 ID sha256:abc123，实际 %s", c.ImageID)
	}
	if c.State != "running" {
		t.Fatalf("期望状态 running，实际 %s", c.State)
	}
	if c.Health != "healthy" {
		t.Fatalf("期望健康状态 healthy，实际 %s", c.Health)
	}
	if c.RestartCount != 3 {
		t.Fatalf("期望重启次数 3，实际 %d", c.RestartCount)
	}
	if c.ComposeProject != "media-server" {
		t.Fatalf("期望 compose 项目 media-server，实际 %s", c.ComposeProject)
	}
	if c.CreatedAt != "2026-10-17T08:00:00Z" {
		t.Fatalf("期望创建时间 2026-10-17T08:00:00Z，实际 %s", c.CreatedAt)
	}
	if c.FinishedAt != nil {
		t.Fatalf("运行中的容器 finishedAt 应为 null，实际 %v", *c.FinishedAt)
	}
	if len(c.Mounts) != 1 {
		t.Fatalf("期望 1 个挂载，实际 %d", len(c.Mounts))
	}
	if c.Mounts[0].Name != "nginx_data" || c.Mounts[0].Destination != "/data" {
		t.Fatalf("挂载信息不正确: %+v", c.Mounts[0])
	}
	if c.Mounts[0].ReadOnly {
		t.Fatal("RW 挂载不应为只读")
	}
	if len(c.Networks) != 1 {
		t.Fatalf("期望 1 个网络，实际 %d", len(c.Networks))
	}
}

func TestHumanStatus(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name  string
		state *types.ContainerState
		want  string
	}{
		{"running", &types.ContainerState{Status: "running", StartedAt: now.Add(-3 * time.Hour).Format(time.RFC3339Nano)}, "Up 3 hours"},
		{"exited", &types.ContainerState{Status: "exited", ExitCode: 137, FinishedAt: now.Add(-2 * time.Minute).Format(time.RFC3339Nano)}, "Exited (137) 2 minutes ago"},
		{"created", &types.ContainerState{Status: "created"}, "Created"},
		{"paused", &types.ContainerState{Status: "running", Paused: true, StartedAt: now.Add(-time.Hour).Format(time.RFC3339Nano)}, "Up 1 hours (Paused)"},
		{"dead", &types.ContainerState{Status: "dead"}, "Dead"},
	}

	for _, tc := range cases {
		cj := types.ContainerJSON{ContainerJSONBase: &types.ContainerJSONBase{State: tc.state}}
		got := docker.HumanStatusForTest(cj)
		if got != tc.want {
			t.Fatalf("%s: 期望 %q，实际 %q", tc.name, tc.want, got)
		}
	}
}

func TestCalculateCPUPercent(t *testing.T) {
	cur := &container.CPUStats{
		CPUUsage:    container.CPUUsage{TotalUsage: 2_000_000_000},
		SystemUsage: 20_000_000_000,
	}
	pre := &container.CPUStats{
		CPUUsage:    container.CPUUsage{TotalUsage: 1_000_000_000},
		SystemUsage: 10_000_000_000,
	}

	// (1e9 / 1e10) * 4 * 100 = 40
	got := docker.CalculateCPUPercentForTest(cur, pre, 4)
	if got < 39.9 || got > 40.1 {
		t.Fatalf("期望 CPU 40%%，实际 %f", got)
	}

	// 无增量时应为 0
	if got := docker.CalculateCPUPercentForTest(pre, pre, 4); got != 0 {
		t.Fatalf("期望 CPU 0，实际 %f", got)
	}
}

func TestApplyFilter(t *testing.T) {
	list := []*model.Container{
		{ID: "1", Name: "nginx-proxy", Image: "nginx:1.27", State: "running"},
		{ID: "2", Name: "postgres", Image: "postgres:16", State: "exited"},
		{ID: "3", Name: "redis", Image: "redis:7", State: "running"},
	}

	// 状态过滤
	got := docker.ApplyFilterForTest(list, docker.ListFilter{Status: []string{"running"}})
	if len(got) != 2 {
		t.Fatalf("期望 2 个容器，实际 %d", len(got))
	}

	// 多状态过滤
	got = docker.ApplyFilterForTest(list, docker.ListFilter{Status: []string{"running", "exited"}})
	if len(got) != 3 {
		t.Fatalf("期望 3 个容器，实际 %d", len(got))
	}

	// 名称模糊搜索
	got = docker.ApplyFilterForTest(list, docker.ListFilter{Search: "ngin"})
	if len(got) != 1 || got[0].Name != "nginx-proxy" {
		t.Fatalf("名称搜索结果不正确: %+v", got)
	}

	// 镜像模糊搜索
	got = docker.ApplyFilterForTest(list, docker.ListFilter{Search: "postgres"})
	if len(got) != 1 || got[0].ID != "2" {
		t.Fatalf("镜像搜索结果不正确: %+v", got)
	}

	// 分页
	got = docker.ApplyFilterForTest(list, docker.ListFilter{Limit: 2})
	if len(got) != 2 {
		t.Fatalf("期望 2 个容器，实际 %d", len(got))
	}
	got = docker.ApplyFilterForTest(list, docker.ListFilter{Offset: 1, Limit: 1})
	if len(got) != 1 || got[0].ID != "2" {
		t.Fatalf("分页结果不正确: %+v", got)
	}

	// 偏移越界
	got = docker.ApplyFilterForTest(list, docker.ListFilter{Offset: 99})
	if len(got) != 0 {
		t.Fatalf("期望空结果，实际 %d", len(got))
	}
}

func TestContainerFromSummary(t *testing.T) {
	raw := types.Container{
		ID:      "abc",
		Names:   []string{"/web"},
		Image:   "nginx:1.27",
		ImageID: "sha256:abc",
		State:   "running",
		Status:  "Up 3 days",
		Labels:  map[string]string{"com.docker.compose.project": "proj"},
	}
	c := docker.ContainerFromSummaryForTest(&raw)
	if c.ID != "abc" || c.Name != "web" {
		t.Fatalf("摘要转换不正确: %+v", c)
	}
	if c.ComposeProject != "proj" {
		t.Fatalf("期望 compose 项目 proj，实际 %s", c.ComposeProject)
	}
	if c.Ports == nil || c.Networks == nil || c.Mounts == nil || c.Labels == nil {
		t.Fatal("集合字段不应为 nil")
	}
}

func TestDeltaUint(t *testing.T) {
	if got := docker.DeltaUintForTest(100, 40); got != 60 {
		t.Fatalf("期望 60，实际 %d", got)
	}
	// 计数器回绕
	if got := docker.DeltaUintForTest(10, 100); got != 10 {
		t.Fatalf("期望 10，实际 %d", got)
	}
}
