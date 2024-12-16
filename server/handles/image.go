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

	if limit.IsIpLimited(clientIP) {
		http.Error(w, "IP limited", http.StatusTooManyRequests)
		return
	}

	query := utils.GetQuery(r)
	if query == "" {
		handleRandomImage(w, clientIP, userAgent)
	} else {
		handleCategoryImage(w, query, clientIP, userAgent)
	}
}

func handleRandomImage(w http.ResponseWriter, clientIP, userAgent string) {
	image := <-common.ImageChan
	if image == nil {
		http.Error(w, "Failed to load image from imageChan", http.StatusInternalServerError)
		return
	}

	logRequest(image.Name, clientIP, userAgent)
	writeImageResponse(w, image.Content)
}

func handleCategoryImage(w http.ResponseWriter, category, clientIP, userAgent string) {
	images := common.MapImages[category]
	if images == nil {
		http.Error(w, "Category not found", http.StatusNotFound)
		return
	}

	image, err := utils.LoadImage(utils.Random(images))
	if err != nil {
		logger.Logger.Printf("Error loading image: %v", err)
		http.Error(w, "Failed to load image from category", http.StatusInternalServerError)
		return
	}

	logRequest(image.Name, clientIP, userAgent)
	writeImageResponse(w, image.Content)
}

func logRequest(imageName, clientIP, userAgent string) {
	logger.Logger.Printf("Image: %v, IP: %s, User-Agent: %s", imageName, clientIP, userAgent)
}

func writeImageResponse(w http.ResponseWriter, content []byte) {
	// 设置响应头
	// 不需要图片缓存
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	_, err := w.Write(content)
	if err != nil {
		logger.Logger.Printf("Error writing image response: %v", err)
	}
}
