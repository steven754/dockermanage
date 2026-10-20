package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// chiURLParam 读取 chi 路由参数
func chiURLParam(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}
