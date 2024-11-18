package main

import (
	"fmt"
	"fxaz-random-image/config"
	"log"
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
	Config, err := config.InitConfig("./config.yml")
	if err != nil {
		log.Panic("file open error ", err)
	}
	imageDir := Config.File.Path // 替换为本地图片文件夹路径
	images, err := getImages(imageDir)
	if err != nil {
		log.Panic("Error reading images ", err)
	}

	if len(images) == 0 {
		log.Panic("No images found in the directory")
	}

	http.HandleFunc("/random", func(w http.ResponseWriter, r *http.Request) {
		imagePath := randomImage(images)
		fmt.Println("Selected image:", imagePath)
		http.ServeFile(w, r, imagePath)
	})
	fmt.Println("Press Ctrl+C to stop server")
	fmt.Printf("Server started on %v%v\n", Config.Server.Host, Config.Server.Port)
	err = http.ListenAndServe(Config.Server.Port, nil)
	if err != nil {
		log.Panic("ListenAndServe: ", err)
		return
	}
}
