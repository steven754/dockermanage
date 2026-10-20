package model

// ComposeProject compose 项目摘要
type ComposeProject struct {
	Name         string   `json:"name"`
	Path         string   `json:"path"`
	FileCount    int      `json:"fileCount"`
	Files        []string `json:"files"`
	ServiceCount int      `json:"serviceCount"`
	RunningCount int      `json:"runningCount"`
	UpdatedAt    string   `json:"updatedAt"`
}

// ComposeFile compose 项目中的单个文件
type ComposeFile struct {
	Name      string `json:"name"`
	Content   string `json:"content"`
	SizeBytes int64  `json:"sizeBytes"`
	UpdatedAt string `json:"updatedAt"`
}

// ComposeService compose 服务状态
type ComposeService struct {
	Name        string          `json:"name"`
	Image       string          `json:"image"`
	State       string          `json:"state"`
	Status      string          `json:"status"`
	ContainerID string          `json:"containerId"`
	Ports       []ContainerPort `json:"ports"`
}

// ComposeProjectDetail compose 项目详情
type ComposeProjectDetail struct {
	Name     string           `json:"name"`
	Path     string           `json:"path"`
	Files    []ComposeFile    `json:"files"`
	Services []ComposeService `json:"services"`
}

// SaveComposeFileRequest 保存 compose 文件请求体
type SaveComposeFileRequest struct {
	Content string `json:"content"`
}

// SaveComposeFileResponse 保存 compose 文件响应体
type SaveComposeFileResponse struct {
	OK        bool  `json:"ok"`
	SizeBytes int64 `json:"sizeBytes"`
}

// ComposeUpRequest 启动 compose 项目请求体
type ComposeUpRequest struct {
	Services []string `json:"services"`
}

// ComposeUpResponse 启动 compose 项目响应体
type ComposeUpResponse struct {
	OK      bool     `json:"ok"`
	Started []string `json:"started"`
}

// ComposeDownRequest 停止 compose 项目请求体
type ComposeDownRequest struct {
	RemoveVolumes bool `json:"removeVolumes"`
}
