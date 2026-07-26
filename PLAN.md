# Viberay Daemon — Implementation Plan

## Overview

Transform the existing Viberay CLI (batch testing) into a **long-running daemon** that fetches subscription URLs, tests sharelinks via TCP → Xray pipeline, outputs working configs, and serves them via HTTP.

Architecture reference (kept below the steps):
- New binary: `cmd/viberayd/`
- New package: `internal/daemon/` (config, state, fetcher, tester, loop, http, signals)
- Reuses existing: `pkg/parser`, `pkg/fetcher`, `internal/tester`, `internal/concurrency`, `internal/models`, `internal/cache`
- No external dependencies added

---

## Implementation Steps

### Step 1: Daemon Scaffold + Config Loading

**Goal:** Create the daemon entry point and config loading. Nothing runs yet.

**Files to create:**
- `cmd/viberayd/main.go` — parses `-config` flag, prints "starting viberayd..." + config, exits
- `internal/daemon/config.go` — loads `config.toml`, defines `DaemonConfig` struct
- `config.toml.example` — documented example

**Config struct:**
```go
type Config struct {
    Version int `toml:"version"`
    Daemon  DaemonConfig `toml:"daemon"`
    HTTP    HTTPConfig   `toml:"http"`
}
type DaemonConfig struct {
    URLsFile        string `toml:"urls_file"`        // default: urls.txt
    OutputFile      string `toml:"output_file"`      // default: working.txt
    StateFile       string `toml:"state_file"`       // default: state.json
    CycleSleepSec   int    `toml:"cycle_sleep"`      // default: 300
    Parallel        int    `toml:"parallel"`          // default: 10, clamp 1-20
    TimeoutSec      int    `toml:"timeout"`           // default: 10
    Depth           string `toml:"depth"`             // default: standard
    KeepSuccessful  bool   `toml:"keep_successful"`   // default: true
    RetestIntervalSec int  `toml:"retest_interval"`   // default: 1800
}
type HTTPConfig struct {
    Enabled  bool   `toml:"enabled"`    // default: false
    Port     int    `toml:"port"`       // default: 8080
    SubPath  string `toml:"sub_path"`  // default: /sub
    APIPort  int    `toml:"api_port"`  // default: 8081
}
```

**How to verify:**
```bash
go build ./cmd/viberayd/
./viberayd -config config.toml.example
# prints "viberayd starting..." and config summary, exits
```

---

### Step 2: State Management

**Goal:** Persist and query config state across cycles using `state.json`.

**Files to create:**
- `internal/daemon/state.go` — `State` struct, `LoadState`, `SaveState` (atomic write), state transitions

**State model:**
```go
type State struct {
    Version   int                  `json:"version"`
    UpdatedAt time.Time            `json:"updated_at"`
    Configs   map[string]*ConfigEntry `json:"configs"` // key: sha256(raw)
}
type ConfigEntry struct {
    Raw          string `json:"raw"`           // original sharelink
    Host         string `json:"host"`
    Port         int    `json:"port"`
    Protocol     string `json:"protocol"`
    SourceURL    string `json:"source_url"`
    FirstSeen    time.Time `json:"first_seen"`
    LastTested   time.Time `json:"last_tested"`
    LastSuccess  time.Time `json:"last_success"`
    SuccessCount int    `json:"success_count"`
    FailCount    int    `json:"fail_count"`
    State        string `json:"state"`        // unknown | unreachable | failed | working
    LatencyMs    int    `json:"latency_ms"`
}
```

**State machine transitions:**
```
unknown → (tcping fail) → unreachable
unknown → (xray fail)   → failed
unknown → (xray pass)   → working
working → (retest pass) → working, updated latency
working → (retest fail) → failed
working → (retest unreachable) → unreachable
failed  → (new cycle)   → working | failed | unreachable
unreachable → (new cycle) → working | failed | unreachable
```

