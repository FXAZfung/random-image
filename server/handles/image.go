package handles

import (
	"github.com/FXAZfung/random-image/internal/logger"
	"github.com/FXAZfung/random-image/pkg/limit"
	"github.com/FXAZfung/random-image/pkg/utils"
	"github.com/FXAZfung/random-image/server/common"
	"net/http"
)

func Random(w http.ResponseWriter, r *http.Request) {
	clientIP := utils.GetClientIp(r)
	userAgent := r.Header.Get("User-Agent")
	if no := limit.IsIpLimited(clientIP); no {
		http.Error(w, "IP limited", http.StatusTooManyRequests)
		return
	}

	image := <-common.ImageChan // 从管道中取图片
	if image == nil {
		http.Error(w, "Failed to load image", http.StatusInternalServerError)
		return
	}

	// 记录请求信息
	logger.Logger.Printf("Image: %v, IP: %s, User-Agent: %s",
		image.Name, clientIP, userAgent)

	// 设置 HTTP 头部，返回图片内容
	w.Header().Set("Content-Type", "image/jpeg") // 假设为 JPEG，可以动态判断类型
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	_, err := w.Write(image.Content)
	if err != nil {
		logger.Logger.Printf("Error writing image response: %v", err)
	}
}
