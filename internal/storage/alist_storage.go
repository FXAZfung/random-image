// /home/user/random-image/internal/storage/alist_storage.go
package storage

import (
	"context"

	"github.com/FXAZfung/random-image/internal/alist"
)

// AlistStorage Alist 远程存储实现
type AlistStorage struct {
	client *alist.Client
}

// NewAlistStorage 创建 Alist 存储
func NewAlistStorage(client *alist.Client) *AlistStorage {
	return &AlistStorage{client: client}
}

func (s *AlistStorage) Name() string {
	return "alist"
}

func (s *AlistStorage) ListImages(ctx context.Context, dirPath string) ([]string, error) {
	return s.client.ListImages(ctx, dirPath)
}

func (s *AlistStorage) GetImage(ctx context.Context, filePath string) (*ImageData, error) {
	data, contentType, err := s.client.DownloadFile(ctx, filePath)
	if err != nil {
		return nil, err
	}
	return &ImageData{Data: data, ContentType: contentType}, nil
}

func (s *AlistStorage) GetImageURL(ctx context.Context, filePath string) (string, error) {
	return s.client.GetFileURL(ctx, filePath)
}

func (s *AlistStorage) SupportsRedirect() bool {
	return true
}
