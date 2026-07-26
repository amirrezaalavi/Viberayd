package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/amirrezaalavi/Viberay/internal/models"
)

type Daemon struct {
	Config  *Config
	State   *State
	configs map[string]models.ProxyConfig
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
	}
}

func (d *Daemon) Run(ctx context.Context) error {
	slog.Info("daemon started",
		"cycle_sleep", d.Config.Daemon.CycleSleep(),
		"parallel", d.Config.Daemon.Parallel,
		"depth", d.Config.Daemon.Depth,
	)

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

	tcpTimeout := timeout
	if tcpTimeout > 5*time.Second {
		tcpTimeout = 5 * time.Second
	}
	tcpResults := TCPPing(ctx, candidateCfgs, tcpTimeout)

	var survivors []NamedConfig
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
	SaveState(d.Config.Daemon.StateFile, d.State)

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
	data := strings.Join(lines, "\n")
	if len(lines) > 0 {
		data += "\n"
	}
	os.WriteFile(path, []byte(data), 0644)
}

func (d *Daemon) shutdown() error {
	slog.Info("shutting down")
	d.writeOutputFile()
	SaveState(d.Config.Daemon.StateFile, d.State)
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
