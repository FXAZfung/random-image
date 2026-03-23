package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/FXAZfung/random-image/internal/cache"
	"github.com/FXAZfung/random-image/internal/config"
	"github.com/FXAZfung/random-image/internal/limiter"
	"github.com/FXAZfung/random-image/internal/picker"
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

func newTestHandler(t *testing.T) *Handler {
	t.Helper()

	store := &fakeStorage{
		images: map[string][]string{
			"/images": {"a.jpg"},
		},
		data: map[string]*storage.ImageData{
			"a.jpg": {
				Data:         []byte("image-a"),
				ContentType:  "image/jpeg",
				LastModified: time.Unix(1700000000, 0).UTC(),
			},
		},
	}

	p := picker.New(
		cache.New(10, 10, time.Minute),
		[]config.CategoryConfig{{Name: "anime", Storage: "fake", Path: "/images"}},
		map[string]storage.Storage{"fake": store},
		0,
		time.Hour,
		0,
	)

	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("start picker: %v", err)
	}
	t.Cleanup(p.Stop)

	return NewHandler(
		p,
		cache.New(10, 10, time.Minute),
		limiter.New(30, 10, 100, time.Minute, time.Minute),
		config.RelayConfig{Mode: "proxy", CacheControlMaxAge: 30 * time.Second},
		"v1.2.3",
	)
}

func TestHandleIndexUsesInjectedVersion(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.HandleIndex(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if body["version"] != "v1.2.3" {
		t.Fatalf("expected injected version, got %#v", body["version"])
	}
}

func TestHandleRandomSetsCachingHeadersAndSupports304(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/anime", nil)
	rec := httptest.NewRecorder()
	h.HandleRandom(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Fatal("expected ETag header")
	}
	if rec.Header().Get("Last-Modified") == "" {
		t.Fatal("expected Last-Modified header")
	}
	if got := rec.Header().Get("Cache-Control"); got != "private, max-age=30, must-revalidate" {
		t.Fatalf("unexpected cache-control header: %q", got)
	}

	conditionalReq := httptest.NewRequest(http.MethodGet, "/api/anime", nil)
	conditionalReq.Header.Set("If-None-Match", etag)
	conditionalRec := httptest.NewRecorder()
	h.HandleRandom(conditionalRec, conditionalReq)

	if conditionalRec.Code != http.StatusNotModified {
		t.Fatalf("expected 304, got %d", conditionalRec.Code)
	}
	if conditionalRec.Body.Len() != 0 {
		t.Fatal("expected empty body for 304 response")
	}
}
