package stats_test

import (
	"testing"
	"time"

	"github.com/steven754/dockermanage/backend/internal/service/stats"
)

func TestHostCollectorCollect(t *testing.T) {
	collector := stats.NewHostCollector("/")

	sample, rxRate, txRate := collector.Collect(t.Context())
	if sample.MemTotalBytes <= 0 {
		t.Fatalf("内存总量应大于 0，实际 %d", sample.MemTotalBytes)
	}
	if sample.DiskTotalBytes <= 0 {
		t.Fatalf("磁盘总量应大于 0，实际 %d", sample.DiskTotalBytes)
	}
	if sample.CPUPct < 0 || sample.CPUPct > 100 {
		t.Fatalf("CPU 使用率应在 0-100 之间，实际 %f", sample.CPUPct)
	}
	if sample.TS == "" {
		t.Fatal("采样时间不应为空")
	}

	// 首次采集的网络速率应为 0（没有上一次基线）
	if rxRate != 0 || txRate != 0 {
		t.Fatalf("首次采集网络速率应为 0，实际 rx=%d tx=%d", rxRate, txRate)
	}

	// 第二次采集应能算出速率
	time.Sleep(100 * time.Millisecond)
	_, rx2, tx2 := collector.Collect(t.Context())
	if rx2 < 0 || tx2 < 0 {
		t.Fatalf("网络速率不应为负数: rx=%d tx=%d", rx2, tx2)
	}
}

func TestHostCollectorDefaultDiskPath(t *testing.T) {
	collector := stats.NewHostCollector("")
	sample, _, _ := collector.Collect(t.Context())
	if sample.DiskTotalBytes <= 0 {
		t.Fatalf("默认磁盘路径应能采集到容量，实际 %d", sample.DiskTotalBytes)
	}
}

func TestUptimeAndHostName(t *testing.T) {
	if stats.Uptime() == 0 {
		t.Fatal("主机运行时长应大于 0")
	}
	if stats.HostName() == "" {
		t.Fatal("主机名不应为空")
	}
}
