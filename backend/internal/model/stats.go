package model

// SystemSample 系统级采样（SSE event: system）
type SystemSample struct {
	TS               string  `json:"ts"`
	CPUPct           float64 `json:"cpuPct"`
	MemUsedBytes     int64   `json:"memUsedBytes"`
	MemTotalBytes    int64   `json:"memTotalBytes"`
	DiskUsedBytes    int64   `json:"diskUsedBytes"`
	DiskTotalBytes   int64   `json:"diskTotalBytes"`
	NetRxBytesPerSec int64   `json:"netRxBytesPerSec"`
	NetTxBytesPerSec int64   `json:"netTxBytesPerSec"`
}

// ContainerSample 容器级采样（SSE event: container）
type ContainerSample struct {
	ID                    string  `json:"id"`
	Name                  string  `json:"name"`
	CPUPct                float64 `json:"cpuPct"`
	MemUsageBytes         int64   `json:"memUsageBytes"`
	MemLimitBytes         int64   `json:"memLimitBytes"`
	NetRxBytesPerSec      int64   `json:"netRxBytesPerSec"`
	NetTxBytesPerSec      int64   `json:"netTxBytesPerSec"`
	BlockReadBytesPerSec  int64   `json:"blockReadBytesPerSec"`
	BlockWriteBytesPerSec int64   `json:"blockWriteBytesPerSec"`
}

// StatsPoint 历史曲线上的一个采样点
type StatsPoint struct {
	TS               string  `json:"ts"`
	CPUPct           float64 `json:"cpuPct"`
	MemUsageBytes    int64   `json:"memUsageBytes"`
	NetRxBytesPerSec int64   `json:"netRxBytesPerSec"`
	NetTxBytesPerSec int64   `json:"netTxBytesPerSec"`
}

// StatsHistory 历史指标响应
type StatsHistory struct {
	Range       string       `json:"range"`
	StepSeconds int          `json:"stepSeconds"`
	Points      []StatsPoint `json:"points"`
}
