package handles

import (
	"github.com/FXAZfung/random-image/internal/logger"
	"github.com/FXAZfung/random-image/pkg/limit"
	"github.com/FXAZfung/random-image/pkg/utils"
	"github.com/FXAZfung/random-image/server/common"
	"net/http"
)

func Random(w http.ResponseWriter, r *http.Request) {

	// 获取客户端IP和User-Agent
	clientIP := utils.GetClientIp(r)
	userAgent := r.Header.Get("User-Agent")
	// 如果IP被限制，则直接返回
	if no := limit.IsIpLimited(clientIP); no {
		http.Error(w, "IP limited", http.StatusTooManyRequests)
		return
	}

	// 如果没有路径参数
	if utils.GetQuery(r) == "" {
		// 从管道中获取图片
		image := <-common.ImageChan
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
		return
	}
	// 如果有路径参数返回对应的类型的图片
	if utils.GetQuery(r) != "" {
		images := common.MapImages[utils.GetQuery(r)]
		//如果不存在
		if images == nil {
			http.Error(w, "Category not found", http.StatusNotFound)
			return
		}
		//如果存在，随机选择一张图片
		image, err := utils.LoadImage(utils.Random(images))
		if err != nil {
			logger.Logger.Printf("Error loading image: %v", err)
			http.Error(w, "Failed to load image", http.StatusInternalServerError)
			return
		}
		// 记录请求信息
		logger.Logger.Printf("Image: %v, IP: %s, User-Agent: %s",
			image.Name, clientIP, userAgent)
		//返回图片
		w.Header().Set("Content-Type", "image/jpeg") // 假设为 JPEG，可以动态判断类型
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		_, err = w.Write(image.Content)
		if err != nil {
			logger.Logger.Printf("Error writing image response: %v", err)
		}
		return
	}

}
