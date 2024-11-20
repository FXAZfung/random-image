package main

import (
	"fmt"
	"fxaz-random-image/config"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var mainConfig *config.Config

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
	var err error
	mainConfig, err = config.InitConfig("./config.yml")
	// 创建日志文件
	logFile, err := os.OpenFile(fmt.Sprintf("app-%v-%v.log", time.Now().Month(), time.Now().Day()), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	// 设置日志输出到文件和控制台
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)

	log.SetFlags(log.LstdFlags)
	log.SetPrefix(fmt.Sprintf("[%s] ", mainConfig.App.Name))

	if err != nil {
		log.Panic("file open error ", err)
	}
	imageDir := mainConfig.File.Path // 替换为本地图片文件夹路径
	images, err := getImages(imageDir)
	if err != nil {
		log.Panic("Error reading images ", err)
	}

	if len(images) == 0 {
		log.Panic("No images found in the directory")
	}

	http.HandleFunc("/random", func(w http.ResponseWriter, r *http.Request) {
		imagePath := randomImage(images)
		log.Printf("IP: %v Selected image: %v\n", r.RemoteAddr, imagePath)
		// 禁用缓存
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		http.ServeFile(w, r, imagePath)
	})
	fmt.Printf("Server started on %v%v\n", mainConfig.Server.Host, mainConfig.Server.Port)
	fmt.Println("Press Ctrl+C to stop server")
	err = http.ListenAndServe(mainConfig.Server.Port, nil)
	if err != nil {
		log.Panic("ListenAndServe: ", err)
		return
	}
}
