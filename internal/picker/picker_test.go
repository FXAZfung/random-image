package picker

import (
	"context"
	"testing"
	"time"

	"github.com/FXAZfung/random-image/internal/cache"
	"github.com/FXAZfung/random-image/internal/config"
	"github.com/FXAZfung/random-image/internal/storage"
)

type fakeStorage struct {
	images map[string][]string
	data   map[string]*storage.ImageData
}

func (f *fakeStorage) Name() string { return "fake" }

func (f *fakeStorage) ListImages(_ context.Context, dirPath string) ([]string, error) {
	return append([]string(nil), f.images[dirPath]...), nil
}

func (f *fakeStorage) GetImage(_ context.Context, filePath string) (*storage.ImageData, error) {
	return f.data[filePath], nil
}

func (f *fakeStorage) GetImageURL(_ context.Context, filePath string) (string, error) {
	return "https://example.com/" + filePath, nil
}

func (f *fakeStorage) SupportsRedirect() bool { return true }

func TestPickerAvoidsImmediateRepeats(t *testing.T) {
	store := &fakeStorage{
		images: map[string][]string{
			"/images": {"a.jpg", "b.jpg", "c.jpg"},
		},
		data: map[string]*storage.ImageData{
			"a.jpg": {Data: []byte("a"), ContentType: "image/jpeg", LastModified: time.Unix(1, 0)},
			"b.jpg": {Data: []byte("b"), ContentType: "image/jpeg", LastModified: time.Unix(2, 0)},
			"c.jpg": {Data: []byte("c"), ContentType: "image/jpeg", LastModified: time.Unix(3, 0)},
		},
	}

	p := New(
		cache.New(10, 10, time.Minute),
		[]config.CategoryConfig{{Name: "anime", Storage: "fake", Path: "/images"}},
		map[string]storage.Storage{"fake": store},
		0,
		time.Hour,
		2,
	)

	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("start picker: %v", err)
	}
	defer p.Stop()

	first, err := p.PickPath(context.Background(), "anime")
	if err != nil {
		t.Fatalf("first pick: %v", err)
	}
	second, err := p.PickPath(context.Background(), "anime")
	if err != nil {
		t.Fatalf("second pick: %v", err)
	}

	if first == second {
		t.Fatalf("expected dedupe to avoid immediate repeat, got %q twice", first)
	}
}

func TestReadyCategoryCount(t *testing.T) {
	store := &fakeStorage{
		images: map[string][]string{
			"/ready": {"a.jpg"},
			"/empty": {},
		},
		data: map[string]*storage.ImageData{
			"a.jpg": {Data: []byte("a"), ContentType: "image/jpeg", LastModified: time.Unix(1, 0)},
		},
	}

	p := New(
		cache.New(10, 10, time.Minute),
		[]config.CategoryConfig{
			{Name: "ready", Storage: "fake", Path: "/ready"},
			{Name: "empty", Storage: "fake", Path: "/empty"},
		},
		map[string]storage.Storage{"fake": store},
		0,
		time.Hour,
		1,
	)

	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("start picker: %v", err)
	}
	defer p.Stop()

	if got := p.ReadyCategoryCount(); got != 1 {
		t.Fatalf("expected 1 ready category, got %d", got)
	}
}
