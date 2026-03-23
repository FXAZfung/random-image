// /home/user/random-image/internal/storage/local_storage.go
package storage

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// 支持的图片扩展名
var imageExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
	".webp": true, ".bmp": true, ".svg": true, ".ico": true,
	".avif": true,
}

// LocalStorage 本地文件存储实现
type LocalStorage struct {
	basePath string // 图片根目录的绝对路径
}

// NewLocalStorage 创建本地存储
func NewLocalStorage(basePath string) (*LocalStorage, error) {
	// 转为绝对路径
	absPath, err := filepath.Abs(basePath)
	if err != nil {
		return nil, fmt.Errorf("resolve base path: %w", err)
	}

	// 检查目录是否存在
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("stat base path %q: %w", absPath, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("base path %q is not a directory", absPath)
	}

	return &LocalStorage{basePath: absPath}, nil
}

func (s *LocalStorage) Name() string {
	return "local"
}

// ListImages 递归扫描目录，返回所有图片文件的相对路径
func (s *LocalStorage) ListImages(_ context.Context, dirPath string) ([]string, error) {
	fullPath := s.resolvePath(dirPath)

	// 安全检查：确保不会跳出 basePath
	if !strings.HasPrefix(fullPath, s.basePath) {
		return nil, fmt.Errorf("path traversal detected: %s", dirPath)
	}

	var images []string

	err := filepath.WalkDir(fullPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // 跳过无法访问的文件
		}
		if d.IsDir() {
			return nil
		}
		if isImageFile(d.Name()) {
			// 返回相对于 basePath 的路径
			relPath, err := filepath.Rel(s.basePath, path)
			if err != nil {
				return nil
			}
			// 统一使用正斜杠
			images = append(images, filepath.ToSlash(relPath))
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk directory: %w", err)
	}

	return images, nil
}

// GetImage 读取本地图片文件
func (s *LocalStorage) GetImage(_ context.Context, filePath string) (*ImageData, error) {
	fullPath := s.resolvePath(filePath)

	// 安全检查
	if !strings.HasPrefix(fullPath, s.basePath) {
		return nil, fmt.Errorf("path traversal detected: %s", filePath)
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}

	contentType := detectContentType(filePath, data)

	return &ImageData{
		Data:         data,
		ContentType:  contentType,
		LastModified: info.ModTime().UTC(),
	}, nil
}

// GetImageURL 本地存储不支持直链
func (s *LocalStorage) GetImageURL(_ context.Context, _ string) (string, error) {
	return "", fmt.Errorf("local storage does not support direct URL")
}

// SupportsRedirect 本地存储不支持 302 跳转
func (s *LocalStorage) SupportsRedirect() bool {
	return false
}

// resolvePath 将相对路径解析为绝对路径
func (s *LocalStorage) resolvePath(relPath string) string {
	// 清理路径，防止 .. 之类的路径穿越
	cleaned := filepath.Clean(relPath)
	return filepath.Join(s.basePath, cleaned)
}

func isImageFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return imageExtensions[ext]
}

func detectContentType(filePath string, data []byte) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	case ".svg":
		return "image/svg+xml"
	case ".avif":
		return "image/avif"
	case ".ico":
		return "image/x-icon"
	default:
		return http.DetectContentType(data)
	}
}
