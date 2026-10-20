package stats_test

import (
	"context"
	"testing"
	"time"

	"github.com/steven754/dockermanage/backend/internal/db"
	"github.com/steven754/dockermanage/backend/internal/model"
	"github.com/steven754/dockermanage/backend/internal/repo"
	"github.com/steven754/dockermanage/backend/internal/service/stats"
)

// fakeCollector 可控的假采集器
type fakeCollector struct {
	sample model.SystemSample
	calls  int
}

func (f *fakeCollector) Collect(ctx context.Context) (model.SystemSample, int64, int64) {
	f.calls++
	return f.sample, 100, 200
}

// newTestSampler 构造带内存库的采样器
func newTestSampler(t *testing.T, collector stats.Collector) *stats.Sampler {
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

	return stats.New(store.SystemStats, store.ContainerStats, collector, time.Second, 7*24*time.Hour)
}

func TestSamplerRecordsAndBroadcasts(t *testing.T) {
	collector := &fakeCollector{sample: model.SystemSample{
		CPUPct:        25.5,
		MemUsedBytes:  1000,
		MemTotalBytes: 4000,
	}}
	sampler := newTestSampler(t, collector)

	ch, cancel := sampler.Broker().Subscribe(4)
	defer cancel()

	sampler.SampleOnce(t.Context())

	select {
	case msg := <-ch:
		if msg.Type != stats.EventTypeSystem {
			t.Fatalf("期望系统事件，实际 %s", msg.Type)
		}
		if msg.System.CPUPct != 25.5 {
			t.Fatalf("期望 CPU 25.5，实际 %f", msg.System.CPUPct)
		}
		if msg.System.NetRxBytesPerSec != 100 {
			t.Fatalf("期望网络速率 100，实际 %d", msg.System.NetRxBytesPerSec)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("未收到系统采样广播")
	}

	// 最近采样应被缓存
	sample, ok := sampler.LatestSystemSample()
	if !ok {
		t.Fatal("应缓存最近一次采样")
	}
	if sample.CPUPct != 25.5 {
		t.Fatalf("期望 CPU 25.5，实际 %f", sample.CPUPct)
	}
}

func TestSamplerContainerSnapshot(t *testing.T) {
	sampler := newTestSampler(t, &fakeCollector{})

	sampler.RecordContainers(t.Context(), []model.ContainerSample{
		{ID: "abc", Name: "web", CPUPct: 1.5, MemUsageBytes: 100},
	})

	snap, ok := sampler.ContainerStatsSnapshot("abc")
	if !ok {
		t.Fatal("应能取到容器采样")
	}
	if snap.CPUPct != 1.5 {
		t.Fatalf("期望 CPU 1.5，实际 %f", snap.CPUPct)
	}
	if sampler.ContainerName("abc") != "web" {
		t.Fatalf("期望容器名 web，实际 %s", sampler.ContainerName("abc"))
	}

	if _, ok := sampler.ContainerStatsSnapshot("not-exist"); ok {
		t.Fatal("不存在的容器不应有采样")
	}
}

func TestSamplerHistory(t *testing.T) {
	sampler := newTestSampler(t, &fakeCollector{})

	// 写入两个采样点
	sampler.SampleOnce(t.Context())
	time.Sleep(1100 * time.Millisecond)
	sampler.SampleOnce(t.Context())

	history, err := sampler.History(t.Context(), "1h")
	if err != nil {
		t.Fatalf("查询历史失败: %v", err)
	}
	if history.Range != "1h" {
		t.Fatalf("期望 range 1h，实际 %s", history.Range)
	}
	if len(history.Points) == 0 {
		t.Fatal("历史曲线不应为空")
	}
	if history.StepSeconds != 1 {
		t.Fatalf("期望 stepSeconds 1，实际 %d", history.StepSeconds)
	}

	// 非法 range
	if _, err := sampler.History(t.Context(), "3h"); err == nil {
		t.Fatal("非法 range 应返回错误")
	}
}

func TestSamplerContainerHistory(t *testing.T) {
	sampler := newTestSampler(t, &fakeCollector{})

	sampler.RecordContainers(t.Context(), []model.ContainerSample{
		{ID: "abc", Name: "web", CPUPct: 1.5},
	})

	history, err := sampler.ContainerHistory(t.Context(), "1h", "abc")
	if err != nil {
		t.Fatalf("查询容器历史失败: %v", err)
	}
	if len(history.Points) == 0 {
		t.Fatal("容器历史曲线不应为空")
	}
}

func TestSamplerCleanup(t *testing.T) {
	sampler := newTestSampler(t, &fakeCollector{})
	sampler.SampleOnce(t.Context())

	// 保留期内的数据不应被清理
	sampler.Cleanup(t.Context())

	history, err := sampler.History(t.Context(), "1h")
	if err != nil {
		t.Fatalf("查询历史失败: %v", err)
	}
	if len(history.Points) == 0 {
		t.Fatal("保留期内的采样不应被清理")
	}
}
