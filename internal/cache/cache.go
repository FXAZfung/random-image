package cache

import (
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

type Item struct {
	Key          string
	Data         []byte
	ContentType  string
	Size         int64
	CreatedAt    time.Time
	LastModified time.Time
	ETag         string
}

type Cache struct {
	mu          sync.RWMutex
	maxItems    int
	maxMemBytes int64
	ttl         time.Duration

	items    map[string]*list.Element
	lruList  *list.List
	curBytes int64
}

func New(maxItems int, maxMemMB int, ttl time.Duration) *Cache {
	c := &Cache{
		maxItems:    maxItems,
		maxMemBytes: int64(maxMemMB) * 1024 * 1024,
		ttl:         ttl,
		items:       make(map[string]*list.Element),
		lruList:     list.New(),
	}

	if ttl > 0 {
		go c.cleanupLoop()
	}

	return c
}

func (c *Cache) Get(key string) (*Item, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		return nil, false
	}

	item := elem.Value.(*Item)
	if c.ttl > 0 && time.Since(item.CreatedAt) > c.ttl {
		c.removeElement(elem)
		return nil, false
	}

	c.lruList.MoveToFront(elem)

	result := *item
	return &result, true
}

func (c *Cache) Put(key string, data []byte, contentType string, lastModified time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	itemSize := int64(len(data))
	if c.maxMemBytes > 0 && itemSize > c.maxMemBytes/2 {
		return
	}

	etag := buildETag(data)
	if lastModified.IsZero() {
		lastModified = time.Now().UTC()
	}

	if elem, ok := c.items[key]; ok {
		old := elem.Value.(*Item)
		c.curBytes -= old.Size

		old.Data = data
		old.ContentType = contentType
		old.Size = itemSize
		old.CreatedAt = time.Now().UTC()
		old.LastModified = lastModified.UTC()
		old.ETag = etag

		c.curBytes += itemSize
		c.lruList.MoveToFront(elem)
		return
	}

	for (c.maxItems > 0 && c.lruList.Len() >= c.maxItems) ||
		(c.maxMemBytes > 0 && c.curBytes+itemSize > c.maxMemBytes && c.lruList.Len() > 0) {
		c.removeLRU()
	}

	item := &Item{
		Key:          key,
		Data:         data,
		ContentType:  contentType,
		Size:         itemSize,
		CreatedAt:    time.Now().UTC(),
		LastModified: lastModified.UTC(),
		ETag:         etag,
	}

	elem := c.lruList.PushFront(item)
	c.items[key] = elem
	c.curBytes += itemSize
}

func (c *Cache) Stats() (items int, memBytes int64) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lruList.Len(), c.curBytes
}

func (c *Cache) removeLRU() {
	elem := c.lruList.Back()
	if elem != nil {
		c.removeElement(elem)
	}
}

func (c *Cache) removeElement(elem *list.Element) {
	item := elem.Value.(*Item)
	c.lruList.Remove(elem)
	delete(c.items, item.Key)
	c.curBytes -= item.Size
}

func (c *Cache) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanupExpired()
	}
}

func (c *Cache) cleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for elem := c.lruList.Back(); elem != nil; {
		item := elem.Value.(*Item)
		if now.Sub(item.CreatedAt) <= c.ttl {
			break
		}

		prev := elem.Prev()
		c.removeElement(elem)
		elem = prev
	}
}

func buildETag(data []byte) string {
	sum := sha256.Sum256(data)
	return `"` + hex.EncodeToString(sum[:]) + `"`
}
