// Package version 提供构建期注入的版本信息。
package version

// 以下变量通过 ldflags 在构建时注入，例如：
// go build -ldflags "-X github.com/steven754/dockermanage/backend/internal/version.Version=v1.0.0"
var (
	// Version 语义化版本号
	Version = "dev"
	// Commit git commit short hash
	Commit = "unknown"
	// BuildTime 构建时间（RFC3339）
	BuildTime = "unknown"
)

// Info 版本信息响应体，对应 GET /api/v1/version
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"buildTime"`
}

// Get 返回当前版本信息
func Get() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildTime: BuildTime,
	}
}
