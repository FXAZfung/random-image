package main

import (
	"fmt"
	"fxaz-random-image/config"
	"fxaz-random-image/middleware"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
)

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

// 随机选择一张图片
func randomImage(images []string) string {
	return images[rand.Intn(len(images))]
}

func main() {

	if err := config.InitConfig("./config.yml"); err != nil {
		middleware.Logger.Fatal("config init error ", err)
	}

	if err := middleware.InitLogger(); err != nil {
		middleware.Logger.Fatal("logger init error ", err)
	}

	imageDir := config.MainConfig.File.Path // 替换为本地图片文件夹路径
	images, err := getImages(imageDir)
	if err != nil {
		middleware.Logger.Fatal("Error reading images ", err)
	}

	if len(images) == 0 {
		middleware.Logger.Fatal("No images found in the directory")
	}

	// 创建基本路由
	mux := http.NewServeMux()

	mux.HandleFunc("/random", func(w http.ResponseWriter, r *http.Request) {
		imagePath := randomImage(images)
		middleware.Logger.Printf("IP: %v Selected image: %v\n", r.RemoteAddr, imagePath)
		// 禁用缓存
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		http.ServeFile(w, r, imagePath)
	})

	//mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./images"))))

	loggedMux := middleware.LoggingMiddleware(mux)

	fmt.Printf("Server started on %v%v\n", config.MainConfig.Server.Host, config.MainConfig.Server.Port)
	if err = http.ListenAndServe(config.MainConfig.Server.Port, loggedMux); err != nil {
		middleware.Logger.Fatal("ListenAndServe: ", err)
		return
	}
}
