// Package stats 负责主机与容器资源的周期性采样、历史持久化与实时广播。
package stats

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/steven754/dockermanage/backend/internal/model"
	"github.com/steven754/dockermanage/backend/internal/repo"
	"github.com/steven754/dockermanage/backend/internal/service/broker"
)

// EventType 广播消息类型
type EventType string

const (
	// EventTypeSystem 系统级采样
	EventTypeSystem EventType = "system"
	// EventTypeContainer 容器级采样
	EventTypeContainer EventType = "container"
)

// Message 广播消息
type Message struct {
	Type EventType
	// System 系统级采样，Type 为 EventTypeSystem 时有效
	System model.SystemSample
	// Container 容器级采样，Type 为 EventTypeContainer 时有效
	Container model.ContainerSample
}

// Sampler 采样器
type Sampler struct {
	systemRepo    *repo.SystemStatsRepo
	containerRepo *repo.ContainerStatsRepo
	collector     Collector
	broker        *broker.Broker[Message]

	interval  time.Duration
	retention time.Duration

	mu                sync.RWMutex
	containerSnapshot map[string]model.ContainerStats
	names             map[string]string
	lastSample        model.SystemSample
	hasSample         bool
}

// Collector 采集宿主机的系统资源
type Collector interface {
	// Collect 返回本次采样的系统资源与网络速率
	Collect(ctx context.Context) (model.SystemSample, int64, int64)
}

// New 创建采样器
func New(
	systemRepo *repo.SystemStatsRepo,
	containerRepo *repo.ContainerStatsRepo,
	collector Collector,
	interval time.Duration,
	retention time.Duration,
) *Sampler {
	if interval <= 0 {
		interval = 10 * time.Second
	}
	if retention <= 0 {
		retention = 7 * 24 * time.Hour
	}
	return &Sampler{
		systemRepo:        systemRepo,
		containerRepo:     containerRepo,
		collector:         collector,
		broker:            broker.New[Message](),
		interval:          interval,
		retention:         retention,
		containerSnapshot: map[string]model.ContainerStats{},
		names:             map[string]string{},
	}
}

// Broker 返回广播中心
func (s *Sampler) Broker() *broker.Broker[Message] { return s.broker }

// LatestSystemSample 返回最近一次系统采样
func (s *Sampler) LatestSystemSample() (model.SystemSample, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastSample, s.hasSample
}

// ContainerStatsSnapshot 返回容器采样缓存
func (s *Sampler) ContainerStatsSnapshot(id string) (model.ContainerStats, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snap, ok := s.containerSnapshot[id]
	return snap, ok
}

// ContainerName 返回容器名
func (s *Sampler) ContainerName(id string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.names[id]
}

// Run 启动采样循环，直到 ctx 取消
func (s *Sampler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// 启动后先立刻采一次，避免前端等待一个周期
	s.SampleOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			slog.Info("采样器已停止")
			return
		case <-ticker.C:
			s.SampleOnce(ctx)
		}
	}
}

// SampleOnce 执行一次采样
func (s *Sampler) SampleOnce(ctx context.Context) {
	sample, rxRate, txRate := s.collector.Collect(ctx)
	sample.NetRxBytesPerSec = rxRate
	sample.NetTxBytesPerSec = txRate

	ts := time.Now().Unix()
	record := repo.SystemStat{
		TS:               ts,
		CPUPct:           sample.CPUPct,
		MemUsedBytes:     sample.MemUsedBytes,
		MemTotalBytes:    sample.MemTotalBytes,
		DiskUsedBytes:    sample.DiskUsedBytes,
		DiskTotalBytes:   sample.DiskTotalBytes,
		NetRxBytesPerSec: rxRate,
		NetTxBytesPerSec: txRate,
	}
	if err := s.systemRepo.Insert(record); err != nil {
		slog.Error("写入系统采样失败", "error", err)
	}

	s.broker.Publish(Message{Type: EventTypeSystem, System: sample})

	s.mu.Lock()
	s.lastSample = sample
	s.hasSample = true
	s.mu.Unlock()
}

