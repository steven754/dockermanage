package stats_test

import (
	"context"
	"testing"
	"time"

	"github.com/steven754/dockermanage/backend/internal/db"
	"github.com/steven754/dockermanage/backend/internal/model"
	"github.com/steven754/dockermanage/backend/internal/repo"
	"github.com/steven754/dockermanage/backend/internal/service/docker"
	"github.com/steven754/dockermanage/backend/internal/service/stats"
)

// fakeContainerProvider 可控的容器来源
type fakeContainerProvider struct {
	listErr    error
	statErr    error
	containers []*model.Container
	stats      map[string]*model.ContainerStats
}

func (f *fakeContainerProvider) List(ctx context.Context, fl docker.ListFilter) ([]*model.Container, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.containers, nil
}

func (f *fakeContainerProvider) Stats(ctx context.Context, id string) (*model.ContainerStats, error) {
	if f.statErr != nil {
		return nil, f.statErr
	}
	if s, ok := f.stats[id]; ok {
		return s, nil
	}
	return &model.ContainerStats{}, nil
}

// newRealSampler 构造带真实仓库的采样器
func newRealSampler(t *testing.T, collector stats.Collector, interval time.Duration) *stats.Sampler {
	t.Helper()

	sqlDB, err := db.Open("", true)
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.Migrate(sqlDB); err != nil {
		t.Fatalf("执行迁移失败: %v", err)
	}
	store := repo.New(sqlDB)
	if collector == nil {
		collector = &fakeCollector{}
	}
	return stats.New(store.SystemStats, store.ContainerStats, collector, interval, time.Hour)
}

func TestSamplerRunStopsOnContextCancel(t *testing.T) {
	collector := &fakeCollector{sample: model.SystemSample{CPUPct: 1}}
	sampler := newRealSampler(t, collector, 100*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		sampler.Run(ctx)
		close(done)
	}()

	time.Sleep(350 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("采样器未在上下文取消后退出")
	}

	if collector.calls < 2 {
		t.Fatalf("期望至少 2 次采样，实际 %d", collector.calls)
	}
}

func TestContainerSamplerCollectsRunningContainers(t *testing.T) {
	sampler := newRealSampler(t, nil, time.Second)

	provider := &fakeContainerProvider{
		containers: []*model.Container{
			{ID: "abc", Name: "web", State: "running"},
			{ID: "def", Name: "db", State: "exited"},
		},
		stats: map[string]*model.ContainerStats{
			"abc": {CPUPct: 12.5, MemUsageBytes: 1024},
		},
	}
	cs := stats.NewContainerSampler(provider, sampler, 100*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		cs.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("容器采样器未退出")
	}

	// 只有 running 的容器应被采样
	snap, ok := sampler.ContainerStatsSnapshot("abc")
	if !ok {
		t.Fatal("运行中的容器应有采样")
	}
	if snap.CPUPct != 12.5 {
		t.Fatalf("期望 CPU 12.5，实际 %f", snap.CPUPct)
	}
	if _, ok := sampler.ContainerStatsSnapshot("def"); ok {
		t.Fatal("已停止的容器不应有采样")
	}
}

func TestContainerSamplerHandlesListError(t *testing.T) {
	sampler := newRealSampler(t, nil, time.Second)

	provider := &fakeContainerProvider{listErr: context.DeadlineExceeded}
	cs := stats.NewContainerSampler(provider, sampler, 100*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		cs.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("容器采样器未退出")
	}
}
