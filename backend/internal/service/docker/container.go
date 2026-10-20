package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/steven754/dockermanage/backend/internal/model"
)

// StatsProvider 提供容器实时资源采样缓存，由 stats 服务注入。
type StatsProvider interface {
	// ContainerStatsSnapshot 返回指定容器的采样快照，第二个返回值表示是否存在
	ContainerStatsSnapshot(id string) (model.ContainerStats, bool)
}

// ListFilter 容器列表过滤条件
type ListFilter struct {
	// Status 状态过滤，逗号分隔的多值
	Status []string
	// Search 名称 / 镜像模糊匹配
	Search string
	// Label 标签过滤，形如 k=v
	Label string
	// Limit 返回条数上限，<=0 表示不限
	Limit int
	// Offset 偏移量
	Offset int
}

// ContainerService 容器相关操作
type ContainerService struct {
	cli *Client

	mu            sync.RWMutex
	stats         StatsProvider
	cache         []*model.Container
	cacheAt       time.Time
	cacheTTL      time.Duration
	statsCounters map[string]statsCounter
}

// NewContainerService 创建容器服务
func NewContainerService(cli *Client) *ContainerService {
	return &ContainerService{
		cli:           cli,
		cacheTTL:      2 * time.Second,
		statsCounters: map[string]statsCounter{},
	}
}

// SetStatsProvider 注入采样器（避免包循环依赖，由 stats 服务在启动时注册）
func (s *ContainerService) SetStatsProvider(p StatsProvider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats = p
}

// InvalidateCache 使容器列表缓存失效（写操作后调用）
func (s *ContainerService) InvalidateCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = nil
	s.cacheAt = time.Time{}
}

// List 列出容器（含已停止），返回符合过滤条件的容器列表
func (s *ContainerService) List(ctx context.Context, f ListFilter) ([]*model.Container, error) {
	cached, fresh := s.cachedList()
	if !fresh {
		list, err := s.fetchList(ctx)
		if err != nil {
			return nil, err
		}
		s.storeCache(list)
		cached = list
	}

	filtered := applyFilter(cached, f)
	out := make([]*model.Container, 0, len(filtered))
	for _, c := range filtered {
		// 缓存中的对象视为只读，这里复制后再附加实时字段，避免并发请求互相覆盖
		item := copyContainer(c)
		s.attachStats(item)
		out = append(out, item)
	}
	return out, nil
}

