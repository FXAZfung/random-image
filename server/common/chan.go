package common

import "github.com/FXAZfung/random-image/internal/model"

func InitChan() {
	ImageChan = make(chan *model.ImageData, 10) // 缓冲大小为 10
}

func CloseChan() {
	close(ImageChan)
}
