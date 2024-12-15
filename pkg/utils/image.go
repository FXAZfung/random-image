package utils

import (
	"github.com/FXAZfung/random-image/internal/model"
	"os"
	"path/filepath"
	"strings"
)

// GetImages 获取目录下的以及所有子目录的所有图片
func GetImages(dir string) ([]string, error) {
	var images []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// 检查文件是否是图片
		if !info.IsDir() && IsImage(path) {
			images = append(images, path)
		}
		return nil
	})
	return images, err
}

// IsImage 检查文件是否是图片
func IsImage(path string) bool {
	ext := filepath.Ext(path)
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif":
		return true
	}
	return false
}

// GetImagesFromSubDir 获取目录下的所有子目录名称和图片数组，不包括主目录
func GetImagesFromSubDir(dir string) (map[string][]string, error) {
	mapImages := make(map[string][]string)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// 检查文件是否是图片
		if info.IsDir() {
			imgs, err := GetImages(path)
			if err != nil {
				return err
			}
			mapImages[LowerString(info.Name())] = imgs
		}
		return nil
	})
	return mapImages, err
}

// LoadImage 从文件加载图片内容
func LoadImage(path string) (*model.ImageData, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return &model.ImageData{Name: filepath.Base(path), Content: content}, nil
}

// LowerString 将字符串改成小写
func LowerString(s string) string {
	return strings.ToLower(s)
}
