package server

import (
	"github.com/FXAZfung/random-image/internal/config"
	"github.com/FXAZfung/random-image/server/handles"
	"net/http"
)

func InitRoute() *http.ServeMux {
	// 创建基本路由
	mux := http.NewServeMux()
	mux.HandleFunc(config.MainConfig.Server.Path, handles.Random)
	return mux
}