// fetchList 从 Docker daemon 拉取容器列表并补齐 inspect 字段
func (s *ContainerService) fetchList(ctx context.Context) ([]*model.Container, error) {
	list, err := s.cli.Raw().ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, dockerErrorf("列出容器失败", err)
	}

	// 并发 inspect 补齐 exitCode / health / 时间戳等列表接口不含的字段
	results := make([]*model.Container, len(list))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for i := range list {
		wg.Add(1)
		go func(idx int, rawID string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			inspect, err := s.cli.Raw().ContainerInspect(ctx, rawID)
			if err != nil {
				// inspect 失败时退化为使用列表摘要
				results[idx] = containerFromSummary(&list[idx])
				return
			}
			c := containerFromInspect(inspect)
			// 列表接口提供的 Status 已是人类可读格式，优先采用
			if list[idx].Status != "" {
				c.Status = list[idx].Status
			}
			results[idx] = c
		}(i, list[i].ID)
	}
	wg.Wait()

	out := make([]*model.Container, 0, len(results))
	for _, c := range results {
		if c == nil {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

// copyContainer 浅拷贝容器对象
func copyContainer(c *model.Container) *model.Container {
	cp := *c
	return &cp
}

// cachedList 返回缓存的容器列表及其新鲜度
func (s *ContainerService) cachedList() ([]*model.Container, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cache == nil || time.Since(s.cacheAt) > s.cacheTTL {
		return nil, false
	}
	return s.cache, true
}

// storeCache 写入容器列表缓存
func (s *ContainerService) storeCache(list []*model.Container) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = list
	s.cacheAt = time.Now()
}

// applyFilter 在客户端应用搜索与分页过滤
func applyFilter(list []*model.Container, f ListFilter) []*model.Container {
	search := strings.ToLower(strings.TrimSpace(f.Search))
	states := make(map[string]bool, len(f.Status))
	for _, st := range f.Status {
		st = strings.ToLower(strings.TrimSpace(st))
		if st != "" {
			states[st] = true
		}
	}

	out := make([]*model.Container, 0, len(list))
	for _, c := range list {
		if len(states) > 0 && !states[strings.ToLower(c.State)] {
			continue
		}
		if search != "" {
			hit := strings.Contains(strings.ToLower(c.Name), search) ||
				strings.Contains(strings.ToLower(c.Image), search) ||
				strings.Contains(strings.ToLower(c.ID), search)
			if !hit {
				continue
			}
		}
		out = append(out, c)
	}

	if f.Offset > 0 {
		if f.Offset >= len(out) {
			return []*model.Container{}
		}
		out = out[f.Offset:]
	}
	if f.Limit > 0 && f.Limit < len(out) {
		out = out[:f.Limit]
	}
	return out
}

// containerFromSummary 在 inspect 失败时，用列表摘要构造最小可用对象
func containerFromSummary(raw *types.Container) *model.Container {
	name := ""
	if len(raw.Names) > 0 {
		name = strings.TrimPrefix(raw.Names[0], "/")
	}
	return &model.Container{
		ID:             raw.ID,
		Name:           name,
		Image:          raw.Image,
		ImageID:        raw.ImageID,
		State:          raw.State,
		Status:         raw.Status,
		Health:         "none",
		Ports:          []model.ContainerPort{},
		Networks:       []model.ContainerNetwork{},
		Mounts:         []model.ContainerMount{},
		Labels:         map[string]string{},
		ComposeProject: raw.Labels["com.docker.compose.project"],
	}
}

// Get 获取单个容器详情
func (s *ContainerService) Get(ctx context.Context, id string) (*model.Container, error) {
	inspect, err := s.cli.Raw().ContainerInspect(ctx, id)
	if err != nil {
		return nil, dockerErrorf("查询容器失败", err)
	}
	c := containerFromInspect(inspect)
	c.Status = humanStatus(inspect)
	s.attachStats(c)
	return c, nil
}

// humanStatus 生成人类可读状态，格式对齐 docker ps 输出
func humanStatus(cj types.ContainerJSON) string {
	if cj.State == nil {
		return "unknown"
	}
	st := cj.State.Status
	switch st {
	case "running":
		up := durationSince(cj.State.StartedAt)
		if cj.State.Paused {
			return "Up " + up + " (Paused)"
		}
		if cj.State.Health != nil {
			switch cj.State.Health.Status {
			case "healthy":
				return "Up " + up + " (healthy)"
			case "unhealthy":
				return "Up " + up + " (unhealthy)"
			case "starting":
				return "Up " + up + " (health: starting)"
			}
		}
		return "Up " + up
	case "exited":
		return fmt.Sprintf("Exited (%d) %s ago", cj.State.ExitCode, durationSince(cj.State.FinishedAt))
	case "created":
		return "Created"
	case "paused":
		return "Up " + durationSince(cj.State.StartedAt) + " (Paused)"
	case "restarting":
		return "Restarting (" + durationSince(cj.State.FinishedAt) + ")"
	case "dead":
		return "Dead"
	case "removing":
		return "Removing"
	default:
		return st
	}
}

// durationSince 计算距给定时间点的时长描述
func durationSince(ts string) string {
	t := parseDockerTime(ts)
	if t.IsZero() {
		return "unknown"
	}
	d := time.Since(t)
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%d seconds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d hours", int(d.Hours()))
	default:
		return fmt.Sprintf("%d days", int(d.Hours()/24))
	}
}

// InspectRaw 返回 docker inspect 的原始 JSON
func (s *ContainerService) InspectRaw(ctx context.Context, id string) ([]byte, error) {
	_, raw, err := s.cli.Raw().ContainerInspectWithRaw(ctx, id, false)
	if err != nil {
		return nil, dockerErrorf("读取容器 inspect 失败", err)
	}
	return raw, nil
}

// attachStats 从采样缓存附加实时资源字段
//
// 无采样数据时字段置为 null，避免沿用上一次采样留下的陈旧值。
func (s *ContainerService) attachStats(c *model.Container) {
	c.CPUPct = nil
	c.MemUsageBytes = nil
	c.MemLimitBytes = nil
	c.NetRxBytesPerSec = nil
	c.NetTxBytesPerSec = nil

	s.mu.RLock()
	provider := s.stats
	s.mu.RUnlock()
	if provider == nil {
		return
	}
	snap, ok := provider.ContainerStatsSnapshot(c.ID)
	if !ok {
		return
	}
	cpu := snap.CPUPct
	mem := snap.MemUsageBytes
	limit := snap.MemLimitBytes
	rx := snap.NetRxBytesPerSec
	tx := snap.NetTxBytesPerSec
	c.CPUPct = &cpu
	c.MemUsageBytes = &mem
	c.MemLimitBytes = &limit
	c.NetRxBytesPerSec = &rx
	c.NetTxBytesPerSec = &tx
}

