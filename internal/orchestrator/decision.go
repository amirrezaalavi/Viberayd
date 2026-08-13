// Package orchestrator provides the AI decision layer for VibeRay.
//
// Rather than a complex subsystem, this is a minimal heuristic function
// that analyses the system context and input statistics, then recommends
// concurrency, test depth, output style, retry policy, and caching.
//
// CLI flags always override heuristic recommendations when the user
// explicitly sets a non-zero / non-default value.
package orchestrator

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/amirrezaalavi/Viberayd/internal/models"
)

// UserPreferences captures CLI overrides. Zero-values mean "auto / not set".
type UserPreferences struct {
	Depth       models.TestDepth
	Style       models.OutputStyle
	Workers     int
	Timeout     time.Duration
	MaxRetries  int
	Cache       *bool // nil = unset, true/false = explicit
}

// Decide analyses ctx and user preferences, returning the final run parameters.
//
// Heuristics (overridden by any non-default UserPreferences field):
//   - 0 configs          → defaults (quick, json, 1 worker)
//   - 1–10 configs       → comprehensive depth
//   - 11–100 configs     → full depth
//   - 101–500 configs    → standard depth
//   - 500+ configs       → quick depth
//   - Reality present    → bump depth by one level (capped at comprehensive)
//   - Duplicates > 5%    → enable cache
//   - Auto workers       → min(cpu*2, 100)
//   - Auto timeout       → 5s (base), +2s if any Reality configs present
//   - Auto retries       → 2 for small batches, 1 for large
//   - Auto output style  → detailed json (<10), table (10–100), csv (>100)
func Decide(ctx models.TestContext, prefs UserPreferences) (models.OrchestratorDecision, error) {
	d := models.OrchestratorDecision{}

	// 1. Depth
	d.Depth = decideDepth(ctx, prefs.Depth)

	// 2. Output style
	d.Style = decideStyle(ctx, prefs.Style)

	// 3. Concurrency
	d.Concurrency = decideConcurrency(ctx, prefs.Workers)

	// 4. Retry policy
	d.Retry = decideRetry(ctx, prefs.MaxRetries)

	// 5. Cache
	d.CacheEnabled = decideCache(ctx, prefs.Cache)

	// 6. Timeout per test
	d.TimeoutPerTest = decideTimeout(ctx, prefs.Timeout)

	return d, nil
}

func decideDepth(ctx models.TestContext, override models.TestDepth) models.TestDepth {
	if override.IsValid() {
		return override
	}

	total := ctx.Input.Total
	if total == 0 {
		return models.DepthQuick
	}

	var depth models.TestDepth
	switch {
	case total <= 10:
		depth = models.DepthComprehensive
	case total <= 100:
		depth = models.DepthFull
	case total <= 500:
		depth = models.DepthStandard
	default:
		depth = models.DepthQuick
	}

	// Bump depth by one level if Reality configs are present (they're higher-signal).
	if ctx.Input.ProtocolDistribution[string(models.ProtocolReality)] > 0 {
		depth = bumpDepth(depth)
	}

	return depth
}

func bumpDepth(d models.TestDepth) models.TestDepth {
	switch d {
	case models.DepthQuick:
		return models.DepthStandard
	case models.DepthStandard:
		return models.DepthFull
	case models.DepthFull, models.DepthComprehensive:
		return models.DepthComprehensive
	}
	return d
}

func decideStyle(ctx models.TestContext, override models.OutputStyle) models.OutputStyle {
	if override.IsValid() && override != models.StyleAuto {
		return override
	}
	if override == models.StyleAuto {
		// user explicitly passed "auto" — fall through to heuristic
	}

	total := ctx.Input.Total
	switch {
	case total <= 100:
		return models.StyleTable
	default:
		return models.StyleCSV
	}
}

func decideConcurrency(ctx models.TestContext, override int) models.ConcurrencySettings {
	if override > 0 {
		return models.ConcurrencySettings{
			Workers:     override,
			MaxParallel: override,
		}
	}

	workers := ctx.System.CPUCount * 2
	if workers <= 0 {
		workers = runtime.NumCPU() * 2
	}
	const maxWorkers = 100
	if workers > maxWorkers {
		workers = maxWorkers
	}
	return models.ConcurrencySettings{
		Workers:     workers,
		MaxParallel: workers,
	}
}

func decideRetry(ctx models.TestContext, override int) models.RetryPolicy {
	if override >= 0 {
		return models.RetryPolicy{
			MaxRetries:  override,
			BackoffBase: time.Second,
		}
	}

	// Small batches: be more patient. Large batches: fail fast.
	retries := 2
	if ctx.Input.Total > 100 {
		retries = 1
	}
	return models.RetryPolicy{
		MaxRetries:  retries,
		BackoffBase: time.Second,
	}
}

func decideCache(ctx models.TestContext, override *bool) bool {
	if override != nil {
		return *override
	}
	if ctx.Input.Total == 0 {
		return false
	}
	ratio := float64(ctx.Input.Duplicates) / float64(ctx.Input.Total)
	return ratio > 0.05
}

func decideTimeout(ctx models.TestContext, override time.Duration) time.Duration {
	if override > 0 {
		return override
	}
	timeout := 5 * time.Second
	// Reality handshakes can be slower.
	if ctx.Input.ProtocolDistribution[string(models.ProtocolReality)] > 0 {
		timeout += 2 * time.Second
	}
	return timeout
}

// BuildContext gathers system info and input stats from the parsed configs.
func BuildContext(configs []models.ProxyConfig, parseErrors int) models.TestContext {
	ctx := models.TestContext{
		System: gatherSystemInfo(),
		Input: models.InputStats{
			Total:               len(configs),
			ProtocolDistribution: make(map[string]int),
			ParseErrors:         parseErrors,
			Duplicates:          countDuplicates(configs),
		},
	}

	for _, cfg := range configs {
		p := string(cfg.Protocol())
		if p != "" {
			ctx.Input.ProtocolDistribution[p]++
		}
	}

	return ctx
}

func gatherSystemInfo() models.SystemInfo {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	si := models.SystemInfo{
		CPUCount:  runtime.NumCPU(),
		MemoryMB:  int64(m.Sys / 1024 / 1024),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		GoVersion: runtime.Version(),
	}

	// Best-effort xray version detection.
	if out, err := exec.Command("xray", "version").Output(); err == nil {
		parts := strings.Fields(string(out))
		if len(parts) >= 2 {
			si.XrayVersion = parts[1]
		}
	}

	return si
}

// countDuplicates returns the number of configs that share a server:port
// with at least one earlier config (by linear scan; O(n²) but n is small).
func countDuplicates(configs []models.ProxyConfig) int {
	seen := make(map[string]int, len(configs))
	duplicates := 0
	for _, cfg := range configs {
		addr := cfg.Addr()
		if addr == "" {
			continue
		}
		seen[addr]++
		if seen[addr] > 1 {
			duplicates++
		}
	}
	return duplicates
}

// MustDecide is like Decide but panics on error. Useful in tests.
func MustDecide(ctx models.TestContext, prefs UserPreferences) models.OrchestratorDecision {
	d, err := Decide(ctx, prefs)
	if err != nil {
		panic(fmt.Sprintf("orchestrator.MustDecide: %v", err))
	}
	return d
}
