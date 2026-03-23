package storage

import (
	"context"
	"time"
)

type ImageData struct {
	Data         []byte
	ContentType  string
	LastModified time.Time
}

type Storage interface {
	Name() string
	ListImages(ctx context.Context, dirPath string) ([]string, error)
	GetImage(ctx context.Context, filePath string) (*ImageData, error)
	GetImageURL(ctx context.Context, filePath string) (string, error)
	SupportsRedirect() bool
}
