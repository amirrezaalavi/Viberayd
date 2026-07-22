package concurrency

import (
	"context"
	"testing"
)

// BenchmarkPortManager_AllocRelease measures concurrent port allocation
// and release.
func BenchmarkPortManager_AllocRelease(b *testing.B) {
	pm := NewPortManager(60000, 200)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		var ports []int
		for pb.Next() {
			p, err := pm.Allocate()
			if err != nil {
				b.Fatal(err)
			}
			ports = append(ports, p)
			// Release some to keep the range free
			if len(ports) > 10 {
				pm.Release(ports[0])
				ports = ports[1:]
			}
		}
		for _, p := range ports {
			pm.Release(p)
		}
	})
}

// BenchmarkPool_Submit measures throughput of submitting and completing
// no-op work through the bounded worker pool.
func BenchmarkPool_Submit(b *testing.B) {
	p := NewPool(10)
	ctx := context.Background()
	noop := func(ctx context.Context) error { return nil }

	// Submit b.N no-op tasks and wait for completion.
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.Submit(ctx, noop)
	}
	p.Wait()
	b.StopTimer()
}
