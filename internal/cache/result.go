package cache

import (
	"container/list"
	"fmt"
	"sync"
	"time"

	"github.com/amirrezaalavi/Viberayd/internal/models"
)

// ResultCache stores TestResults keyed by server+protocol so duplicate
// configs can reuse previous results.
type ResultCache struct {
	mu      sync.RWMutex
	entries map[string]*resultEntry
	lru     *list.List
	maxSize int
	ttl     time.Duration

	hits   int
	misses int
}

type resultEntry struct {
	key       string
	result    models.TestResult
	timestamp time.Time
	elem      *list.Element
}

// NewResultCache creates a cache. maxSize <= 0 defaults to 500.
func NewResultCache(maxSize int, ttl time.Duration) *ResultCache {
	if maxSize <= 0 {
		maxSize = 500
	}
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return &ResultCache{
		entries: make(map[string]*resultEntry),
		lru:     list.New(),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

// KeyFor builds a cache key from a config.
func KeyFor(cfg models.ProxyConfig) string {
	b := cfg.Base()
	return fmt.Sprintf("%s://%s:%d", cfg.Protocol(), b.Server, b.Port)
}

// Get looks up a cached result.
func (c *ResultCache) Get(key string) (models.TestResult, bool) {
	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		c.mu.Lock()
		c.misses++
		c.mu.Unlock()
		return models.TestResult{}, false
	}
	if time.Since(e.timestamp) > c.ttl {
		c.mu.Lock()
		c.delEntry(e)
		c.misses++
		c.mu.Unlock()
		return models.TestResult{}, false
	}
	c.mu.Lock()
	c.lru.MoveToFront(e.elem)
	c.hits++
	c.mu.Unlock()
	return e.result, true
}

// Put stores a result.
func (c *ResultCache) Put(key string, res models.TestResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if e, ok := c.entries[key]; ok {
		e.result = res
		e.timestamp = time.Now()
		c.lru.MoveToFront(e.elem)
		return
	}

	e := &resultEntry{
		key:       key,
		result:    res,
		timestamp: time.Now(),
	}
	e.elem = c.lru.PushFront(e)
	c.entries[key] = e

	if c.lru.Len() > c.maxSize {
		c.evictOldest()
	}
}

func (c *ResultCache) delEntry(e *resultEntry) {
	c.lru.Remove(e.elem)
	delete(c.entries, e.key)
}

func (c *ResultCache) evictOldest() {
	back := c.lru.Back()
	if back != nil {
		c.delEntry(back.Value.(*resultEntry))
	}
}

// Stats returns hits, misses, and current size.
func (c *ResultCache) Stats() (hits, misses, size, max int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.hits, c.misses, len(c.entries), c.maxSize
}
