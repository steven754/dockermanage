package compose_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/steven754/dockermanage/backend/internal/service/compose"
)

func TestValidateName(t *testing.T) {
	valid := []string{"media-server", "app1", "my_project"}
	for _, name := range valid {
		if err := compose.ValidateName(name); err != nil {
			t.Fatalf("名称 %q 应合法: %v", name, err)
		}
	}

	invalid := []string{"", ".", "..", "../etc", "a/b", "a\\b", "..hidden"}
	for _, name := range invalid {
		if err := compose.ValidateName(name); err == nil {
			t.Fatalf("名称 %q 应被拒绝", name)
		}
	}
}

func TestValidateFileName(t *testing.T) {
	if err := compose.ValidateFileName("docker-compose.yml"); err != nil {
		t.Fatalf("合法文件名应通过: %v", err)
	}
	for _, name := range []string{"", "../passwd", "a/b", "..\\x"} {
		if err := compose.ValidateFileName(name); err == nil {
			t.Fatalf("文件名 %q 应被拒绝", name)
		}
	}
}

func TestParsePsOutputArrayFormat(t *testing.T) {
	out := `[
	  {"ID":"abc123","Name":"web","Image":"nginx:1.27","State":"running","Status":"Up 2 hours",
	   "Publishers":[{"PublishedPort":8080,"TargetPort":80,"Protocol":"tcp"}]},
	  {"ID":"def456","Name":"db","Image":"postgres:16","State":"exited","Status":"Exited (0) 1 minute ago"}
	]`

	services := compose.ParsePsOutputForTest(out)
	if len(services) != 2 {
		t.Fatalf("期望 2 个服务，实际 %d", len(services))
	}
	if services[0].Name != "web" {
		t.Fatalf("期望服务名 web，实际 %s", services[0].Name)
	}
	if services[0].State != "running" {
		t.Fatalf("期望状态 running，实际 %s", services[0].State)
	}
	if len(services[0].Ports) != 1 {
		t.Fatalf("期望 1 个端口映射，实际 %d", len(services[0].Ports))
	}
	if services[0].Ports[0].PublicPort != 8080 || services[0].Ports[0].PrivatePort != 80 {
		t.Fatalf("端口映射不正确: %+v", services[0].Ports[0])
	}
	if services[1].State != "exited" {
		t.Fatalf("期望状态 exited，实际 %s", services[1].State)
	}
}

func TestParsePsOutputNDJSONFormat(t *testing.T) {
	out := `{"ID":"abc","Name":"web","Image":"nginx","State":"running","Status":"Up"}
{"ID":"def","Name":"db","Image":"postgres","State":"running","Status":"Up"}`

	services := compose.ParsePsOutputForTest(out)
	if len(services) != 2 {
		t.Fatalf("期望 2 个服务，实际 %d", len(services))
	}
	if services[1].Name != "db" {
		t.Fatalf("期望服务名 db，实际 %s", services[1].Name)
	}
}

func TestParsePsOutputEmpty(t *testing.T) {
	services := compose.ParsePsOutputForTest("   ")
	if services == nil {
		t.Fatal("空输出应返回空切片而非 nil")
	}
	if len(services) != 0 {
		t.Fatalf("期望 0 个服务，实际 %d", len(services))
	}
}

func TestFindComposeFile(t *testing.T) {
	dir := t.TempDir()

	if compose.FindComposeFile(dir) != "" {
		t.Fatal("空目录不应找到 compose 文件")
	}

	path := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(path, []byte("services:\n  web:\n    image: nginx\n"), 0o600); err != nil {
		t.Fatalf("写入文件失败: %v", err)
	}
	if got := compose.FindComposeFile(dir); got != path {
		t.Fatalf("期望找到 %s，实际 %s", path, got)
	}
}

func TestServiceSaveAndList(t *testing.T) {
	root := t.TempDir()
	svc := compose.New(root, "")

	// 目录不存在时应返回空列表
	projects, err := svc.ListProjects(t.Context())
	if err != nil {
		t.Fatalf("列出项目失败: %v", err)
	}
	if len(projects) != 0 {
		t.Fatalf("期望 0 个项目，实际 %d", len(projects))
	}

	// 创建项目目录与 compose 文件
	projectDir := filepath.Join(root, "media-server")
	if err := os.MkdirAll(projectDir, 0o750); err != nil {
		t.Fatalf("创建项目目录失败: %v", err)
	}
	content := "services:\n  web:\n    image: nginx:1.27\n  db:\n    image: postgres:16\n"
	if err := os.WriteFile(filepath.Join(projectDir, "docker-compose.yml"), []byte(content), 0o600); err != nil {
		t.Fatalf("写入 compose 文件失败: %v", err)
	}

	projects, err = svc.ListProjects(t.Context())
	if err != nil {
		t.Fatalf("列出项目失败: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("期望 1 个项目，实际 %d", len(projects))
	}
	if projects[0].Name != "media-server" {
		t.Fatalf("期望项目名 media-server，实际 %s", projects[0].Name)
	}
	if projects[0].ServiceCount != 2 {
		t.Fatalf("期望 2 个服务，实际 %d", projects[0].ServiceCount)
	}
	if projects[0].FileCount != 1 {
		t.Fatalf("期望 1 个文件，实际 %d", projects[0].FileCount)
	}

	// 保存文件
	size, err := svc.SaveFile("media-server", "docker-compose.yml", content)
	if err != nil {
		t.Fatalf("保存文件失败: %v", err)
	}
	if size != int64(len(content)) {
		t.Fatalf("期望文件大小 %d，实际 %d", len(content), size)
	}

	// 路径穿越应被拒绝
	if _, err := svc.SaveFile("media-server", "../escape.yml", "x"); err == nil {
		t.Fatal("路径穿越应被拒绝")
	}

	// 项目详情
	detail, err := svc.GetProject(t.Context(), "media-server")
	if err != nil {
		t.Fatalf("查询项目详情失败: %v", err)
	}
	if len(detail.Files) != 1 {
		t.Fatalf("期望 1 个文件，实际 %d", len(detail.Files))
	}
	if detail.Files[0].Content != content {
		t.Fatal("文件内容不匹配")
	}

	// 不存在的项目
	if _, err := svc.GetProject(t.Context(), "not-exist"); err == nil {
		t.Fatal("不存在的项目应返回错误")
	}
}
