package common

import (
	"github.com/FXAZfung/random-image/internal/config"
	"github.com/FXAZfung/random-image/internal/model"
)

func InitChan() {
	ImageChan = make(chan *model.ImageData, config.MainConfig.File.Cache) // 缓冲大小为 5
}

func CloseChan() {
	close(ImageChan)
}