// Start 启动容器
func (s *ContainerService) Start(ctx context.Context, id string) error {
	if err := s.cli.Raw().ContainerStart(ctx, id, container.StartOptions{}); err != nil {
		return dockerErrorf("启动容器失败", err)
	}
	s.InvalidateCache()
	return nil
}

// Stop 停止容器
func (s *ContainerService) Stop(ctx context.Context, id string) error {
	if err := s.cli.Raw().ContainerStop(ctx, id, container.StopOptions{}); err != nil {
		return dockerErrorf("停止容器失败", err)
	}
	s.InvalidateCache()
	return nil
}

// Restart 重启容器
func (s *ContainerService) Restart(ctx context.Context, id string) error {
	if err := s.cli.Raw().ContainerRestart(ctx, id, container.StopOptions{}); err != nil {
		return dockerErrorf("重启容器失败", err)
	}
	s.InvalidateCache()
	return nil
}

// Kill 强制杀死容器
func (s *ContainerService) Kill(ctx context.Context, id string) error {
	if err := s.cli.Raw().ContainerKill(ctx, id, "KILL"); err != nil {
		return dockerErrorf("杀死容器失败", err)
	}
	s.InvalidateCache()
	return nil
}

// Pause 暂停容器
func (s *ContainerService) Pause(ctx context.Context, id string) error {
	if err := s.cli.Raw().ContainerPause(ctx, id); err != nil {
		return dockerErrorf("暂停容器失败", err)
	}
	s.InvalidateCache()
	return nil
}

// Unpause 恢复容器
func (s *ContainerService) Unpause(ctx context.Context, id string) error {
	if err := s.cli.Raw().ContainerUnpause(ctx, id); err != nil {
		return dockerErrorf("恢复容器失败", err)
	}
	s.InvalidateCache()
	return nil
}

// Remove 删除容器
func (s *ContainerService) Remove(ctx context.Context, id string, force bool) error {
	if err := s.cli.Raw().ContainerRemove(ctx, id, container.RemoveOptions{Force: force}); err != nil {
		return dockerErrorf("删除容器失败", err)
	}
	s.InvalidateCache()
	return nil
}

// Prune 清理已停止容器
func (s *ContainerService) Prune(ctx context.Context, all bool) (*model.PruneResult, error) {
	args := filters.NewArgs()
	if all {
		args.Add("all", "true")
	}
	report, err := s.cli.Raw().ContainersPrune(ctx, args)
	if err != nil {
		return nil, dockerErrorf("清理容器失败", err)
	}
	deleted := make([]string, 0, len(report.ContainersDeleted))
	for _, id := range report.ContainersDeleted {
		if id != "" {
			deleted = append(deleted, id)
		}
	}
	s.InvalidateCache()
	return &model.PruneResult{Deleted: deleted, SpaceReclaimedBytes: int64(report.SpaceReclaimed)}, nil
}

// LogOptions 容器日志选项
type LogOptions struct {
	Tail       string
	Timestamps bool
	Follow     bool
}

// Logs 打开容器日志流（调用方负责关闭）。
//
// 返回的是剥离了 Docker 多路复用帧头的纯文本，负载中的 ANSI 颜色转义原样保留。
func (s *ContainerService) Logs(ctx context.Context, id string, opts LogOptions) (io.ReadCloser, error) {
	rc, err := s.cli.Raw().ContainerLogs(ctx, id, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: opts.Timestamps,
		Follow:     opts.Follow,
		Tail:       opts.Tail,
	})
	if err != nil {
		return nil, dockerErrorf("读取容器日志失败", err)
	}
	return demuxLogs(rc), nil
}

