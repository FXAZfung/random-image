package main

import (
	"fmt"
	"fxaz-random-image/config"
	"fxaz-random-image/logger"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

// ImageData 用于存储图片的文件名和内容
type ImageData struct {
	Name    string
	Content []byte
}

// 获取指定目录下的所有图片文件路径
func getImages(dir string) ([]string, error) {
	var images []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// 检查文件是否是图片
		if !info.IsDir() && (filepath.Ext(path) == ".jpg" || filepath.Ext(path) == ".png" || filepath.Ext(path) == ".jpeg") {
			images = append(images, path)
		}
		return nil
	})
	return images, err
}

// 从文件加载图片内容
func loadImage(path string) (*ImageData, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return &ImageData{Name: filepath.Base(path), Content: content}, nil
}

// 随机选择图片并加载其内容到管道
func producer(images []string, imageChan chan *ImageData, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case imageChan <- func() *ImageData {
			path := images[rand.Intn(len(images))]
			image, err := loadImage(path)
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
func closeChannel(imageChan chan *ImageData, wg *sync.WaitGroup) {
	wg.Wait()
	close(imageChan)
}

// HTTP 处理函数
func randomImageHandler(imageChan chan *ImageData) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		image := <-imageChan // 从管道中取图片
		if image == nil {
			http.Error(w, "Failed to load image", http.StatusInternalServerError)
			return
		}

		// 记录请求信息
		logger.Logger.Printf("Image: %v, IP: %s, User-Agent: %s",
			image.Name, r.RemoteAddr, r.Header.Get("User-Agent"))

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
	images, err := getImages(imageDir)
	if err != nil {
		logger.Logger.Fatal("Error reading images ", err)
	}

	if len(images) == 0 {
		logger.Logger.Fatal("No images found in the directory")
	}

	// 图片管道及同步机制
	imageChan := make(chan *ImageData, 10) // 缓冲大小为 10
	var wg sync.WaitGroup

	// 启动生产者 Goroutine
	wg.Add(1)
	go producer(images, imageChan, &wg)

	// 创建基本路由
	mux := http.NewServeMux()
	mux.HandleFunc("/random", randomImageHandler(imageChan))

	fmt.Printf("服务已经在本机的 %v%v/random 启动\n", config.MainConfig.Server.Host, config.MainConfig.Server.Port)
	if err = http.ListenAndServe(config.MainConfig.Server.Port, mux); err != nil {
		logger.Logger.Fatal("ListenAndServe: ", err)
	}

	defer closeChannel(imageChan, &wg)
}
