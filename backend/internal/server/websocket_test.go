package server_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"

	"github.com/steven754/dockermanage/backend/internal/db"
	"github.com/steven754/dockermanage/backend/internal/handler"
	"github.com/steven754/dockermanage/backend/internal/repo"
	"github.com/steven754/dockermanage/backend/internal/server"
	"github.com/steven754/dockermanage/backend/internal/service/auth"
	"github.com/steven754/dockermanage/backend/internal/service/docker"
)

// liveEnv 连接真实 Docker daemon 的测试环境
type liveEnv struct {
	srv   *httptest.Server
	token string
}

// wsURL 构造 WebSocket 地址
func (e *liveEnv) wsURL(path string) string {
	return "ws://" + strings.TrimPrefix(e.srv.URL, "http://") + path
}

// resolveDockerHost 解析 Docker daemon 地址
func resolveDockerHost(t *testing.T) string {
	t.Helper()

	if host := strings.TrimSpace(os.Getenv("DOCKER_HOST")); host != "" {
		return host
	}
	out, err := exec.Command("docker", "context", "inspect", "--format", "{{.Endpoints.docker.Host}}").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// localImageForTest 找一个本地已有镜像
func localImageForTest(t *testing.T, cli *docker.Client) string {
	t.Helper()

	images, err := cli.Raw().ImageList(context.Background(), image.ListOptions{All: false})
	if err != nil {
		t.Skipf("列出镜像失败，跳过: %v", err)
	}
	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag != "" && tag != "<none>:<none>" {
				return tag
			}
		}
	}
	t.Skip("本地无可用镜像，跳过")
	return ""
}

// newLiveEnv 构造连接真实 Docker 的测试环境
func newLiveEnv(t *testing.T) (*liveEnv, *docker.Client) {
	t.Helper()

	cli, err := docker.New(resolveDockerHost(t))
	if err != nil {
		t.Skipf("创建 Docker 客户端失败，跳过: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := cli.Ping(ctx); err != nil {
		_ = cli.Close()
		t.Skipf("Docker daemon 不可用，跳过: %v", err)
	}
	t.Cleanup(func() { _ = cli.Close() })

	sqlDB, err := db.Open("", true)
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.Migrate(sqlDB); err != nil {
		t.Fatalf("执行迁移失败: %v", err)
	}
	store := repo.New(sqlDB)

	tokens := auth.NewTokenManager([]byte("0123456789abcdef0123456789abcdef"))
	authSvc := auth.New(store.Users, tokens, t.TempDir())
	password, err := authSvc.Bootstrap(t.Context())
	if err != nil {
		t.Fatalf("初始化用户失败: %v", err)
	}

	h := handler.New(handler.Deps{
		Auth:       authSvc,
		Containers: docker.NewContainerService(cli),
		Images:     docker.NewImageService(cli),
		Volumes:    docker.NewVolumeService(cli),
		Networks:   docker.NewNetworkService(cli),
		System:     docker.NewSystemService(cli),
		Store:      store,
	})

	router := server.NewRouter(server.Deps{Handler: h, Tokens: tokens})
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	// 登录换取 token
	loginResp, err := http.Post(srv.URL+"/api/v1/auth/login", "application/json",
		strings.NewReader(fmt.Sprintf(`{"username":"%s","password":"%s"}`, auth.DefaultUsername, password)))
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}
	defer func() { _ = loginResp.Body.Close() }()
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("登录返回 %d", loginResp.StatusCode)
	}
	var login struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.NewDecoder(loginResp.Body).Decode(&login); err != nil {
		t.Fatalf("解析登录响应失败: %v", err)
	}

	return &liveEnv{srv: srv, token: login.AccessToken}, cli
}

// startTestContainer 创建并启动一个临时容器
func startTestContainer(t *testing.T, cli *docker.Client, prefix, script string) string {
	t.Helper()

	image := localImageForTest(t, cli)
	name := fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())

	created, err := cli.Raw().ContainerCreate(context.Background(),
		&container.Config{
			Image:      image,
			Entrypoint: []string{"/bin/sh"},
			Cmd:        []string{"-c", script},
		}, nil, &network.NetworkingConfig{}, nil, name)
	if err != nil {
		t.Fatalf("创建测试容器失败: %v", err)
	}
	id := created.ID
	t.Cleanup(func() {
		_ = cli.Raw().ContainerRemove(context.Background(), id, container.RemoveOptions{Force: true})
	})

	if err := cli.Raw().ContainerStart(context.Background(), id, container.StartOptions{}); err != nil {
		t.Fatalf("启动测试容器失败: %v", err)
	}
	time.Sleep(600 * time.Millisecond)
	return id
}

// TestWebSocketContainerLogs 端到端验证容器日志 WebSocket
func TestWebSocketContainerLogs(t *testing.T) {
	env, cli := newLiveEnv(t)

	id := startTestContainer(t, cli, "dm-ws-log", "echo hello-ws; sleep 300")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// 浏览器无法自定义 WebSocket 请求头，这里同时验证 ?token= 兜底方式
	conn, _, err := websocket.Dial(ctx,
		fmt.Sprintf("%s/api/v1/containers/%s/logs?tail=50&follow=false&token=%s", env.wsURL(""), id, env.token),
		nil)
	if err != nil {
		t.Fatalf("WebSocket 握手失败: %v", err)
	}
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	var builder strings.Builder
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			break
		}
		builder.Write(data)
		if strings.Contains(builder.String(), "hello-ws") {
			break
		}
	}

	if !strings.Contains(builder.String(), "hello-ws") {
		t.Fatalf("未收到预期日志内容，实际收到: %q", builder.String())
	}
	t.Logf("WS 收到日志: %q", strings.TrimSpace(builder.String()))
}

// TestWebSocketContainerExec 端到端验证容器 exec WebSocket
func TestWebSocketContainerExec(t *testing.T) {
	env, cli := newLiveEnv(t)

	id := startTestContainer(t, cli, "dm-ws-exec", "sleep 300")

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx,
		fmt.Sprintf("%s/api/v1/containers/%s/exec?cmd=/bin/sh&token=%s", env.wsURL(""), id, env.token),
		nil)
	if err != nil {
		t.Fatalf("WebSocket 握手失败: %v", err)
	}
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	if err := conn.Write(ctx, websocket.MessageText, []byte("echo hello-exec\n")); err != nil {
		t.Fatalf("发送命令失败: %v", err)
	}

	var builder strings.Builder
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			break
		}
		builder.Write(data)
		if strings.Contains(builder.String(), "hello-exec") {
			break
		}
	}

	if !strings.Contains(builder.String(), "hello-exec") {
		t.Fatalf("未收到预期 exec 输出，实际收到: %q", builder.String())
	}
	t.Logf("WS 收到 exec 输出: %q", strings.TrimSpace(builder.String()))
}

// TestWebSocketRequiresAuth 验证 WebSocket 同样受鉴权保护
func TestWebSocketRequiresAuth(t *testing.T) {
	env, cli := newLiveEnv(t)

	id := startTestContainer(t, cli, "dm-ws-auth", "sleep 300")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, _, err := websocket.Dial(ctx,
		fmt.Sprintf("%s/api/v1/containers/%s/logs", env.wsURL(""), id), nil)
	if err == nil {
		t.Fatal("未携带 token 的 WebSocket 握手应失败")
	}
	t.Logf("未鉴权握手被拒绝: %v", err)
}
