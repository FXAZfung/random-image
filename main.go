package main

import (
	"flag"
	"fmt"
	"github.com/FXAZfung/random-image/internal/config"
	"github.com/FXAZfung/random-image/internal/initialize"
	"github.com/FXAZfung/random-image/internal/logger"
	"github.com/FXAZfung/random-image/internal/model"
	"github.com/FXAZfung/random-image/pkg/utils"
	"github.com/FXAZfung/random-image/server"
	"github.com/FXAZfung/random-image/server/common"
	"net/http"
)

func initApp(configPath string) error {
	if err := initialize.InitConfig(configPath); err != nil {
		return fmt.Errorf("config init error: %w", err)
	}

	if err := initialize.InitLogger(); err != nil {
		return fmt.Errorf("logger init error: %w", err)
	}

	common.InitImages()
	common.InitChan()

	return nil
}

// Producer 随机选择图片并加载其内容到管道 只有管道中的图片未满时，才会进行加载获取图片
func Producer(imageChan chan *model.ImageData) {
	MainPath := utils.GetLastElement(config.MainConfig.File.Path)
	for {
		if len(imageChan) < cap(imageChan) {
			image := loadRandomImage(MainPath)
			if image != nil {
				imageChan <- image
			}
		}
	}
}

func loadRandomImage(path string) *model.ImageData {
	imagePath := utils.Random(common.MapImages[path])
	image, err := utils.LoadImage(imagePath)
	if err != nil {
		logger.Logger.Printf("Error loading image %v: %v", imagePath, err)
		return nil
	}
	return image
}

func main() {
	configPath := flag.String("config", "./config.yaml", "Path to the configuration file")
	flag.Parse()

	if err := initApp(*configPath); err != nil {
		logger.Logger.Fatal(err)
	}

	mux := server.InitRoute()

	go Producer(common.ImageChan)

	fmt.Printf("服务已经在本机的 %v%v%v 启动\n", config.MainConfig.Server.Host, config.MainConfig.Server.Port, config.MainConfig.Server.Path)
	if err := http.ListenAndServe(config.MainConfig.Server.Port, mux); err != nil {
		logger.Logger.Fatal("ListenAndServe: ", err)
	}

	defer common.CloseChan()
}
