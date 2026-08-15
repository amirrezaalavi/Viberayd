package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/amirrezaalavi/Viberay/internal/models"
)

type Daemon struct {
	Config  *Config
	State   *State
	StateMu sync.RWMutex
	configs map[string]models.ProxyConfig
	trigger chan struct{}
	http    *httpServer
}

func NewDaemon(cfg *Config) *Daemon {
	state, err := LoadState(cfg.Daemon.StateFile)
	if err != nil {
		slog.Warn("load state", "error", err)
		state = NewState()
	}
	return &Daemon{
		Config:  cfg,
		State:   state,
		configs: make(map[string]models.ProxyConfig),
		trigger: make(chan struct{}, 1),
	}
}

func (d *Daemon) Trigger() {
	select {
	case d.trigger <- struct{}{}:
	default:
	}
}

func (d *Daemon) Run(ctx context.Context) error {
	slog.Info("daemon started",
		"cycle_sleep", d.Config.Daemon.CycleSleep(),
		"parallel", d.Config.Daemon.Parallel,
		"depth", d.Config.Daemon.Depth,
	)

	if d.Config.HTTP.Enabled {
		if err := d.StartHTTPServers(); err != nil {
			return fmt.Errorf("start http: %w", err)
		}
	}

	for {
		select {
		case <-ctx.Done():
			return d.shutdown()
		default:
		}

		if err := d.runCycle(ctx); err != nil {
			slog.Error("cycle failed", "error", err)
		}

		select {
		case <-ctx.Done():
			return d.shutdown()
		case <-d.trigger:
		case <-time.After(d.Config.Daemon.CycleSleep()):
		}
	}
}

func (d *Daemon) RunCycle(ctx context.Context) error {
	return d.runCycle(ctx)
}

func (d *Daemon) runCycle(ctx context.Context) error {
	start := time.Now()
	slog.Info("cycle started")

	urls, err := LoadURLs(d.Config.Daemon.URLsFile)
	if err != nil {
		return fmt.Errorf("load urls: %w", err)
	}
	if len(urls) == 0 {
		slog.Warn("no subscription URLs configured")
	}
	slog.Info("loaded urls", "count", len(urls))

	timeout := d.Config.Daemon.Timeout()
	for _, url := range urls {
		if ctx.Err() != nil {
			slog.Info("cycle cancelled during fetch")
			return nil
		}
		fetched, err := FetchAndParse([]string{url}, timeout)
		if err != nil {
			slog.Warn("fetch+parse failed", "url", url, "error", err)
			continue
		}
		d.mergeWithParsed(fetched, url)
	}

	now := time.Now()
	candidates := SelectCandidates(d.State, d.Config.Daemon.RetestInterval(), d.Config.Daemon.KeepSuccessful, now)
	slog.Info("candidates selected", "count", len(candidates))

	if len(candidates) == 0 {
		slog.Info("no candidates to test")
		d.writeOutputFile()
		SaveState(d.Config.Daemon.StateFile, d.State)
		return nil
	}

	candidateCfgs := make(map[string]models.ProxyConfig, len(candidates))
	for _, hash := range candidates {
		if cfg, ok := d.configs[hash]; ok {
			candidateCfgs[hash] = cfg
		}
	}

	if len(candidateCfgs) == 0 {
		slog.Warn("no parsed configs for candidates")
		d.writeOutputFile()
		return nil
	}

	if ctx.Err() != nil {
		slog.Info("cycle cancelled before tcp ping")
		return nil
	}

	tcpTimeout := timeout
	if tcpTimeout > 5*time.Second {
		tcpTimeout = 5 * time.Second
	}

	// DAEMON_TCP_PING gate. When enabled (default), a fast TCP-connect
	// prefilter removes dead hosts before the expensive xray test; failed
	// pings are marked unreachable and skipped. When disabled, the TCP
	// prefilter is skipped entirely and every candidate goes to the xray
	// test, which is the authoritative judge — on networks that filter
	// direct TCP to foreign hosts the prefilter otherwise marks everything
	// unreachable and the pool can never recover.
	var survivors []NamedConfig
	if d.Config.Daemon.TCPPing {
		tcpResults := TCPPing(ctx, candidateCfgs, tcpTimeout)
		for _, hash := range candidates {
			tr, ok := tcpResults[hash]
			if !ok || !tr.Success {
				entry, exists := d.State.Configs[hash]
				if exists && entry.State != StateUnreachable {
					entry.State = StateUnreachable
					entry.LastTested = now
				}
				continue
			}
			if cfg, ok := d.configs[hash]; ok {
				survivors = append(survivors, NamedConfig{Hash: hash, Cfg: cfg})
			}
		}
	} else {
		for _, hash := range candidates {
			if cfg, ok := d.configs[hash]; ok {
				survivors = append(survivors, NamedConfig{Hash: hash, Cfg: cfg})
			}
		}
		slog.Info("tcp ping disabled, testing all candidates via xray", "count", len(survivors))
	}

	if ctx.Err() != nil {
		slog.Info("cycle cancelled before xray test")
		d.writeOutputFile()
		SaveState(d.Config.Daemon.StateFile, d.State)
		return nil
	}

	var xrayResults []TestResult
	if len(survivors) > 0 {
		xrayResults = XrayTest(ctx, survivors, XrayTestConfig{
			Parallel: d.Config.Daemon.Parallel,
			Timeout:  timeout,
			Depth:    d.Config.Daemon.TestDepth(),
			XrayBin:  "xray",
			PortBase: 10820,
		})
	}

	ApplyResults(d.State, xrayResults, now)
	d.writeOutputFile()

	saveCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- SaveState(d.Config.Daemon.StateFile, d.State)
	}()
	select {
	case <-done:
	case <-saveCtx.Done():
		slog.Warn("save state timed out")
	}

	elapsed := time.Since(start)
	slog.Info("cycle complete",
		"duration", elapsed,
		"working", countByState(d.State, StateWorking),
		"failed", countByState(d.State, StateFailed),
		"unreachable", countByState(d.State, StateUnreachable),
	)
	return nil
}

