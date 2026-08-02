# Viberay — Agent Guide

> **Status:** Active. Contributions welcome (see open issues/PRs). Additive changes preferred — extend, don't rewrite.
> **Next project:** [Viberoxy](https://github.com/amirrezaalavi/viberoxy) — proxy aggregation server.

> **Purpose:** Let agents find the exact files and patterns they need without reading everything.

---

## Entry Points

| If you need to... | Start here |
|---|---|
| Fix a bug or add a feature to the daemon | `internal/daemon/` — see [Daemon Packages] below |
| Change how configs are tested (TCP/TLS/Xray) | `internal/tester/` |
| Add a new proxy protocol parser | `pkg/parser/` + `internal/models/` |
| Change the HTTP API | `internal/daemon/http.go` |
| Understand the state machine | `internal/daemon/state.go` — `ApplyResults` |
| Modify config or validate user input | `internal/daemon/config.go` |

---

## Directory Map

```
cmd/
  viberayd/main.go          # daemon entry point: -config, -once, SignalContext
internal/
  daemon/                    # daemon: config, state, fetcher, tester, loop, http, signals
    config.go                # Config struct, LoadConfig, DefaultConfig, validation
    state.go                 # State, ConfigEntry, Load/Save, SelectCandidates, ApplyResults
    fetcher.go               # LoadURLs, FetchAndParse, MergeIntoState
    tester.go                # TCPPing (parallel TCP dial), XrayTest (bounded xray pool)
    loop.go                  # Daemon struct, Run (infinite loop), RunCycle, shutdown
    http.go                  # GET /sub (base64 subscription), management API
    signals.go               # SignalContext — SIGINT/SIGTERM → context cancel
  models/                    # domain types: ProxyConfig, TestResult, TestDepth, etc.
  tester/                    # testing pipeline: TCP → TLS → Protocol → Xray
  concurrency/               # Pool, PortManager, XrayPool
  cache/                     # DNS + result LRU caches
  errors/                    # sentinel errors, categorization, recovery strategies
  output/                    # formatters (table, json, csv, markdown)
  logging/                   # slog initialization
  orchestrator/              # Decide() heuristic — depth, concurrency, timeout
pkg/
  parser/                    # Parse, ParseSingle, per-protocol parsers
  fetcher/                   # HTTP fetch for subscription URLs
```

---

## Daemon Packages (`internal/daemon/`)

### Data Flow (one cycle)

```
LoadURLs(urls.txt)
  → FetchAndParse (HTTP GET per URL → parser.Parse → dedup by sha256)
  → MergeIntoState (new configs → state.json, state="unknown")
  → SelectCandidates (unknown + failed + unreachable + working due for retest)
  → TCPPing (parallel TCP connect, fast semaphore=200)
     → unreachable → state="unreachable", skip
  → XrayTest (Pipeline.Run bounded by `parallel` 1-20)
     → success → state="working", write to output file
     → failure → state="failed"
  → SaveState (atomic tmp+rename)
  → Sleep cycle_sleep (or until signal / trigger)
```

### State Machine

Each config entry in state.json cycles through:

```
unknown → (tcping fail) → unreachable
unknown → (xray fail)   → failed
unknown → (xray pass)   → working
working → (retest pass) → working, updated latency
working → (retest fail) → failed
working → (retest unreachable) → unreachable
failed  → (next cycle)  → working | failed | unreachable
```

### Key Functions

| Function | File | What it does |
|---|---|---|
| `LoadConfigFromEnv()` | `config.go` | Parse env vars, apply defaults, clamp parallel 1-20 |
| `NewState()` | `state.go` | Empty state with version=1 |
| `LoadState(path)` | `state.go` | Read state.json, return empty if missing |
| `SaveState(path, s)` | `state.go` | Atomic write (tmp + rename) |
| `SelectCandidates(s, interval, keep, now)` | `state.go` | Filter configs due for testing |
| `ApplyResults(s, results, now)` | `state.go` | Update states based on test outcomes |
| `LoadURLs(path)` | `fetcher.go` | Read URLs file, skip comments/blanks |
| `FetchAndParse(urls, timeout)` | `fetcher.go` | Fetch subscriptions → parse → dedup |
| `MergeIntoState(s, configs, source)` | `fetcher.go` | Add new configs, preserve existing |
| `TCPPing(ctx, candidates, timeout)` | `tester.go` | Parallel TCP dial, semaphore=200 |
| `XrayTest(ctx, configs, cfg)` | `tester.go` | Bounded Pipeline.Run, PortManager |
| `Daemon.Run(ctx)` | `loop.go` | Infinite cycle loop, starts HTTP if enabled |
| `Daemon.RunCycle(ctx)` | `loop.go` | Single cycle (used by -once) |
| `Daemon.Trigger()` | `loop.go` | Interrupt sleep, start next cycle |
| `SignalContext(ctx)` | `signals.go` | Context cancelled by SIGINT/SIGTERM |

---

## Conventions

- **Error wrapping:** Use `fmt.Errorf("context: %w", err)` — never `errors.New` inside the daemon package.
- **Logging:** `slog.Info`/`Warn`/`Error` with key-value pairs. Never `log.Println`.
- **Config validation:** Clamp out-of-range values with a warning, never fail hard on `parallel=100` (clamp to 20).
- **State files:** Atomic writes only (tmp + rename). Partial writes corrupt state.
- **Concurrency:** Use channels not `sync.Mutex` for goroutine communication where possible. Mutex is OK for map writes.

---

## Adding a New HTTP Endpoint

1. Add handler method on `httpServer` in `http.go`
2. Register it in `StartHTTPServers()` with `apiMux.HandleFunc`
3. Add test in `http_test.go` with `httptest.NewRecorder`
4. If it accesses state, use `StateMu.RLock()`/`RUnlock()`

## Changing Config

1. Update `Config`/`DaemonConfig`/`HTTPConfig` structs in `config.go`
2. Update `DefaultConfig()` defaults
3. Update `LoadConfigFromEnv()` and `validate()` if needed
4. Update tests in `config_test.go`

---

## Daemon config reference

```
DAEMON_URLS_FILE=urls.txt         # one subscription URL per line
DAEMON_OUTPUT_FILE=working.txt    # overwritten each cycle: <raw-uri> <latency-ms>
DAEMON_STATE_FILE=state.json      # persisted config state
DAEMON_CYCLE_SLEEP=300            # seconds between cycles (min 10, default 300)
DAEMON_PARALLEL=10                # concurrent xray tests (1-20, default 10)
DAEMON_TIMEOUT=10                 # per-test timeout in seconds (min 1, default 10)
DAEMON_DEPTH=standard             # quick | standard | full | comprehensive
DAEMON_KEEP_SUCCESSFUL=true       # re-test working configs
DAEMON_RETEST_INTERVAL=1800       # seconds before re-testing a working config
HTTP_ENABLED=false                # enable HTTP servers
HTTP_PORT=8080                    # subscription endpoint port
HTTP_SUB_PATH=/sub                # GET /sub returns base64 of working.txt
HTTP_API_PORT=8081                # management API port
```
