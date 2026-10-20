package model

// Dashboard 面板总览
type Dashboard struct {
	Host         DashboardHost     `json:"host"`
	Resources    DashboardResource `json:"resources"`
	Containers   DashboardCount    `json:"containers"`
	Images       DashboardImages   `json:"images"`
	Volumes      DashboardVolumes  `json:"volumes"`
	Networks     DashboardNetworks `json:"networks"`
	BuildCache   DashboardBuild    `json:"buildCache"`
	RecentEvents []Event           `json:"recentEvents"`
}

// DashboardHost 主机信息
type DashboardHost struct {
	Name           string `json:"name"`
	DockerVersion  string `json:"dockerVersion"`
	ComposeVersion string `json:"composeVersion"`
	OS             string `json:"os"`
	Arch           string `json:"arch"`
	CPUCores       int    `json:"cpuCores"`
	UptimeSeconds  uint64 `json:"uptimeSeconds"`
}

// DashboardResource 主机资源占用
type DashboardResource struct {
	CPU     DashboardCPU     `json:"cpu"`
	Memory  DashboardMemory  `json:"memory"`
	Disk    DashboardDisk    `json:"disk"`
	Network DashboardNetwork `json:"network"`
}

// DashboardCPU CPU 占用
type DashboardCPU struct {
	UsedPct float64 `json:"usedPct"`
	Cores   int     `json:"cores"`
}

// DashboardMemory 内存占用
type DashboardMemory struct {
	UsedBytes  int64   `json:"usedBytes"`
	TotalBytes int64   `json:"totalBytes"`
	UsedPct    float64 `json:"usedPct"`
}

// DashboardDisk 磁盘占用
type DashboardDisk struct {
	UsedBytes  int64   `json:"usedBytes"`
	TotalBytes int64   `json:"totalBytes"`
	UsedPct    float64 `json:"usedPct"`
	Mountpoint string  `json:"mountpoint"`
}

// DashboardNetwork 网络速率
type DashboardNetwork struct {
	RxBytesPerSec int64 `json:"rxBytesPerSec"`
	TxBytesPerSec int64 `json:"txBytesPerSec"`
}

// DashboardCount 容器计数
type DashboardCount struct {
	Total   int `json:"total"`
	Running int `json:"running"`
	Stopped int `json:"stopped"`
	Error   int `json:"error"`
}

// DashboardImages 镜像统计
type DashboardImages struct {
	Total          int   `json:"total"`
	TotalSizeBytes int64 `json:"totalSizeBytes"`
}

// DashboardVolumes 卷统计
type DashboardVolumes struct {
	Total  int `json:"total"`
	Orphan int `json:"orphan"`
}

// DashboardNetworks 网络统计
type DashboardNetworks struct {
	Total int `json:"total"`
}

// DashboardBuild 构建缓存统计
type DashboardBuild struct {
	SizeBytes int64 `json:"sizeBytes"`
}