func (d *Daemon) mergeWithParsed(configs map[string]models.ProxyConfig, sourceURL string) {
	for hash, cfg := range configs {
		if _, exists := d.configs[hash]; !exists {
			d.configs[hash] = cfg
		}
	}
	MergeIntoState(d.State, configs, sourceURL)
}

func (d *Daemon) writeOutputFile() {
	path := d.Config.Daemon.OutputFile
	if path == "" {
		return
	}
	var lines []string
	for _, entry := range d.State.Configs {
		if entry.State != StateWorking {
			continue
		}
		line := entry.Raw
		if entry.LatencyMs > 0 {
			line = fmt.Sprintf("%s %dms", entry.Raw, entry.LatencyMs)
		}
		lines = append(lines, line)
	}
	// Sort ascending by latency (best/fastest configs first) so consumers of
	// the subscription get the most reliable configs up front. Map iteration
	// order is random; without this the /sub output order is nondeterministic.
	sort.SliceStable(lines, func(i, j int) bool {
		li, lj := parseLatencyMs(lines[i]), parseLatencyMs(lines[j])
		if li == lj {
			return lines[i] < lines[j]
		}
		return li < lj
	})
	data := strings.Join(lines, "\n")
	if len(lines) > 0 {
		data += "\n"
	}
	os.WriteFile(path, []byte(data), 0644)
}

// parseLatencyMs extracts the trailing "NNNms" latency annotation from a
// working.txt line. Lines without an annotation are treated as +inf so they
// sort after all measured configs.
func parseLatencyMs(line string) int {
	idx := strings.LastIndex(line, " ")
	if idx < 0 {
		return int(^uint(0) >> 1)
	}
	s := line[idx+1:]
	if !strings.HasSuffix(s, "ms") {
		return int(^uint(0) >> 1)
	}
	n, err := strconv.Atoi(strings.TrimSuffix(s, "ms"))
	if err != nil {
		return int(^uint(0) >> 1)
	}
	return n
}

func (d *Daemon) shutdown() error {
	slog.Info("shutting down")
	if d.Config.HTTP.Enabled {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		d.StopHTTPServers(ctx)
	}
	d.writeOutputFile()
	saveCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- SaveState(d.Config.Daemon.StateFile, d.State)
	}()
	select {
	case <-done:
	case <-saveCtx.Done():
		slog.Warn("save state timed out during shutdown")
	}
	return nil
}

func countByState(s *State, state string) int {
	n := 0
	for _, e := range s.Configs {
		if e.State == state {
			n++
		}
	}
	return n
}
