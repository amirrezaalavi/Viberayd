package cache

import (
	"container/list"
	"net"
	"sync"
	"time"
)

// DNSCache is an LRU cache for DNS lookups with TTL.
type DNSCache struct {
	mu      sync.Mutex
	entries map[string]*dnsEntry
	lru     *list.List
	maxSize int
	ttl     time.Duration
}

type dnsEntry struct {
	key       string
	addrs     []string
	timestamp time.Time
	elem      *list.Element
}

// NewDNSCache creates a cache. maxSize <= 0 defaults to 1000.
func NewDNSCache(maxSize int, ttl time.Duration) *DNSCache {
	if maxSize <= 0 {
		maxSize = 1000
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &DNSCache{
		entries: make(map[string]*dnsEntry),
		lru:     list.New(),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

// Lookup returns cached addresses or performs a real lookup.
func (c *DNSCache) Lookup(host string) ([]string, error) {
	c.mu.Lock()
	if e, ok := c.entries[host]; ok {
		if time.Since(e.timestamp) < c.ttl {
			c.lru.MoveToFront(e.elem)
			addrs := make([]string, len(e.addrs))
			copy(addrs, e.addrs)
			c.mu.Unlock()
			return addrs, nil
		}
		// expired
		c.delEntry(e)
	}
	c.mu.Unlock()

	addrs, err := net.LookupHost(host)
	if err != nil {
		return nil, err
	}
	c.set(host, addrs)
	return addrs, nil
}

func (c *DNSCache) set(key string, addrs []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if e, ok := c.entries[key]; ok {
		e.addrs = addrs
		e.timestamp = time.Now()
		c.lru.MoveToFront(e.elem)
		return
	}

	e := &dnsEntry{
		key:       key,
		addrs:     addrs,
		timestamp: time.Now(),
	}
	e.elem = c.lru.PushFront(e)
	c.entries[key] = e

	if c.lru.Len() > c.maxSize {
		c.evictOldest()
	}
}

func (c *DNSCache) delEntry(e *dnsEntry) {
	c.lru.Remove(e.elem)
	delete(c.entries, e.key)
}

func (c *DNSCache) evictOldest() {
	back := c.lru.Back()
	if back != nil {
		c.delEntry(back.Value.(*dnsEntry))
	}
}

// Stats returns current cache size.
func (c *DNSCache) Stats() (entries, max int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries), c.maxSize
}
