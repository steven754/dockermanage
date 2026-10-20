// Package compose 通过 docker compose CLI 管理 compose 项目。
package compose

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/steven754/dockermanage/backend/internal/model"
)

// DefaultComposeFileName 默认 compose 文件名（简写保存接口使用）
const DefaultComposeFileName = "docker-compose.yml"

// composeFileNames 候选 compose 文件名（按优先级）
var composeFileNames = []string{
	"docker-compose.yml",
	"docker-compose.yaml",
	"compose.yml",
	"compose.yaml",
}

// Service compose 服务
type Service struct {
	rootDir     string
	dockerHost  string
	composeBin  string
	execTimeout time.Duration
}

// New 创建 compose 服务
func New(rootDir, dockerHost string) *Service {
	return &Service{
		rootDir:     rootDir,
		dockerHost:  dockerHost,
		composeBin:  "docker",
		execTimeout: 5 * time.Minute,
	}
}

// 业务错误
var (
	// ErrProjectNotFound 项目不存在
	ErrProjectNotFound = errors.New("compose 项目不存在")
	// ErrInvalidName 项目名非法
	ErrInvalidName = errors.New("compose 项目名非法")
	// ErrFileNotFound 文件不存在
	ErrFileNotFound = errors.New("文件不存在")
)

// ValidateName 校验项目名，禁止路径穿越
func ValidateName(name string) error {
	if name == "" || name == "." || name == ".." {
		return ErrInvalidName
	}
	if strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") {
		return ErrInvalidName
	}
	return nil
}

// ValidateFileName 校验文件名，禁止路径穿越
func ValidateFileName(name string) error {
	if name == "" {
		return ErrInvalidName
	}
	if strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") {
		return ErrInvalidName
	}
	return nil
}

// ProjectPath 返回项目目录绝对路径
func (s *Service) ProjectPath(name string) (string, error) {
	if err := ValidateName(name); err != nil {
		return "", err
	}
	return filepath.Join(s.rootDir, name), nil
}

// ComposeFilePath 返回项目内 compose 文件路径
func (s *Service) ComposeFilePath(name string) (string, error) {
	dir, err := s.ProjectPath(name)
	if err != nil {
		return "", err
	}
	file := FindComposeFile(dir)
	if file == "" {
		return "", ErrFileNotFound
	}
	return file, nil
}

// FindComposeFile 在目录中查找 compose 文件
func FindComposeFile(dir string) string {
	for _, candidate := range composeFileNames {
		path := filepath.Join(dir, candidate)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	return ""
}

// ListProjects 列出 compose 目录下的所有项目
func (s *Service) ListProjects(ctx context.Context) ([]*model.ComposeProject, error) {
	entries, err := os.ReadDir(s.rootDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*model.ComposeProject{}, nil
		}
		return nil, fmt.Errorf("读取 compose 目录失败: %w", err)
	}

	out := make([]*model.ComposeProject, 0)
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		project, err := s.describeProject(ctx, entry.Name())
		if err != nil {
			// 单个项目异常不影响整体列表
			continue
		}
		out = append(out, project)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// describeProject 构造项目摘要
func (s *Service) describeProject(ctx context.Context, name string) (*model.ComposeProject, error) {
	dir, err := s.ProjectPath(name)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	project := &model.ComposeProject{
		Name:  name,
		Path:  dir,
		Files: []string{},
	}
	var latest time.Time
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		project.Files = append(project.Files, e.Name())
		if info.ModTime().After(latest) {
			latest = info.ModTime()
		}
	}
	project.FileCount = len(project.Files)
	project.UpdatedAt = model.FormatTime(latest)

	if file := FindComposeFile(dir); file != "" {
		if services, err := parseServiceCount(file); err == nil {
			project.ServiceCount = services
		}
	}

	if services, err := s.Ps(ctx, name); err == nil {
		for _, svc := range services {
			if svc.State == "running" {
				project.RunningCount++
			}
		}
	}
	return project, nil
}

