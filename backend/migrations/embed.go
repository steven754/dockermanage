// Package migrations 通过 embed 打包 SQL 迁移文件，保证迁移随二进制一起分发。
package migrations

import "embed"

// FS 内嵌的迁移文件，按文件名顺序执行。
//
//go:embed *.sql
var FS embed.FS
