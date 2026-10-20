package model

// Container 容器摘要对象
type Container struct {
	ID             string             `json:"id"`
	Name           string             `json:"name"`
	Image          string             `json:"image"`
	ImageID        string             `json:"imageId"`
	State          string             `json:"state"`
	Status         string             `json:"status"`
	Health         string             `json:"health"`
	ExitCode       int                `json:"exitCode"`
	CreatedAt      string             `json:"createdAt"`
	StartedAt      string             `json:"startedAt"`
	FinishedAt     *string            `json:"finishedAt"`
	RestartCount   int                `json:"restartCount"`
	Ports          []ContainerPort    `json:"ports"`
	Networks       []ContainerNetwork `json:"networks"`
	Mounts         []ContainerMount   `json:"mounts"`
	Labels         map[string]string  `json:"labels"`
	ComposeProject string             `json:"composeProject"`

	// 以下为采样器缓存的实时字段，可能为 null
	CPUPct           *float64 `json:"cpuPct"`
	MemUsageBytes    *int64   `json:"memUsageBytes"`
	MemLimitBytes    *int64   `json:"memLimitBytes"`
	NetRxBytesPerSec *int64   `json:"netRxBytesPerSec"`
	NetTxBytesPerSec *int64   `json:"netTxBytesPerSec"`
}

// ContainerPort 端口映射
type ContainerPort struct {
	PrivatePort uint16 `json:"privatePort"`
	PublicPort  uint16 `json:"publicPort"`
	Type        string `json:"type"`
	IP          string `json:"ip"`
}

// ContainerNetwork 容器所在网络
type ContainerNetwork struct {
	Name    string   `json:"name"`
	IP      string   `json:"ip"`
	Aliases []string `json:"aliases"`
}

// ContainerMount 容器挂载
type ContainerMount struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	ReadOnly    bool   `json:"readOnly"`
}

// ContainerStats 容器实时资源快照
type ContainerStats struct {
	CPUPct                float64 `json:"cpuPct"`
	MemUsageBytes         int64   `json:"memUsageBytes"`
	MemLimitBytes         int64   `json:"memLimitBytes"`
	NetRxBytesPerSec      int64   `json:"netRxBytesPerSec"`
	NetTxBytesPerSec      int64   `json:"netTxBytesPerSec"`
	BlockReadBytesPerSec  int64   `json:"blockReadBytesPerSec"`
	BlockWriteBytesPerSec int64   `json:"blockWriteBytesPerSec"`
}

// PruneResult 清理结果
type PruneResult struct {
	Deleted             []string `json:"deleted"`
	SpaceReclaimedBytes int64    `json:"spaceReclaimedBytes"`
}

// UpdateResult 一键更新容器结果
type UpdateResult struct {
	OK       bool   `json:"ok"`
	OldImage string `json:"oldImage"`
	NewImage string `json:"newImage"`
}
