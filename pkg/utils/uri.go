package utils

import "net/http"

// GetQuery 获取路由参数的query 例如/random?tag=cat 中的cat
func GetQuery(r *http.Request) string {
	return LowerString(r.URL.Query().Get("tag"))
}
