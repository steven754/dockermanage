package compose_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/steven754/dockermanage/backend/internal/service/compose"
)

// localComposeImage 找一个本地镜像，避免联网拉取
func localComposeImage(t *testing.T) string {
	t.Helper()

	out, err := exec.Command("docker", "images", "--format", "{{.Repository}}:{{.Tag}}").Output()
	if err != nil {
		t.Skipf("docker CLI 不可用，跳过 compose 集成测试: %v", err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "<none>") {
			continue
		}
		return line
	}
	t.Skip("本地无可用镜像，跳过 compose 集成测试")
	return ""
}

// dockerHostEnv 解析 Docker daemon 地址
func dockerHostEnv() string {
	if host := os.Getenv("DOCKER_HOST"); host != "" {
		return host
	}
	out, err := exec.Command("docker", "context", "inspect", "--format", "{{.Endpoints.docker.Host}}").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func TestComposeUpPsDown(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("未找到 docker CLI，跳过 compose 集成测试")
	}
	image := localComposeImage(t)

	root := t.TempDir()
	projectName := fmt.Sprintf("dm-compose-test-%d", time.Now().UnixNano())
	projectDir := filepath.Join(root, projectName)
	if err := os.MkdirAll(projectDir, 0o750); err != nil {
		t.Fatalf("创建项目目录失败: %v", err)
	}

	composeYAML := fmt.Sprintf(`services:
  worker:
    image: %s
    entrypoint: ["/bin/sleep", "300"]
`, image)
	if err := os.WriteFile(filepath.Join(projectDir, "docker-compose.yml"), []byte(composeYAML), 0o600); err != nil {
		t.Fatalf("写入 compose 文件失败: %v", err)
	}

	svc := compose.New(root, dockerHostEnv())
	t.Cleanup(func() {
		_ = svc.Down(t.Context(), projectName, true)
	})

	// 项目列表应包含新建项目
	projects, err := svc.ListProjects(t.Context())
	if err != nil {
		t.Fatalf("列出项目失败: %v", err)
	}
	found := false
	for _, p := range projects {
		if p.Name == projectName {
			found = true
			if p.ServiceCount != 1 {
				t.Fatalf("期望 1 个服务，实际 %d", p.ServiceCount)
			}
		}
	}
	if !found {
		t.Fatal("新建的 compose 项目未出现在列表中")
	}

	// 启动
	started, err := svc.Up(t.Context(), projectName, nil)
	if err != nil {
		t.Fatalf("启动 compose 项目失败: %v", err)
	}
	if len(started) != 1 || started[0] != "worker" {
		t.Fatalf("期望启动 worker，实际 %v", started)
	}

	// 状态查询
	services, err := svc.Ps(t.Context(), projectName)
	if err != nil {
		t.Fatalf("查询服务状态失败: %v", err)
	}
	if len(services) != 1 {
		t.Fatalf("期望 1 个服务，实际 %d", len(services))
	}
	if services[0].Name != "worker" {
		t.Fatalf("期望服务名 worker，实际 %s", services[0].Name)
	}
	if services[0].ContainerID == "" {
		t.Fatal("服务应带有容器 ID")
	}
	t.Logf("服务状态: %s / %s", services[0].State, services[0].Status)

	// 停止
	if err := svc.Down(t.Context(), projectName, true); err != nil {
		t.Fatalf("停止 compose 项目失败: %v", err)
	}
}

func TestComposeVersion(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("未找到 docker CLI，跳过")
	}
	svc := compose.New(t.TempDir(), dockerHostEnv())

	version := svc.Version(t.Context())
	if version == "" {
		t.Skip("docker compose 插件不可用，跳过")
	}
	t.Logf("docker compose 版本: %s", version)
}