// Stats 读取容器一次性资源快照，网络与磁盘 IO 已换算为每秒速率。
func (s *ContainerService) Stats(ctx context.Context, id string) (*model.ContainerStats, error) {
	sr, err := s.rawStats(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.toRate(id, sr), nil
}

// statsCounter 容器累计计数快照，用于换算速率
type statsCounter struct {
	at         time.Time
	rx         uint64
	tx         uint64
	blockRead  uint64
	blockWrite uint64
}

// rawStats 读取容器原始累计统计
func (s *ContainerService) rawStats(ctx context.Context, id string) (*container.StatsResponse, error) {
	resp, err := s.cli.Raw().ContainerStatsOneShot(ctx, id)
	if err != nil {
		return nil, dockerErrorf("读取容器资源失败", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var sr container.StatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, fmt.Errorf("解析容器资源数据失败: %w", err)
	}
	return &sr, nil
}

// toRate 把累计计数换算为每秒速率
func (s *ContainerService) toRate(id string, sr *container.StatsResponse) *model.ContainerStats {
	var rx, tx, blockRead, blockWrite uint64
	for _, net := range sr.Networks {
		rx += net.RxBytes
		tx += net.TxBytes
	}
	for _, entry := range sr.BlkioStats.IoServiceBytesRecursive {
		switch strings.ToLower(entry.Op) {
		case "read":
			blockRead += entry.Value
		case "write":
			blockWrite += entry.Value
		}
	}

	now := time.Now()
	s.mu.Lock()
	prev, hasPrev := s.statsCounters[id]
	s.statsCounters[id] = statsCounter{at: now, rx: rx, tx: tx, blockRead: blockRead, blockWrite: blockWrite}
	s.mu.Unlock()

	out := &model.ContainerStats{
		CPUPct:        calculateCPUPercent(&sr.CPUStats, &sr.PreCPUStats, sr.CPUStats.OnlineCPUs),
		MemUsageBytes: int64(sr.MemoryStats.Usage),
		MemLimitBytes: int64(sr.MemoryStats.Limit),
	}

	if hasPrev {
		if dt := now.Sub(prev.at).Seconds(); dt > 0 {
			out.NetRxBytesPerSec = int64(float64(deltaUint(rx, prev.rx)) / dt)
			out.NetTxBytesPerSec = int64(float64(deltaUint(tx, prev.tx)) / dt)
			out.BlockReadBytesPerSec = int64(float64(deltaUint(blockRead, prev.blockRead)) / dt)
			out.BlockWriteBytesPerSec = int64(float64(deltaUint(blockWrite, prev.blockWrite)) / dt)
		}
	}
	return out
}

// deltaUint 计算累计值增量（处理计数器回绕）
func deltaUint(cur, prev uint64) uint64 {
	if cur >= prev {
		return cur - prev
	}
	return cur
}

// calculateCPUPercent 计算容器 CPU 使用率（0-100）
func calculateCPUPercent(cur, pre *container.CPUStats, onlineCPUs uint32) float64 {
	cpuDelta := float64(cur.CPUUsage.TotalUsage) - float64(pre.CPUUsage.TotalUsage)
	systemDelta := float64(cur.SystemUsage) - float64(pre.SystemUsage)
	if cpuDelta <= 0 || systemDelta <= 0 {
		return 0
	}
	cpus := float64(onlineCPUs)
	if cpus == 0 {
		cpus = float64(runtime.NumCPU())
	}
	if cpus == 0 {
		cpus = 1
	}
	return (cpuDelta / systemDelta) * cpus * 100.0
}

// ExecOptions exec 会话选项
type ExecOptions struct {
	Cmd     []string
	TTY     bool
	User    string
	Env     []string
	WorkDir string
}

// ExecSession exec 会话
type ExecSession struct {
	ID string
}

// ExecCreate 创建 exec 会话
func (s *ContainerService) ExecCreate(ctx context.Context, containerID string, opts ExecOptions) (*ExecSession, error) {
	resp, err := s.cli.Raw().ContainerExecCreate(ctx, containerID, container.ExecOptions{
		Cmd:          opts.Cmd,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          opts.TTY,
		User:         opts.User,
		Env:          opts.Env,
		WorkingDir:   opts.WorkDir,
	})
	if err != nil {
		return nil, dockerErrorf("创建 exec 会话失败", err)
	}
	return &ExecSession{ID: resp.ID}, nil
}

// ExecAttach 附加到 exec 会话，返回双向流（调用方负责关闭）
func (s *ContainerService) ExecAttach(ctx context.Context, execID string) (*types.HijackedResponse, error) {
	resp, err := s.cli.Raw().ContainerExecAttach(ctx, execID, container.ExecAttachOptions{
		Tty: true,
	})
	if err != nil {
		return nil, dockerErrorf("附加 exec 会话失败", err)
	}
	return &resp, nil
}

// ExecResize 调整 exec 终端尺寸
func (s *ContainerService) ExecResize(ctx context.Context, execID string, width, height uint) error {
	if err := s.cli.Raw().ContainerExecResize(ctx, execID, container.ResizeOptions{Width: width, Height: height}); err != nil {
		return dockerErrorf("调整终端尺寸失败", err)
	}
	return nil
}

// Update 拉取新镜像并以原配置重建容器
func (s *ContainerService) Update(ctx context.Context, id string) (*model.UpdateResult, error) {
	inspect, err := s.cli.Raw().ContainerInspect(ctx, id)
	if err != nil {
		return nil, dockerErrorf("查询容器失败", err)
	}
	if inspect.Config == nil || inspect.Config.Image == "" {
		return nil, errors.New("容器缺少镜像信息，无法更新")
	}

	oldImage := inspect.Config.Image
	newImage := oldImage

	// 拉取新镜像（镜像没有 tag 时无法拉取，直接跳过）
	if strings.Contains(lastSegment(oldImage), ":") {
		if _, err := s.PullImage(ctx, oldImage); err != nil {
			return nil, err
		}
		newImage = oldImage
	}

	// 停止旧容器
	_ = s.cli.Raw().ContainerStop(ctx, id, container.StopOptions{})

	// 改名腾出原名
	origName := strings.TrimPrefix(inspect.Name, "/")
	backupName := fmt.Sprintf("%s_old_%d", origName, time.Now().Unix())
	if err := s.cli.Raw().ContainerRename(ctx, id, backupName); err != nil {
		return nil, dockerErrorf("重命名旧容器失败", err)
	}

	// 以原配置重建
	newID, err := s.recreate(ctx, inspect, origName)
	if err != nil {
		// 回滚：把旧容器名字改回来
		_ = s.cli.Raw().ContainerRename(ctx, id, origName)
		return nil, err
	}

	if err := s.cli.Raw().ContainerStart(ctx, newID, container.StartOptions{}); err != nil {
		_ = s.cli.Raw().ContainerRemove(ctx, newID, container.RemoveOptions{Force: true})
		_ = s.cli.Raw().ContainerRename(ctx, id, origName)
		return nil, dockerErrorf("启动新容器失败", err)
	}

	// 删除旧容器
	_ = s.cli.Raw().ContainerRemove(ctx, id, container.RemoveOptions{Force: true})
	s.InvalidateCache()

	return &model.UpdateResult{OK: true, OldImage: oldImage, NewImage: newImage}, nil
}

// recreate 用给定配置创建新容器
func (s *ContainerService) recreate(ctx context.Context, inspect types.ContainerJSON, name string) (string, error) {
	cfg := inspect.Config
	hc := inspect.HostConfig

	netCfg := &network.NetworkingConfig{EndpointsConfig: map[string]*network.EndpointSettings{}}
	if inspect.NetworkSettings != nil {
		for netName, ep := range inspect.NetworkSettings.Networks {
			if ep == nil {
				continue
			}
			netCfg.EndpointsConfig[netName] = &network.EndpointSettings{
				IPAMConfig:          ep.IPAMConfig,
				Links:               ep.Links,
				Aliases:             ep.Aliases,
				NetworkID:           ep.NetworkID,
				EndpointID:          ep.EndpointID,
				Gateway:             ep.Gateway,
				IPAddress:           ep.IPAddress,
				IPPrefixLen:         ep.IPPrefixLen,
				IPv6Gateway:         ep.IPv6Gateway,
				GlobalIPv6Address:   ep.GlobalIPv6Address,
				GlobalIPv6PrefixLen: ep.GlobalIPv6PrefixLen,
				MacAddress:          ep.MacAddress,
				DriverOpts:          ep.DriverOpts,
			}
		}
	}

	// 重建时清空运行时产生、不可再次指定的字段
	newCfg := *cfg
	newCfg.Hostname = ""
	if hc != nil {
		newHC := *hc
		newHC.ContainerIDFile = ""
		hc = &newHC
	}

	var platform *ocispec.Platform
	resp, err := s.cli.Raw().ContainerCreate(ctx, &newCfg, hc, netCfg, platform, name)
	if err != nil {
		return "", dockerErrorf("重建容器失败", err)
	}
	return resp.ID, nil
}

// lastSegment 取路径最后一段
func lastSegment(s string) string {
	if i := strings.LastIndex(s, "/"); i >= 0 {
		return s[i+1:]
	}
	return s
}

// PullImage 拉取镜像，返回镜像 ID
func (s *ContainerService) PullImage(ctx context.Context, ref string) (string, error) {
	return pullImage(ctx, s.cli, ref)
}
