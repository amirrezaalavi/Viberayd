package daemon

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/amirrezaalavi/Viberayd/internal/models"
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
		"port_base", d.Config.Daemon.PortBase,
	)

	if d.Config.HTTP.Enabled {
		if err := d.StartHTTPServers(); err != nil {
			return fmt.Errorf("start http: %w", err)
		}
	}

	// Publish the output file from the state we just loaded, before the first
	// (possibly hours-long) cycle runs. Otherwise /sub keeps serving whatever
	// the previous process left on disk, which may be stale or, as with the
	// old " <latency>ms" format, unusable by clients.
	d.writeOutputFile()

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

	if ctx.Err() != nil {
		slog.Info("cycle cancelled before xray test")
		d.writeOutputFile()
		SaveState(d.Config.Daemon.StateFile, d.State)
		return nil
	}

	var xrayResults []TestResult
	if len(survivors) > 0 {
		slog.Info("tcp survivors", "count", len(survivors))
		survivors = limitSurvivors(survivors, tcpResults, d.Config.Daemon.MaxXrayTests)
		xrayResults = XrayTest(ctx, survivors, XrayTestConfig{
			Parallel:        d.Config.Daemon.Parallel,
			Timeout:         timeout,
			Depth:           d.Config.Daemon.TestDepth(),
			XrayBin:         "xray",
			PortBase:        d.Config.Daemon.PortBase,
			DomainURLs:      d.Config.Daemon.DomainList(),
			DomainTimeout:   time.Duration(d.Config.Daemon.DomainTimeoutSec) * time.Second,
			DomainMinBytes:  int64(d.Config.Daemon.DomainMinBytes),
			DomainQuorum:    d.Config.Daemon.DomainQuorum,
			StabilityProbes: d.Config.Daemon.StabilityProbes,
		})
	}

	ApplyResults(d.State, xrayResults, now, d.Config.Daemon.MaxLatencyMs)
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

// writeOutputFile writes the working share links, one per line, in ascending
// latency order (best/fastest first). Each line is the verbatim original share
// link: latency is metadata, never part of the URI. An earlier version
// appended " <n>ms" to the URI, which corrupted the subscription served by
// /sub — a line like "vmess://<base64> 55ms" no longer decodes as a share
// link, so strict consumers (and clients) dropped the config entirely.
// Latency stays observable through state.json, the management API and the
// optional CSV sidecar written next to this file.
func (d *Daemon) writeOutputFile() {
	path := d.Config.Daemon.OutputFile
	if path == "" {
		return
	}

	entries := make([]*ConfigEntry, 0, len(d.State.Configs))
	for _, entry := range d.State.Configs {
		if entry.State != StateWorking {
			continue
		}
		entries = append(entries, entry)
	}

	// Sort ascending by latency. Map iteration order is random; without this
	// the /sub output order is nondeterministic.
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].LatencyMs == entries[j].LatencyMs {
			return entries[i].Raw < entries[j].Raw
		}
		return entries[i].LatencyMs < entries[j].LatencyMs
	})

	var b strings.Builder
	for _, entry := range entries {
		b.WriteString(entry.Raw)
		b.WriteByte('\n')
	}
	if err := os.WriteFile(path, []byte(b.String()), 0644); err != nil {
		slog.Warn("write output file", "path", path, "error", err)
		return
	}

	d.writeLatencySidecar(path, entries)
}

// writeLatencySidecar writes "<raw-uri>,<latency_ms>" next to the output file
// (e.g. working.txt -> working.csv) so latency remains available to operators
// without being embedded in the share link. A URI containing a comma is
// quoted by encoding/csv. Failures are logged and never abort the cycle: the
// subscription file itself is already written.
func (d *Daemon) writeLatencySidecar(outputPath string, entries []*ConfigEntry) {
	ext := filepath.Ext(outputPath)
	csvPath := strings.TrimSuffix(outputPath, ext) + ".csv"
	if csvPath == outputPath {
		return
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	for _, entry := range entries {
		if err := w.Write([]string{entry.Raw, strconv.Itoa(entry.LatencyMs)}); err != nil {
			slog.Warn("encode latency sidecar", "path", csvPath, "error", err)
			return
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		slog.Warn("encode latency sidecar", "path", csvPath, "error", err)
		return
	}
	if err := os.WriteFile(csvPath, buf.Bytes(), 0644); err != nil {
		slog.Warn("write latency sidecar", "path", csvPath, "error", err)
	}
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
