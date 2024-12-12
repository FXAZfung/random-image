package main

import (
	"fmt"
	"github.com/FXAZfung/random-image/config"
	"github.com/FXAZfung/random-image/logger"
	"github.com/FXAZfung/random-image/model"
	"github.com/FXAZfung/random-image/utils"
	"golang.org/x/time/rate"
	"math/rand"
	"net/http"
	"sync"
)

// 随机选择图片并加载其内容到管道
func producer(images []string, imageChan chan *model.ImageData, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case imageChan <- func() *model.ImageData {
			path := images[rand.Intn(len(images))]
			image, err := utils.LoadImage(path)
			if err != nil {
				logger.Logger.Printf("Error loading image %v: %v", path, err)
				return nil
			}
			return image
		}():
		}
	}
}

// 关闭管道
func closeChannel(imageChan chan *model.ImageData, wg *sync.WaitGroup) {
	wg.Wait()
	close(imageChan)
}

var ipLimiters sync.Map // 存储每个 IP 的限流器

func getIPLimiter(ip string) *rate.Limiter {
	limiter, ok := ipLimiters.Load(ip)
	if !ok {
		limiter = rate.NewLimiter(5, 2) // 每秒允许 5 个请求，最多存储 2 个令牌
		ipLimiters.Store(ip, limiter)
	}
	return limiter.(*rate.Limiter)
}

func randomImageHandlerWithIPRateLimit(imageChan chan *model.ImageData) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		clientIP := r.Header.Get("X-Forwarded-For")
		userAgent := r.Header.Get("User-Agent")
		if clientIP == "" {
			clientIP = r.RemoteAddr
		}
		limiter := getIPLimiter(clientIP)

		if !limiter.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		image := <-imageChan // 从管道中取图片
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
}

func main() {

	if err := config.InitConfig("./config.yml"); err != nil {
		logger.Logger.Fatal("config init error ", err)
	}

	if err := logger.InitLogger(); err != nil {
		logger.Logger.Fatal("logger init error ", err)
	}

	imageDir := config.MainConfig.File.Path // 替换为本地图片文件夹路径
	images, err := utils.GetImages(imageDir)
	if err != nil {
		logger.Logger.Fatal("Error reading images ", err)
	}

	if len(images) == 0 {
		logger.Logger.Fatal("No images found in the directory")
	}

	// 图片管道及同步机制
	imageChan := make(chan *model.ImageData, 10) // 缓冲大小为 10
	var wg sync.WaitGroup

	// 启动生产者 Goroutine
	wg.Add(1)
	go producer(images, imageChan, &wg)

	// 创建基本路由
	mux := http.NewServeMux()
	mux.HandleFunc(config.MainConfig.Server.Path, randomImageHandlerWithIPRateLimit(imageChan))

	fmt.Printf("服务已经在本机的 %v%v%v 启动\n", config.MainConfig.Server.Host, config.MainConfig.Server.Port, config.MainConfig.Server.Path)
	if err = http.ListenAndServe(config.MainConfig.Server.Port, mux); err != nil {
		logger.Logger.Fatal("ListenAndServe: ", err)
	}

	defer closeChannel(imageChan, &wg)
}
