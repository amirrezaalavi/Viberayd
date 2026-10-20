package daemon

import (
	"context"
	"sync"
	"time"

	"github.com/amirrezaalavi/Viberayd/internal/concurrency"
	"github.com/amirrezaalavi/Viberayd/internal/models"
	"github.com/amirrezaalavi/Viberayd/internal/tester"
)

type TCPResult struct {
	Hash    string
	Success bool
	Latency int
}

type NamedConfig struct {
	Hash string
	Cfg  models.ProxyConfig
}

type XrayTestConfig struct {
	Parallel int
	Timeout  time.Duration
	Depth    models.TestDepth
	XrayBin  string
	PortBase int
	// Full-depth domain content check (see tester.CheckDomains). Empty
	// DomainURLs disables the domain stage (used by tests, not the daemon).
	DomainURLs     []string
	DomainTimeout  time.Duration
	DomainMinBytes int64
	DomainQuorum   int
	// StabilityProbes is how many independent end-to-end verifications a
	// config must pass. 1 (or 0) keeps the single-sample behaviour.
	StabilityProbes int
}

func TCPPing(ctx context.Context, candidates map[string]models.ProxyConfig, timeout time.Duration) map[string]TCPResult {
	results := make(map[string]TCPResult)
	var mu sync.Mutex
	var wg sync.WaitGroup

	sem := make(chan struct{}, 200)

	for hash, cfg := range candidates {
		wg.Add(1)
		sem <- struct{}{}
		go func(h string, c models.ProxyConfig) {
			defer wg.Done()
			defer func() { <-sem }()

			if ctx.Err() != nil {
				return
			}

			pingCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			res := tester.TestTCPForConfig(pingCtx, c, timeout)

			mu.Lock()
			results[h] = TCPResult{
				Hash:    h,
				Success: res.Success,
				Latency: int(res.Latency.Milliseconds()),
			}
			mu.Unlock()
		}(hash, cfg)
	}

	wg.Wait()
	return results
}

// resultLatencyMs is the latency the daemon records for a config and orders
// the served list by: the measured end-to-end (proxied request) latency, i.e.
// the median of the stability passes.
//
// It used to be the TOTAL run duration (every stage, every probe, process
// spawns), so a link's position in /sub reflected how long the test took,
// not how fast the tunnel is — a slow-to-test but fast tunnel sorted behind a
// fast-to-test but slow one.
func resultLatencyMs(res models.TestResult) int {
	if ms := int(res.Latencies.Response.Milliseconds()); ms > 0 {
		return ms
	}
	return int(res.Latencies.Total.Milliseconds())
}

func XrayTest(ctx context.Context, configs []NamedConfig, cfg XrayTestConfig) []TestResult {
	if configs == nil {
		return nil
	}

	results := make([]TestResult, 0, len(configs))
	var mu sync.Mutex
	var wg sync.WaitGroup

	sem := make(chan struct{}, cfg.Parallel)
	pm := concurrency.NewPortManager(cfg.PortBase, 100)

	pipeline := tester.NewPipeline(cfg.Depth)
	pipeline.TCPTimeout = cfg.Timeout
	pipeline.TLSTimeout = cfg.Timeout
	pipeline.ProtoTimeout = cfg.Timeout
	pipeline.XrayTimeout = cfg.Timeout
	pipeline.XrayBin = cfg.XrayBin
	pipeline.DomainURLs = cfg.DomainURLs
	pipeline.DomainTimeout = cfg.DomainTimeout
	pipeline.DomainMinBytes = cfg.DomainMinBytes
	pipeline.DomainQuorum = cfg.DomainQuorum
	pipeline.StabilityProbes = cfg.StabilityProbes

	for _, item := range configs {
		wg.Add(1)
		go func(h string, c models.ProxyConfig) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()

			port, err := pm.Allocate()
			if err != nil {
				return
			}
			defer pm.Release(port)

			// The pipeline runs several stages, each with its own timeout;
			// a single cfg.Timeout wrapper cancelled the later stages (and
			// every extra stability probe) on slower configs.
			testCtx, cancel := context.WithTimeout(ctx, pipeline.Budget())
			defer cancel()

			res := pipeline.Run(testCtx, c, port)

			latency := resultLatencyMs(res)

			tr := TestResult{
				Hash:      h,
				Success:   res.Status == models.StatusSuccess,
				LatencyMs: latency,
			}

			mu.Lock()
			results = append(results, tr)
			mu.Unlock()
		}(item.Hash, item.Cfg)
	}

	wg.Wait()
	return results
}
