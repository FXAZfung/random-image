package utils

import (
	"github.com/FXAZfung/random-image/model"
	"os"
	"path/filepath"
)

// GetImages 获取指定目录下的所有图片文件路径
func GetImages(dir string) ([]string, error) {
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

// LoadImage 从文件加载图片内容
func LoadImage(path string) (*model.ImageData, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return &model.ImageData{Name: filepath.Base(path), Content: content}, nil
}
