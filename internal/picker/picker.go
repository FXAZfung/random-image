package picker

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"slices"
	"sync"
	"time"

	"github.com/FXAZfung/random-image/internal/cache"
	"github.com/FXAZfung/random-image/internal/config"
	"github.com/FXAZfung/random-image/internal/storage"
)

type ImageResult struct {
	Data         []byte
	ContentType  string
	Path         string
	ETag         string
	LastModified time.Time
}

type CategoryInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Storage     string `json:"storage"`
	Count       int    `json:"count"`
}

type categoryPicker struct {
	name        string
	path        string
	description string
	storageName string
	store       storage.Storage

	recentPaths []string
	ready       bool

	mu     sync.RWMutex
	images []string
}

type Picker struct {
	imgCache   *cache.Cache
	categories map[string]*categoryPicker

	prefetchCount    int
	prefetchInterval time.Duration
	refreshInterval  time.Duration
	avoidRepeats     int

	stopCh chan struct{}
	wg     sync.WaitGroup
}

func New(
	imgCache *cache.Cache,
	categories []config.CategoryConfig,
	storageMap map[string]storage.Storage,
	prefetchCount int,
	prefetchInterval time.Duration,
	avoidRepeats int,
) *Picker {
	p := &Picker{
		imgCache:         imgCache,
		categories:       make(map[string]*categoryPicker),
		prefetchCount:    prefetchCount,
		prefetchInterval: prefetchInterval,
		refreshInterval:  10 * time.Minute,
		avoidRepeats:     avoidRepeats,
		stopCh:           make(chan struct{}),
	}

	for _, cat := range categories {
		store := storageMap[cat.Storage]
		p.categories[cat.Name] = &categoryPicker{
			name:        cat.Name,
			path:        cat.Path,
			description: cat.Description,
			storageName: cat.Storage,
			store:       store,
		}
	}

	return p
}

func (p *Picker) Start(ctx context.Context) error {
	for name, cat := range p.categories {
		slog.Info("scanning category",
			"name", name,
			"storage", cat.storageName,
			"path", cat.path,
		)
		images, err := cat.store.ListImages(ctx, cat.path)
		if err != nil {
			cat.mu.Lock()
			cat.images = nil
			cat.ready = false
			cat.recentPaths = nil
			cat.mu.Unlock()
			slog.Error("failed to scan category", "name", name, "error", err)
			continue
		}

		cat.mu.Lock()
		cat.images = images
		cat.ready = len(images) > 0
		cat.recentPaths = trimRecent(cat.recentPaths, images, p.avoidRepeats)
		cat.mu.Unlock()
		slog.Info("category scanned", "name", name, "count", len(images))
	}

	p.wg.Add(1)
	go p.prefetchLoop()

	p.wg.Add(1)
	go p.refreshLoop()

	return nil
}

func (p *Picker) Stop() {
	close(p.stopCh)
	p.wg.Wait()
}

func (p *Picker) Pick(ctx context.Context, category string) (*ImageResult, error) {
	imgPath, cat, err := p.pickRandom(category)
	if err != nil {
		return nil, err
	}
	return p.loadImage(ctx, cat, imgPath)
}

func (p *Picker) PickPath(_ context.Context, category string) (string, error) {
	imgPath, _, err := p.pickRandom(category)
	return imgPath, err
}

func (p *Picker) GetStorage(category string) (storage.Storage, bool) {
	cat, ok := p.categories[category]
	if !ok {
		return nil, false
	}
	return cat.store, true
}

func (p *Picker) Categories() []CategoryInfo {
	result := make([]CategoryInfo, 0, len(p.categories))
	for _, cat := range p.categories {
		cat.mu.RLock()
		result = append(result, CategoryInfo{
			Name:        cat.name,
			Description: cat.description,
			Storage:     cat.storageName,
			Count:       len(cat.images),
		})
		cat.mu.RUnlock()
	}
	return result
}

func (p *Picker) HasCategory(name string) bool {
	_, ok := p.categories[name]
	return ok
}

func (p *Picker) ReadyCategoryCount() int {
	ready := 0
	for _, cat := range p.categories {
		cat.mu.RLock()
		if cat.ready && len(cat.images) > 0 {
			ready++
		}
		cat.mu.RUnlock()
	}
	return ready
}

func (p *Picker) pickRandom(category string) (string, *categoryPicker, error) {
	cat, ok := p.categories[category]
	if !ok {
		return "", nil, fmt.Errorf("category not found: %s", category)
	}

	cat.mu.Lock()
	defer cat.mu.Unlock()

	if len(cat.images) == 0 {
		return "", nil, fmt.Errorf("no images in category: %s", category)
	}

	candidates := make([]string, 0, len(cat.images))
	for _, path := range cat.images {
		if !slices.Contains(cat.recentPaths, path) {
			candidates = append(candidates, path)
		}
	}
	if len(candidates) == 0 {
		candidates = append(candidates, cat.images...)
	}

	selected := candidates[rand.IntN(len(candidates))]
	cat.recentPaths = updateRecent(cat.recentPaths, selected, p.avoidRepeats, len(cat.images))
	return selected, cat, nil
}

