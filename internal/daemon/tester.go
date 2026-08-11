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

			testCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
			defer cancel()

			res := pipeline.Run(testCtx, c, port)

			latency := int(res.Latencies.Total.Milliseconds())
			if latency == 0 && res.Latencies.Response > 0 {
				latency = int(res.Latencies.Response.Milliseconds())
			}

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