// Cleanup 清理超过保留期的历史数据
func (s *Sampler) Cleanup(ctx context.Context) {
	cutoff := time.Now().Add(-s.retention).Unix()
	if n, err := s.systemRepo.DeleteBefore(cutoff); err != nil {
		slog.Error("清理系统采样失败", "error", err)
	} else if n > 0 {
		slog.Info("已清理系统采样", "rows", n)
	}
	if n, err := s.containerRepo.DeleteBefore(cutoff); err != nil {
		slog.Error("清理容器采样失败", "error", err)
	} else if n > 0 {
		slog.Info("已清理容器采样", "rows", n)
	}
}

// History 系统级历史曲线
func (s *Sampler) History(ctx context.Context, rangeKey string) (*model.StatsHistory, error) {
	start, _, err := repo.RangeStart(rangeKey)
	if err != nil {
		return nil, err
	}
	rows, err := s.systemRepo.ListSince(start)
	if err != nil {
		return nil, err
	}
	points := make([]model.StatsPoint, 0, len(rows))
	for _, r := range rows {
		points = append(points, model.StatsPoint{
			TS:               model.FormatTime(time.Unix(r.TS, 0)),
			CPUPct:           r.CPUPct,
			MemUsageBytes:    r.MemUsedBytes,
			NetRxBytesPerSec: r.NetRxBytesPerSec,
			NetTxBytesPerSec: r.NetTxBytesPerSec,
		})
	}
	return &model.StatsHistory{
		Range:       rangeKey,
		StepSeconds: int(s.interval.Seconds()),
		Points:      points,
	}, nil
}

// ContainerHistory 容器级历史曲线
func (s *Sampler) ContainerHistory(ctx context.Context, rangeKey, containerID string) (*model.StatsHistory, error) {
	start, _, err := repo.RangeStart(rangeKey)
	if err != nil {
		return nil, err
	}
	rows, err := s.containerRepo.ListSince(start, containerID)
	if err != nil {
		return nil, err
	}
	points := make([]model.StatsPoint, 0, len(rows))
	for _, r := range rows {
		points = append(points, model.StatsPoint{
			TS:               model.FormatTime(time.Unix(r.TS, 0)),
			CPUPct:           r.CPUPct,
			MemUsageBytes:    r.MemUsedBytes,
			NetRxBytesPerSec: r.NetRxBytesPerSec,
			NetTxBytesPerSec: r.NetTxBytesPerSec,
		})
	}
	return &model.StatsHistory{
		Range:       rangeKey,
		StepSeconds: int(s.interval.Seconds()),
		Points:      points,
	}, nil
}

// RecordContainers 记录容器级采样并广播
func (s *Sampler) RecordContainers(ctx context.Context, list []model.ContainerSample) {
	if len(list) == 0 {
		return
	}
	ts := time.Now().Unix()
	records := make([]repo.ContainerStat, 0, len(list))
	for _, c := range list {
		records = append(records, repo.ContainerStat{
			TS:                    ts,
			ContainerID:           c.ID,
			ContainerName:         c.Name,
			CPUPct:                c.CPUPct,
			MemUsedBytes:          c.MemUsageBytes,
			MemLimitBytes:         c.MemLimitBytes,
			NetRxBytesPerSec:      c.NetRxBytesPerSec,
			NetTxBytesPerSec:      c.NetTxBytesPerSec,
			BlockReadBytesPerSec:  c.BlockReadBytesPerSec,
			BlockWriteBytesPerSec: c.BlockWriteBytesPerSec,
		})
	}
	if err := s.containerRepo.InsertBatch(records); err != nil {
		slog.Error("写入容器采样失败", "error", err)
	}

	s.mu.Lock()
	for _, c := range list {
		s.containerSnapshot[c.ID] = model.ContainerStats{
			CPUPct:                c.CPUPct,
			MemUsageBytes:         c.MemUsageBytes,
			MemLimitBytes:         c.MemLimitBytes,
			NetRxBytesPerSec:      c.NetRxBytesPerSec,
			NetTxBytesPerSec:      c.NetTxBytesPerSec,
			BlockReadBytesPerSec:  c.BlockReadBytesPerSec,
			BlockWriteBytesPerSec: c.BlockWriteBytesPerSec,
		}
		s.names[c.ID] = c.Name
	}
	s.mu.Unlock()

	for _, c := range list {
		s.broker.Publish(Message{Type: EventTypeContainer, Container: c})
	}
}
