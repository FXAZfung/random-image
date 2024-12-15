package main

import (
	"fmt"
	"github.com/FXAZfung/random-image/internal/config"
	"github.com/FXAZfung/random-image/internal/initialize"
	"github.com/FXAZfung/random-image/internal/logger"
	"github.com/FXAZfung/random-image/internal/model"
	"github.com/FXAZfung/random-image/pkg/utils"
	"github.com/FXAZfung/random-image/server"
	"github.com/FXAZfung/random-image/server/common"
	"math/rand"
	"net/http"
)

// Producer 随机选择图片并加载其内容到管道
func Producer(imageChan chan *model.ImageData) {
	for {
		select {
		case imageChan <- func() *model.ImageData {
			path := common.Images[rand.Intn(len(common.Images))]
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

func main() {

	if err := initialize.InitConfig("./config.yaml"); err != nil {
		logger.Logger.Fatal("config init error ", err)
	}

	if err := initialize.InitLogger(); err != nil {
		logger.Logger.Fatal("logger init error ", err)
	}

	common.InitImages()
	common.InitChan()

	mux := server.InitRoute()

	go Producer(common.ImageChan)

	fmt.Printf("服务已经在本机的 %v%v%v 启动\n", config.MainConfig.Server.Host, config.MainConfig.Server.Port, config.MainConfig.Server.Path)
	if err := http.ListenAndServe(config.MainConfig.Server.Port, mux); err != nil {
		logger.Logger.Fatal("ListenAndServe: ", err)
	}

	defer common.CloseChan()
}
