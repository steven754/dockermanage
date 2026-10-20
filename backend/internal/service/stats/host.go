package stats

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"

	"github.com/steven754/dockermanage/backend/internal/model"
)

// HostCollector 基于 gopsutil 采集宿主机资源
type HostCollector struct {
	diskPath string

	mu     sync.Mutex
	prevRx uint64
	prevTx uint64
	prevAt time.Time
}

// NewHostCollector 创建宿主机采集器，diskPath 为需要统计的挂载点
func NewHostCollector(diskPath string) *HostCollector {
	if diskPath == "" {
		diskPath = "/"
	}
	return &HostCollector{diskPath: diskPath}
}

// Collect 采集一次宿主机资源，返回系统采样与网络速率
func (c *HostCollector) Collect(ctx context.Context) (model.SystemSample, int64, int64) {
	sample := model.SystemSample{TS: model.FormatTime(time.Now())}

	// CPU：gopsutil 内部保存上次采样，与本次做差
	if percents, err := cpu.Percent(0, false); err == nil && len(percents) > 0 {
		sample.CPUPct = percents[0]
	} else {
		slog.Debug("采集 CPU 使用率失败", "error", err)
	}

	// 内存
	if vm, err := mem.VirtualMemory(); err == nil {
		sample.MemUsedBytes = int64(vm.Used)
		sample.MemTotalBytes = int64(vm.Total)
	} else {
		slog.Debug("采集内存使用率失败", "error", err)
	}

	// 磁盘
	if du, err := disk.Usage(c.diskPath); err == nil {
		sample.DiskUsedBytes = int64(du.Used)
		sample.DiskTotalBytes = int64(du.Total)
	} else {
		slog.Debug("采集磁盘使用率失败", "path", c.diskPath, "error", err)
	}

	// 网络速率（累计值做差）
	rxRate, txRate := c.networkRate()
	sample.NetRxBytesPerSec = rxRate
	sample.NetTxBytesPerSec = txRate

	return sample, rxRate, txRate
}

// networkRate 计算网络收发速率
func (c *HostCollector) networkRate() (int64, int64) {
	counters, err := net.IOCounters(false)
	if err != nil || len(counters) == 0 {
		slog.Debug("采集网络计数器失败", "error", err)
		return 0, 0
	}
	var totalRx, totalTx uint64
	for _, ct := range counters {
		totalRx += ct.BytesRecv
		totalTx += ct.BytesSent
	}

	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()

	prevRx, prevTx, prevAt := c.prevRx, c.prevTx, c.prevAt
	c.prevRx, c.prevTx, c.prevAt = totalRx, totalTx, now

	if prevAt.IsZero() {
		return 0, 0
	}
	dt := now.Sub(prevAt).Seconds()
	if dt <= 0 {
		return 0, 0
	}
	return int64(float64(deltaUint64(totalRx, prevRx)) / dt),
		int64(float64(deltaUint64(totalTx, prevTx)) / dt)
}

// deltaUint64 计算累计值增量
func deltaUint64(cur, prev uint64) uint64 {
	if cur >= prev {
		return cur - prev
	}
	return cur
}

// Uptime 返回宿主机运行时长（秒）
func Uptime() uint64 {
	up, err := host.Uptime()
	if err != nil {
		slog.Debug("读取主机运行时长失败", "error", err)
		return 0
	}
	return up
}

// HostName 返回主机名
func HostName() string {
	name, err := os.Hostname()
	if err != nil {
		return "localhost"
	}
	return name
}