**Key functions:**
- `LoadState(path string) (*State, error)` — read file, unmarshal, return empty state if missing
- `SaveState(path string, s *State) error` — marshal, write to tmp, rename
- `SelectCandidates(s *State, retestInterval time.Duration) []string` — return hashes of configs to test this cycle: `unknown` + `failed` + `unreachable` + `working` (if last_tested + retestInterval < now)
- `ApplyResults(s *State, results []TestResult)` — update entries based on test outcomes

**How to verify:**
```bash
go test ./internal/daemon/ -run State -v
```
- Save + load round-trip (temp file)
- All state transitions via table-driven test
- Candidate selection with various timestamps
- Empty state file → returns empty state (no error)

---

### Step 3: Subscription Fetcher + Parser

**Goal:** Fetch subscription URLs from `urls.txt`, parse sharelinks, deduplicate, merge into state.

**Files to create:**
- `internal/daemon/fetcher.go` — fetches subscriptions, parses bodies, deduplicates

**Key functions:**
```go
// FetchAndParse fetches all subscription URLs, parses sharelinks, returns deduplicated map.
// key = sha256(raw), uses pkg/fetcher.Fetch and pkg/parser.Parse.
func FetchAndParse(urls []string, timeout time.Duration) (map[string]models.ProxyConfig, error)

// MergeIntoState adds new configs from fetched map into state.
// Existing entries keep their state. New entries get state="unknown".
func MergeIntoState(s *State, configs map[string]models.ProxyConfig, sourceURL string)
```

**Reused packages:**
- `pkg/fetcher.Fetch(url, timeout)` → raw string
- `pkg/parser.Parse(raw)` → `[]models.ProxyConfig` (handles base64 + multiple URIs internally)
- `internal/models.ProxyConfig.Raw` → original sharelink text

**Parsing `urls.txt`:** One subscription URL per line. Lines starting with `#` are comments.

```go
// LoadURLs reads urls.txt and returns the list of subscription URLs.
func LoadURLs(path string) ([]string, error)
```

**How to verify:**
- Unit test with `testdata/subscriptions/chunk_000.txt` as a local file subscription
- Test deduplication: same sharelink from 2 sources → one entry
- Test `MergeIntoState`: new configs added, existing configs preserved

---

### Step 4: Test Pipeline Integration

**Goal:** Run Stage 1 (TCP ping filter) and Stage 2 (Xray test) against candidates, return results.

**Files to create:**
- `internal/daemon/tester.go` — orchestrates TCP ping + Xray test phases

**Key functions:**
```go
// TCPPing tests TCP connectivity to host:port for all candidates in parallel.
// Returns map[sha256]reachable (bool) + latency.
func TCPPing(ctx context.Context, candidates map[string]models.ProxyConfig, timeout time.Duration) map[string]TCPResult

// XrayTest runs the full pipeline on configs that passed TCP ping.
// Bounded by parallel limit (1-20). Uses internal/tester.Pipeline.
// Returns TestResults.
func XrayTest(ctx context.Context, configs []models.ProxyConfig, parallel int, timeout time.Duration, depth models.TestDepth) []TestResult
```

**Reused packages:**
- `internal/tester.TestTCPForConfig(cfg, timeout)` — per-config TCP dial
- `internal/tester.Pipeline.Run()` — TCP → TLS → Protocol → Xray stages
- `internal/concurrency.Pool` — worker pool for parallelism
- `internal/concurrency.PortManager` — port allocation for Xray instances

**How to verify:**
- Integration test: `go test -run TestTCPSurvivors`
- Unit test mock TCP dialer (use a local listener + port that refuses)
- Verify: TCP ping runs on ALL candidates in parallel (not bounded by `parallel`)
- Verify: Xray test only runs on TCP survivors, bounded by `parallel`

---

### Step 5: Daemon Loop Orchestration

**Goal:** Wire config → fetch → merge → select → tcp ping → xray test → save → sleep.

**Files to create:**
- `internal/daemon/loop.go` — the core cycle loop
- `cmd/viberayd/main.go` — updated to start the loop (still exits after first cycle, or runs forever with signal handling in Step 7)

