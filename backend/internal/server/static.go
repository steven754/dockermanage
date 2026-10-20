package server

import (
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/steven754/dockermanage/backend/internal/server/web"
)

// placeholderHTML 前端未构建时的占位页面
const placeholderHTML = `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<title>Dockermanage</title>
<style>
  body { font-family: -apple-system, "PingFang SC", sans-serif; margin: 0; padding: 48px; background: #0b0f19; color: #e6e9f0; }
  code { background: #1b2130; padding: 2px 6px; border-radius: 4px; }
</style>
</head>
<body>
  <h1>Dockermanage</h1>
  <p>后端服务已启动，但未找到前端构建产物。</p>
  <p>请执行 <code>npm run build</code> 生成 <code>frontend/dist</code> 后拷贝至 <code>backend/internal/server/web/dist</code>，或使用 <code>--web-dir</code> 指定静态资源目录。</p>
  <p>接口健康检查：<code>/healthz</code> · <code>/readyz</code></p>
</body>
</html>
`

// staticHandler 前端静态资源处理器（支持 SPA 回退）
type staticHandler struct {
	// files 前端文件系统
	files fs.FS
	// diskDir 磁盘目录（未指定时使用内嵌资源）
	diskDir string
}

// newStaticHandler 创建静态资源处理器
func newStaticHandler(diskDir string) *staticHandler {
	return &staticHandler{diskDir: diskDir}
}

// ServeHTTP 处理静态资源请求
func (s *staticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// API 路径统一返回 JSON 404
	if strings.HasPrefix(r.URL.Path, "/api/") {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"NOT_FOUND","message":"接口不存在"}}`))
		return
	}

	if s.diskDir != "" {
		s.serveFromDisk(w, r)
		return
	}
	s.serveFromEmbed(w, r)
}

// serveFromDisk 从磁盘目录提供静态资源
func (s *staticHandler) serveFromDisk(w http.ResponseWriter, r *http.Request) {
	clean := path.Clean("/" + strings.TrimPrefix(r.URL.Path, "/"))
	full := path.Join(s.diskDir, clean)

	info, err := os.Stat(full)
	if err == nil && !info.IsDir() {
		http.ServeFile(w, r, full)
		return
	}
	http.ServeFile(w, r, path.Join(s.diskDir, "index.html"))
}

// serveFromEmbed 从内嵌资源提供静态资源
func (s *staticHandler) serveFromEmbed(w http.ResponseWriter, r *http.Request) {
	sub, err := web.Sub()
	if err != nil {
		s.servePlaceholder(w, r)
		return
	}

	clean := path.Clean("/" + strings.TrimPrefix(r.URL.Path, "/"))
	name := strings.TrimPrefix(clean, "/")
	if name == "" {
		name = "index.html"
	}

	data, err := fs.ReadFile(sub, name)
	if err != nil {
		// SPA 回退到 index.html
		index, ierr := fs.ReadFile(sub, "index.html")
		if ierr != nil {
			s.servePlaceholder(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(index)
		return
	}

	w.Header().Set("Content-Type", contentType(name))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// servePlaceholder 返回占位页面
func (s *staticHandler) servePlaceholder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(placeholderHTML))
}

// contentType 根据扩展名推断 Content-Type
func contentType(name string) string {
	switch path.Ext(name) {
	case ".html":
		return "text/html; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".ico":
		return "image/x-icon"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	default:
		return "application/octet-stream"
	}
}
