// Package web 提供内嵌的前端静态资源。
package web

import (
	"embed"
	"io/fs"
)

// Dist 内嵌的前端构建产物目录。
//
// 前端 dist 由 Dockerfile 拷贝到此处；开发期目录内只有 .gitkeep，
// 此时 Serve 会返回占位页面。
//
//go:embed all:dist
var Dist embed.FS

// Sub 返回 dist 子目录的文件系统
func Sub() (fs.FS, error) {
	return fs.Sub(Dist, "dist")
}