**Cycle function:**
```go
func RunCycle(ctx context.Context, cfg *Config, state *State) error {
    // 1. Load subscription URLs from urls_file
    urls, err := LoadURLs(cfg.Daemon.URLsFile)
    // 2. Fetch + parse all subscriptions
    fetched, err := FetchAndParse(urls, timeout)
    // 3. Merge into state
    MergeIntoState(state, fetched, srcURL)
    // 4. Select candidates
    candidates := SelectCandidates(state, retestInterval)
    // 5. TCP ping filter
    tcpResults := TCPPing(ctx, candidates, tcpTimeout)
    // 6. Xray test survivors
    results := XrayTest(ctx, survivors, cfg.Daemon.Parallel, testTimeout, depth)
    // 7. Apply results to state
    ApplyResults(state, results)
    // 8. Write output_file (only working configs)
    WriteOutputFile(cfg.Daemon.OutputFile, state)
    // 9. Save state
    SaveState(cfg.Daemon.StateFile, state)
}
```

**How to verify:**
```bash
go build ./cmd/viberayd/
printf 'https://raw.githubusercontent.com/.../test_sub.txt' > /tmp/test_urls.txt
./viberayd -config /tmp/test_config.toml
```
- Verify `state.json` is created with entries
- Verify `working.txt` is created (may be empty with test data)
- Verify `cycle_sleep` delay between cycles
- Kill with Ctrl+C mid-cycle → verify state is partially saved (graceful shutdown in Step 7)

---

### Step 6: HTTP Server (Subscription + Management API)

**Goal:** Serve working configs as a subscription endpoint + management API.

**Files to create:**
- `internal/daemon/http.go` — HTTP handlers, server start/stop

**Subscription endpoint (public, `http_port`):**
```
GET /sub
→ base64(working.txt lines joined by \n)
Content-Type: text/plain; charset=utf-8
```

**Management API (private, `api_port`):**
```
GET    /api/health           → {"status":"ok"}
GET    /api/urls             → ["https://sub.example.com/list", ...]
POST   /api/urls             → add URL (body: {"url":"..."})
DELETE /api/urls/{id}        → remove URL (id = url-encoded URL or index)
GET    /api/stats            → {"total":150, "working":12, "failed":88, "unreachable":50, "last_cycle":"2026-07-25T12:00:00Z"}
POST   /api/cycle/trigger    → signal daemon to start next cycle immediately
GET    /api/configs          → all configs with state, paginated (?page=1&per_page=50)
```

**State access:**
- `http.go` receives a reference to the daemon state (protected by `sync.RWMutex`)
- Output file read is direct `os.ReadFile` for the `/sub` endpoint

**How to verify:**
```bash
go test ./internal/daemon/ -run HTTP -v
```
- `httptest.NewServer` for both endpoints
- `GET /sub` returns valid base64 of working configs
- `POST /api/urls` adds to list, persists to `urls_file`
- `GET /api/stats` returns correct counts
- `POST /api/cycle/trigger` signals a channel (mock the daemon)
- Verify `http_port` and `api_port` are separate listeners

---

### Step 7: Signal Handling + Graceful Shutdown

**Goal:** Daemon shuts down cleanly on SIGINT/SIGTERM — persist state, kill Xray processes, close HTTP.

**Files to create:**
- `internal/daemon/signals.go` — signal handling, shutdown coordination
- `cmd/viberayd/main.go` — final version with full daemon lifecycle

**Shutdown sequence:**
1. Catch SIGINT/SIGTERM
2. Cancel root context → in-flight tests receive ctx cancellation
3. Wait for TCP ping goroutines (fast, wait max 3s)
4. Wait for Xray test goroutines (wait max `timeout` seconds)
5. Kill orphan Xray processes (port cleanup)
6. Save state.json with partial results
7. Write working.txt with whatever is working so far
8. Close HTTP servers (shutdown with 5s deadline)
9. Exit(0)

**Implementation:**
```go
func HandleSignals(ctx context.Context, cancel context.CancelFunc, daemon *Daemon) {
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        select {
        case <-ctx.Done():
            return
        case <-sigCh:
            log.Println("shutting down...")
            daemon.Shutdown()
            cancel()
        }
    }()
}
```

