package common

import (
	"github.com/FXAZfung/random-image/internal/config"
	"github.com/FXAZfung/random-image/internal/logger"
	"github.com/FXAZfung/random-image/pkg/utils"
)

func InitImages() {
	var err error
	MapImages, err = utils.GetImagesFromSubDir(config.MainConfig.File.Path)
	if err != nil {
		logger.Logger.Fatalf("Error loading images: %v", err)
	}
}
