package stats

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/steven754/dockermanage/backend/internal/model"
	"github.com/steven754/dockermanage/backend/internal/service/docker"
)

// ContainerStatsProvider 提供容器列表与资源采样能力。
//
// 由 docker.ContainerService 实现，抽象成接口以便单测替换。
type ContainerStatsProvider interface {
	// List 列出容器
	List(ctx context.Context, f docker.ListFilter) ([]*model.Container, error)
	// Stats 读取容器资源快照
	Stats(ctx context.Context, id string) (*model.ContainerStats, error)
}

// ContainerSampler 周期性采集运行中容器的资源占用
type ContainerSampler struct {
	containers  ContainerStatsProvider
	sampler     *Sampler
	interval    time.Duration
	concurrency int
}

// NewContainerSampler 创建容器采样器
func NewContainerSampler(containers ContainerStatsProvider, sampler *Sampler, interval time.Duration) *ContainerSampler {
	if interval <= 0 {
		interval = 10 * time.Second
	}
	return &ContainerSampler{
		containers:  containers,
		sampler:     sampler,
		interval:    interval,
		concurrency: 4,
	}
}

// Run 启动容器采样循环，直到 ctx 取消
func (s *ContainerSampler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("容器采样器已停止")
			return
		case <-ticker.C:
			s.sample(ctx)
		}
	}
}

// sample 采集一次容器资源
func (s *ContainerSampler) sample(ctx context.Context) {
	containers, err := s.containers.List(ctx, docker.ListFilter{})
	if err != nil {
		slog.Error("列出容器失败，跳过本轮容器采样", "error", err)
		return
	}

	running := make([]*model.Container, 0, len(containers))
	for _, c := range containers {
		if c.State == "running" {
			running = append(running, c)
		}
	}
	if len(running) == 0 {
		return
	}

	samples := make([]model.ContainerSample, len(running))
	var wg sync.WaitGroup
	sem := make(chan struct{}, s.concurrency)
	for i, c := range running {
		wg.Add(1)
		go func(idx int, id, name string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			stat, err := s.containers.Stats(ctx, id)
			if err != nil {
				slog.Debug("采集容器资源失败", "container", name, "error", err)
				return
			}
			samples[idx] = model.ContainerSample{
				ID:                    id,
				Name:                  name,
				CPUPct:                stat.CPUPct,
				MemUsageBytes:         stat.MemUsageBytes,
				MemLimitBytes:         stat.MemLimitBytes,
				NetRxBytesPerSec:      stat.NetRxBytesPerSec,
				NetTxBytesPerSec:      stat.NetTxBytesPerSec,
				BlockReadBytesPerSec:  stat.BlockReadBytesPerSec,
				BlockWriteBytesPerSec: stat.BlockWriteBytesPerSec,
			}
		}(i, c.ID, c.Name)
	}
	wg.Wait()

	valid := make([]model.ContainerSample, 0, len(samples))
	for _, sm := range samples {
		if sm.ID != "" {
			valid = append(valid, sm)
		}
	}
	s.sampler.RecordContainers(ctx, valid)
}