func (p *Picker) loadImage(ctx context.Context, cat *categoryPicker, imgPath string) (*ImageResult, error) {
	cacheKey := cat.storageName + ":" + imgPath
	if item, ok := p.imgCache.Get(cacheKey); ok {
		slog.Debug("cache hit", "storage", cat.storageName, "path", imgPath)
		return &ImageResult{
			Data:         item.Data,
			ContentType:  item.ContentType,
			Path:         imgPath,
			ETag:         item.ETag,
			LastModified: item.LastModified,
		}, nil
	}

	slog.Debug("cache miss, loading", "storage", cat.storageName, "path", imgPath)

	imgData, err := cat.store.GetImage(ctx, imgPath)
	if err != nil {
		return nil, fmt.Errorf("load image from %s: %w", cat.storageName, err)
	}

	p.imgCache.Put(cacheKey, imgData.Data, imgData.ContentType, imgData.LastModified)
	if item, ok := p.imgCache.Get(cacheKey); ok {
		return &ImageResult{
			Data:         item.Data,
			ContentType:  item.ContentType,
			Path:         imgPath,
			ETag:         item.ETag,
			LastModified: item.LastModified,
		}, nil
	}

	return &ImageResult{
		Data:         imgData.Data,
		ContentType:  imgData.ContentType,
		Path:         imgPath,
		LastModified: imgData.LastModified,
	}, nil
}

func (p *Picker) prefetchLoop() {
	defer p.wg.Done()

	p.prefetch()

	ticker := time.NewTicker(p.prefetchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.prefetch()
		}
	}
}

func (p *Picker) prefetch() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	for name, cat := range p.categories {
		cat.mu.RLock()
		images := make([]string, len(cat.images))
		copy(images, cat.images)
		cat.mu.RUnlock()

		if len(images) == 0 {
			continue
		}

		count := p.prefetchCount
		if count > len(images) {
			count = len(images)
		}

		perm := rand.Perm(len(images))
		fetched := 0

		for _, idx := range perm {
			if fetched >= count {
				break
			}

			select {
			case <-p.stopCh:
				return
			case <-ctx.Done():
				return
			default:
			}

			imgPath := images[idx]
			cacheKey := cat.storageName + ":" + imgPath

			if _, ok := p.imgCache.Get(cacheKey); ok {
				continue
			}

			if _, err := p.loadImage(ctx, cat, imgPath); err != nil {
				slog.Warn("prefetch failed", "category", name, "path", imgPath, "error", err)
				continue
			}

			fetched++
			slog.Debug("prefetched", "category", name, "storage", cat.storageName, "path", imgPath)
		}

		if fetched > 0 {
			slog.Info("prefetch complete", "category", name, "fetched", fetched)
		}
	}
}

func (p *Picker) refreshLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.refreshCategories()
		}
	}
}

func (p *Picker) refreshCategories() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	for name, cat := range p.categories {
		select {
		case <-p.stopCh:
			return
		case <-ctx.Done():
			slog.Warn("refresh timeout")
			return
		default:
		}

		images, err := cat.store.ListImages(ctx, cat.path)
		if err != nil {
			cat.mu.Lock()
			cat.ready = false
			cat.mu.Unlock()
			slog.Error("refresh category failed", "name", name, "error", err)
			continue
		}

		cat.mu.Lock()
		oldCount := len(cat.images)
		cat.images = images
		cat.ready = len(images) > 0
		cat.recentPaths = trimRecent(cat.recentPaths, images, p.avoidRepeats)
		cat.mu.Unlock()

		slog.Info("category refreshed",
			"name", name,
			"storage", cat.storageName,
			"old_count", oldCount,
			"new_count", len(images),
		)
	}
}

func updateRecent(recent []string, selected string, avoidRepeats int, imageCount int) []string {
	if avoidRepeats <= 0 || imageCount <= 1 {
		return nil
	}

	window := avoidRepeats
	if max := imageCount - 1; window > max {
		window = max
	}
	if window <= 0 {
		return nil
	}

	filtered := make([]string, 0, len(recent)+1)
	for _, path := range recent {
		if path != selected {
			filtered = append(filtered, path)
		}
	}
	filtered = append(filtered, selected)
	if len(filtered) > window {
		filtered = filtered[len(filtered)-window:]
	}
	return filtered
}

func trimRecent(recent []string, images []string, avoidRepeats int) []string {
	if avoidRepeats <= 0 || len(images) <= 1 {
		return nil
	}

	allowed := make(map[string]struct{}, len(images))
	for _, image := range images {
		allowed[image] = struct{}{}
	}

	filtered := make([]string, 0, len(recent))
	for _, path := range recent {
		if _, ok := allowed[path]; ok {
			filtered = append(filtered, path)
		}
	}

	window := avoidRepeats
	if max := len(images) - 1; window > max {
		window = max
	}
	if len(filtered) > window {
		filtered = filtered[len(filtered)-window:]
	}
	return filtered
}