**How to verify:**
```bash
# Start daemon in background
./viberayd -config /tmp/test.toml &
PID=$!
sleep 2
kill -TERM $PID
wait $PID
# Verify state.json is valid JSON and has configs
# Verify working.txt exists
# Verify no orphaned xray processes: $(pgrep xray) is empty
```

---

### Step 8: Examples, Tests, Polish

**Goal:** Production-ready configuration files, comprehensive tests, CI integration.

**Files to create/modify:**
- `config.toml.example` — full documented example (may already exist from Step 1)
- `urls.txt.example` — example subscription list
- `internal/daemon/state_test.go` — if not already comprehensive
- `internal/daemon/config_test.go` — config loading edge cases
- `internal/daemon/loop_test.go` — integration test with mock subscriptions
- `.github/workflows/ci.yml` — add `viberayd` build to CI

**Verification checklist:**
- `go build ./cmd/viberayd/` passes
- `go vet ./...` — no issues
- `go test ./... -short` — all existing tests pass
- `go test ./internal/daemon/... -v -race` — daemon tests pass
- Coverage: `go test -coverprofile=coverage.out ./...`
- CI builds both `viberay` and `viberayd`

---

## File Layout (final)

```
viberay/
├── cmd/
│   ├── viberay/                  # existing CLI (unchanged)
│   └── viberayd/                 # NEW: daemon
│       └── main.go               # Step 1/5/7
├── internal/
│   └── daemon/                   # NEW
│       ├── config.go             # Step 1: config loading
│       ├── state.go              # Step 2: state management
│       ├── fetcher.go            # Step 3: subscription fetch + parse
│       ├── tester.go             # Step 4: TCP ping + Xray test
│       ├── loop.go               # Step 5: cycle orchestration
│       ├── http.go               # Step 6: HTTP endpoints
│       └── signals.go            # Step 7: graceful shutdown
├── pkg/                          # existing (reused)
├── config.toml.example           # Step 1/8
├── urls.txt.example              # Step 8
└── PLAN.md                       # this file
```

---

## Config file (`config.toml.example`)

```toml
version = 1

[daemon]
urls_file = "urls.txt"
output_file = "working.txt"
state_file = "state.json"
cycle_sleep = 300
parallel = 10
timeout = 10
depth = "standard"
keep_successful = true
retest_interval = 1800

[http]
enabled = true
port = 8080
sub_path = "/sub"
api_port = 8081
```

---

## Acceptance Criteria (end-to-end)

1. `./viberayd -config config.toml` starts daemon, serves HTTP on `:8080/sub`
2. Reads `urls.txt`, fetches subscriptions, parses, deduplicates, TCP-pings, Xray-tests, writes `working.txt`
3. `working.txt` contains only working sharelinks: `vless://...@host:port?...#name 124ms`
4. `GET /sub` returns base64 of `working.txt` (valid subscription for Xray/V2Ray)
5. `POST /api/urls` adds URL to `urls.txt`, persists across restarts
6. `keep_successful=true` + `retest_interval=1800` re-tests working configs after 30min
7. `parallel` is clamped to 1–20
8. `cycle_sleep` controls interval between cycles
9. SIGINT/SIGTERM → graceful shutdown, state persisted, no orphan Xray processes
10. All existing Viberay tests still pass

---

## Existing Code Reuse Summary

| Existing Package | How it's Used |
|------------------|---------------|
| `pkg/parser.Parse()` | Parse fetched subscription bodies → `[]ProxyConfig` |
| `pkg/fetcher.Fetch()` | Download subscription URLs |
| `internal/tester.TestTCPForConfig()` | Stage 1 TCP ping filter |
| `internal/tester.Pipeline.Run()` | Stage 2 Xray test |
| `internal/concurrency.Pool` | Bound Xray test parallelism |
| `internal/concurrency.PortManager` | Allocate ports for Xray instances |
| `internal/models` | ProxyConfig, TestDepth, TestResult — no changes |
