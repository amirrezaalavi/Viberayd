package cache

import (
	"testing"
	"time"

	"github.com/amirrezaalavi/Viberayd/internal/models"
)

// BenchmarkResultCache_PutGet measures concurrent Put and Get operations
// on the result cache.
func BenchmarkResultCache_PutGet(b *testing.B) {
	c := NewResultCache(500, 10*time.Minute)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "vmess://server-" + itoa(i) + ":443"
			res := models.TestResult{ID: key, Status: models.StatusSuccess}
			c.Put(key, res)
			_, _ = c.Get(key)
			i++
		}
	})
}

// BenchmarkDNSCache_Lookup measures concurrent Lookup calls on the
// DNS cache with a hot cache (all entries pre-populated).
func BenchmarkDNSCache_Lookup(b *testing.B) {
	c := NewDNSCache(1000, 5*time.Minute)

	// Pre-populate the cache
	for i := 0; i < 100; i++ {
		host := "host-" + itoa(i) + ".example.com"
		_ = c.setDirect(host, []string{"127.0.0.1"})
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			host := "host-" + itoa(i%100) + ".example.com"
			_, _ = c.Lookup(host)
			i++
		}
	})
}

// itoa is a simple int-to-string for local use.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
