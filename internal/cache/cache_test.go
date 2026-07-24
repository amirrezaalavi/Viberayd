package cache

import (
	"testing"
	"time"

	"github.com/amirrezaalavi/Viberay/internal/models"
)

func TestDNSCache_Lookup(t *testing.T) {
	c := NewDNSCache(10, 5*time.Minute)
	// Look up a well-known host
	addrs, err := c.Lookup("localhost")
	if err != nil {
		t.Fatalf("lookup localhost: %v", err)
	}
	if len(addrs) == 0 {
		t.Fatal("expected at least one address for localhost")
	}

	// Second lookup should be cached
	addrs2, err := c.Lookup("localhost")
	if err != nil {
		t.Fatalf("second lookup: %v", err)
	}
	if len(addrs2) != len(addrs) {
		t.Error("cached result length mismatch")
	}

	entries, max := c.Stats()
	if entries != 1 {
		t.Errorf("expected 1 entry, got %d", entries)
	}
	if max != 10 {
		t.Errorf("expected max 10, got %d", max)
	}
}

func TestDNSCache_Expiry(t *testing.T) {
	c := NewDNSCache(10, 1*time.Millisecond)
	_, _ = c.Lookup("localhost")
	time.Sleep(20 * time.Millisecond)
	// Entry should be expired; lookup again should miss and re-resolve
	_, err := c.Lookup("localhost")
	if err != nil {
		t.Fatalf("re-lookup after expiry: %v", err)
	}
}

func TestDNSCache_Eviction(t *testing.T) {
	c := NewDNSCache(2, 5*time.Minute)
	_ = c.setDirect("a", []string{"1.1.1.1"})
	_ = c.setDirect("b", []string{"2.2.2.2"})
	_ = c.setDirect("c", []string{"3.3.3.3"}) // should evict "a"

	c.mu.Lock()
	_, okA := c.entries["a"]
	c.mu.Unlock()
	if okA {
		t.Error("expected 'a' to be evicted")
	}
}

func (c *DNSCache) setDirect(key string, addrs []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.entries[key]; ok {
		e.addrs = addrs
		e.timestamp = time.Now()
		c.lru.MoveToFront(e.elem)
		return nil
	}
	e := &dnsEntry{key: key, addrs: addrs, timestamp: time.Now()}
	e.elem = c.lru.PushFront(e)
	c.entries[key] = e
	if c.lru.Len() > c.maxSize {
		c.evictOldest()
	}
	return nil
}

func TestResultCache_GetPut(t *testing.T) {
	c := NewResultCache(10, 5*time.Minute)
	key := "vmess://1.2.3.4:443"
	res := models.TestResult{ID: "test-1", Status: models.StatusSuccess}

	c.Put(key, res)
	got, ok := c.Get(key)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got.ID != res.ID {
		t.Error("result mismatch")
	}

	hits, misses, size, max := c.Stats()
	if hits != 1 {
		t.Errorf("expected 1 hit, got %d", hits)
	}
	if misses != 0 {
		t.Errorf("expected 0 misses, got %d", misses)
	}
	if size != 1 || max != 10 {
		t.Errorf("unexpected size/max: %d/%d", size, max)
	}
}

func TestResultCache_Miss(t *testing.T) {
	c := NewResultCache(10, 5*time.Minute)
	_, ok := c.Get("missing")
	if ok {
		t.Error("expected cache miss")
	}
	hits, misses, _, _ := c.Stats()
	if hits != 0 || misses != 1 {
		t.Errorf("expected 0 hits 1 miss, got %d/%d", hits, misses)
	}
}

func TestResultCache_Expiry(t *testing.T) {
	c := NewResultCache(10, 1*time.Millisecond)
	c.Put("k", models.TestResult{Status: models.StatusSuccess})
	time.Sleep(20 * time.Millisecond)
	_, ok := c.Get("k")
	if ok {
		t.Error("expected expired entry to miss")
	}
}

func TestResultCache_Eviction(t *testing.T) {
	c := NewResultCache(2, 5*time.Minute)
	c.Put("a", models.TestResult{Status: models.StatusSuccess})
	c.Put("b", models.TestResult{Status: models.StatusSuccess})
	c.Put("c", models.TestResult{Status: models.StatusSuccess}) // evict a

	_, ok := c.Get("a")
	if ok {
		t.Error("expected 'a' to be evicted")
	}
}

func TestKeyFor(t *testing.T) {
	cfg := models.ProxyConfig{
		VMess: &models.VMessConfig{
			BaseConfig: models.BaseConfig{Server: "example.com", Port: 443, Protocol: models.ProtocolVMess},
		},
	}
	key := KeyFor(cfg)
	if key != "vmess://example.com:443" {
		t.Errorf("unexpected key: %s", key)
	}
}
