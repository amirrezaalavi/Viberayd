// Command bench measures per-stage cost of the real tester pipeline on a
// sample of configs taken from a state.json. It is a diagnostic tool for
// cycle-time regressions: it answers "how many configs reach each stage and
// how long do they spend there", which the daemon's cycle log does not say.
//
// Usage: go run ./cmd/bench -state /opt/viberayd/state.json -n 80
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/amirrezaalavi/Viberayd/internal/models"
	"github.com/amirrezaalavi/Viberayd/internal/tester"
	"github.com/amirrezaalavi/Viberayd/pkg/parser"
)

type entry struct {
	Raw   string `json:"raw"`
	State string `json:"state"`
}

type sample struct {
	stage   string
	elapsed time.Duration
	status  models.TestStatus
	err     string
}

func main() {
	statePath := flag.String("state", "/opt/viberayd/state.json", "state.json to sample")
	n := flag.Int("n", 80, "sample size")
	parallel := flag.Int("parallel", 10, "concurrent tests")
	depth := flag.String("depth", "full", "quick|standard|full")
	probes := flag.Int("probes", 2, "stability probes")
	portBase := flag.Int("port-base", 12100, "first SOCKS port")
	flag.Parse()

	data, err := os.ReadFile(*statePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read state:", err)
		os.Exit(1)
	}
	var st struct {
		Configs map[string]entry `json:"configs"`
	}
	if err := json.Unmarshal(data, &st); err != nil {
		fmt.Fprintln(os.Stderr, "parse state:", err)
		os.Exit(1)
	}

	// Sample only configs that already passed the TCP pre-filter
	// (failed/working): those are the survivors that reach the deep stages.
	var survivors []string
	for _, e := range st.Configs {
		if e.State == "failed" || e.State == "working" {
			survivors = append(survivors, e.Raw)
		}
	}
	sort.Strings(survivors) // deterministic sample
	if len(survivors) == 0 {
		fmt.Fprintln(os.Stderr, "no survivors in state")
		os.Exit(1)
	}
	step := len(survivors) / *n
	if step < 1 {
		step = 1
	}
	var picked []string
	for i := 0; i < len(survivors) && len(picked) < *n; i += step {
		picked = append(picked, survivors[i])
	}

	d := models.DepthFull
	switch *depth {
	case "quick":
		d = models.DepthQuick
	case "standard":
		d = models.DepthStandard
	}

	ctx := context.Background()
	results := make([]sample, len(picked))
	var wg sync.WaitGroup
	sem := make(chan struct{}, *parallel)
	start := time.Now()
	for i, raw := range picked {
		wg.Add(1)
		go func(i int, raw string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			cfg, err := parser.ParseSingle(raw)
			if err != nil {
				results[i] = sample{stage: "parse", err: err.Error()}
				return
			}
			p := tester.NewPipeline(d)
			p.StabilityProbes = *probes
			p.PortBase = *portBase + i
			t0 := time.Now()
			res := p.Run(ctx, cfg, p.PortBase)
			results[i] = sample{
				stage:   res.Stage,
				elapsed: time.Since(t0),
				status:  res.Status,
				err:     firstErr(res.Errors),
			}
		}(i, raw)
	}
	wg.Wait()
	wall := time.Since(start)

	byStage := map[string][]time.Duration{}
	var ok int
	for _, r := range results {
		byStage[r.stage] = append(byStage[r.stage], r.elapsed)
		if r.status == models.StatusSuccess {
			ok++
		}
	}
	fmt.Printf("sample=%d parallel=%d depth=%s probes=%d wall=%s ok=%d\n",
		len(picked), *parallel, *depth, *probes, wall.Round(time.Second), ok)
	fmt.Printf("%-10s %5s %8s %8s %8s %10s\n", "stage", "count", "mean", "p50", "p95", "total")
	var total time.Duration
	keys := make([]string, 0, len(byStage))
	for k := range byStage {
		keys = append(keys, string(k))
	}
	sort.Strings(keys)
	for _, k := range keys {
		ds := byStage[k]
		sort.Slice(ds, func(i, j int) bool { return ds[i] < ds[j] })
		var sum time.Duration
		for _, x := range ds {
			sum += x
		}
		total += sum
		fmt.Printf("%-10s %5d %8s %8s %8s %10s\n", k, len(ds),
			(sum / time.Duration(len(ds))).Round(time.Millisecond),
			ds[len(ds)/2].Round(time.Millisecond),
			ds[len(ds)*95/100].Round(time.Millisecond),
			sum.Round(time.Second))
	}
	fmt.Printf("sum of per-config time=%s (wall*parallel=%s)\n", total.Round(time.Second), (wall * time.Duration(*parallel)).Round(time.Second))
}

func firstErr(errs []string) string {
	if len(errs) == 0 {
		return ""
	}
	return errs[0]
}