// parseServiceCount 解析 compose 文件中的服务数量
func parseServiceCount(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var doc struct {
		Services map[string]any `yaml:"services"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return 0, err
	}
	return len(doc.Services), nil
}

// GetProject 返回项目详情
func (s *Service) GetProject(ctx context.Context, name string) (*model.ComposeProjectDetail, error) {
	dir, err := s.ProjectPath(name)
	if err != nil {
		return nil, err
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return nil, ErrProjectNotFound
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	detail := &model.ComposeProjectDetail{
		Name:     name,
		Path:     dir,
		Files:    []model.ComposeFile{},
		Services: []model.ComposeService{},
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		detail.Files = append(detail.Files, model.ComposeFile{
			Name:      e.Name(),
			Content:   string(content),
			SizeBytes: info.Size(),
			UpdatedAt: model.FormatTime(info.ModTime()),
		})
	}

	services, err := s.Ps(ctx, name)
	if err == nil {
		detail.Services = services
	}
	return detail, nil
}

// SaveFile 保存项目内的文件
func (s *Service) SaveFile(name, fileName, content string) (int64, error) {
	dir, err := s.ProjectPath(name)
	if err != nil {
		return 0, err
	}
	if err := ValidateFileName(fileName); err != nil {
		return 0, err
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return 0, ErrProjectNotFound
	}

	path := filepath.Join(dir, fileName)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return 0, fmt.Errorf("写入文件失败: %w", err)
	}
	return int64(len(content)), nil
}

// Up 启动 compose 项目
func (s *Service) Up(ctx context.Context, name string, services []string) ([]string, error) {
	file, err := s.ComposeFilePath(name)
	if err != nil {
		return nil, err
	}
	args := s.baseArgs(name, file, "up", "-d", "--remove-orphans")
	if len(services) > 0 {
		args = append(args, services...)
	}
	if _, err := s.run(ctx, args...); err != nil {
		return nil, err
	}

	started := services
	if len(started) == 0 {
		started = []string{}
		if ps, err := s.Ps(ctx, name); err == nil {
			for _, svc := range ps {
				started = append(started, svc.Name)
			}
		}
	}
	return started, nil
}

// Down 停止 compose 项目
func (s *Service) Down(ctx context.Context, name string, removeVolumes bool) error {
	file, err := s.ComposeFilePath(name)
	if err != nil {
		return err
	}
	args := s.baseArgs(name, file, "down")
	if removeVolumes {
		args = append(args, "-v")
	}
	_, err = s.run(ctx, args...)
	return err
}

// Ps 查询 compose 项目服务状态
func (s *Service) Ps(ctx context.Context, name string) ([]model.ComposeService, error) {
	file, err := s.ComposeFilePath(name)
	if err != nil {
		return nil, err
	}
	out, err := s.run(ctx, s.baseArgs(name, file, "ps", "--format", "json")...)
	if err != nil {
		return nil, err
	}
	return parsePsOutput(out), nil
}

// Version 返回 compose 插件版本号
func (s *Service) Version(ctx context.Context) string {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, s.composeBin, "compose", "version", "--short")
	if s.dockerHost != "" {
		cmd.Env = append(os.Environ(), "DOCKER_HOST="+s.dockerHost)
	}
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// baseArgs 组装 compose 命令行参数
func (s *Service) baseArgs(name, file string, args ...string) []string {
	base := []string{"compose", "-p", name, "-f", file}
	return append(base, args...)
}

// run 执行 compose 命令，返回标准输出
func (s *Service) run(ctx context.Context, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, s.execTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, s.composeBin, args...)
	if s.dockerHost != "" {
		cmd.Env = append(os.Environ(), "DOCKER_HOST="+s.dockerHost)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg == "" {
			msg = err.Error()
		}
		return stdout.String(), fmt.Errorf("docker compose 执行失败: %s", msg)
	}
	return stdout.String(), nil
}

// psItem compose ps --format json 的单条记录
type psItem struct {
	ID         string `json:"ID"`
	Name       string `json:"Name"`
	Service    string `json:"Service"`
	Image      string `json:"Image"`
	State      string `json:"State"`
	Status     string `json:"Status"`
	Publishers []struct {
		PublishedPort uint16 `json:"PublishedPort"`
		TargetPort    uint16 `json:"TargetPort"`
		Protocol      string `json:"Protocol"`
	} `json:"Publishers"`
}

// parsePsOutput 解析 compose ps 输出，兼容数组与 NDJSON 两种格式
func parsePsOutput(out string) []model.ComposeService {
	out = strings.TrimSpace(out)
	if out == "" {
		return []model.ComposeService{}
	}

	var items []psItem
	if err := json.Unmarshal([]byte(out), &items); err != nil {
		// 回退到 NDJSON 逐行解析
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var item psItem
			if err := json.Unmarshal([]byte(line), &item); err != nil {
				continue
			}
			items = append(items, item)
		}
	}

	services := make([]model.ComposeService, 0, len(items))
	for _, it := range items {
		// 新版 compose 的 Name 是容器名，Service 才是服务名；旧版只有 Name
		name := it.Service
		if name == "" {
			name = it.Name
		}
		svc := model.ComposeService{
			Name:        name,
			Image:       it.Image,
			State:       it.State,
			Status:      it.Status,
			ContainerID: it.ID,
			Ports:       []model.ContainerPort{},
		}
		for _, p := range it.Publishers {
			svc.Ports = append(svc.Ports, model.ContainerPort{
				PrivatePort: p.TargetPort,
				PublicPort:  p.PublishedPort,
				Type:        strings.ToLower(p.Protocol),
			})
		}
		services = append(services, svc)
	}
	return services
}
