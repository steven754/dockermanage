package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// chiURLParam 读取 chi 路由参数
func chiURLParam(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}

// chiWildcardParam 读取 chi 通配路由捕获的整段路径
//
// 用于镜像 / 网络这类引用可能带斜杠的资源，例如 myrepo/nginx:1.27。
func chiWildcardParam(r *http.Request) string {
	return chi.URLParam(r, "*")
}
