# VibeRay — Agent Guide

> **Purpose:** Help AI agents (and humans) navigate the codebase quickly.

---

## Project Overview

VibeRay is an AI-driven Xray/V2Ray proxy configuration testing system written in Go.
It parses proxy URIs (SS, VMess, VLESS, Trojan, Reality), validates them, runs
connectivity tests (TCP → TLS → Protocol → Xray proxy), and generates reports.

---

## Directory Structure

```
viberay/
├── cmd/viberay/              # CLI entry point
│   └── main.go               # flag parsing, signal handling, wires everything
├── internal/
│   ├── models/               # Data models (ALL domain types)
│   │   ├── config.go         # ProxyConfig tagged-union + per-protocol structs + Raw field
│   │   ├── result.go         # TestResult, Summary, ValidationResult
│   │   ├── context.go        # TestContext, TestDepth, OutputStyle, OrchestratorDecision
│   │   └── models_test.go
│   ├── errors/               # Sentinel errors + categorization + recovery strategies
│   │   ├── errors.go         # ErrInvalidProtocol, ErrTCPConnect, CategorizedError, etc.
│   │   ├── recovery.go       # RecoveryAction, Recommend(), StrategyFor()
│   │   └── recovery_test.go
│   ├── logging/              # Structured logging setup
│   │   └── logger.go         # slog initialization
│   ├── orchestrator/         # AI heuristic decision layer (Phase 5)
│   │   ├── decision.go       # Decide(), BuildContext(), UserPreferences
│   │   └── decision_test.go
│   ├── output/               # Output generator + checkpointing (Phase 6/7)
│   │   ├── formatter.go      # Formatter interface + New()
│   │   ├── json.go           # JSONFormatter — outputs working configs as raw URI + latency
│   │   ├── csv.go            # CSVFormatter — outputs working configs as raw URI + latency
│   │   ├── table.go          # TableFormatter — outputs working configs as raw URI + latency
│   │   ├── markdown.go       # MarkdownFormatter — outputs working configs as raw URI + latency
│   │   ├── exporter.go       # Categorized file export (valid/failed/reality/legacy)
│   │   ├── checkpoint.go     # SaveCheckpoint, LoadCheckpoint, RemoveCheckpoint
│   │   ├── checkpoint_test.go
│   │   └── output_test.go
│   ├── tester/               # Testing engine (Phase 3/7)
│   │   ├── tcp.go            # TCP connectivity tests
│   │   ├── tls.go            # TLS handshake tests + fingerprinting
│   │   ├── protocol.go       # Protocol-specific probes
│   │   ├── xray.go           # Xray process runner + SOCKS5 probe
│   │   ├── xray_config.go    # Xray JSON config generation (all 5 protocols)
│   │   ├── pipeline.go       # Pipeline.Run with depth-gated stages
│   │   ├── resilience.go     # ResilientRunner with retry/backoff/load reduction
│   │   ├── resilience_test.go
│   │   └── tester_test.go
│   ├── concurrency/          # Resource management (Phase 4)
│   │   ├── port.go           # PortManager + StaggeredAllocator
│   │   ├── pool.go           # Bounded worker Pool
│   │   ├── xray_pool.go      # Reusable XrayInstance pool
│   │   └── concurrency_test.go
│   └── cache/                # Caching layer (Phase 4)
│       ├── dns.go            # DNS LRU cache (TTL-based)
│       ├── result.go         # TestResult cache for duplicate servers
│       └── cache_test.go
├── pkg/parser/               # Parser engine (Phase 2)
│   ├── parser.go             # Unified Parse() + ParseSingle() entry points (sets Raw field)
│   ├── detector.go           # Protocol detection, base64 helpers, fragment extraction
│   ├── validator.go          # UUID, port, public key, flow, network validators
│   ├── ss.go                 # Shadowsocks parser (SIP002 + legacy base64)
│   ├── vmess.go              # VMess parser (base64 JSON)
│   ├── vless.go              # VLESS parser (URL params)
│   ├── trojan.go             # Trojan parser
│   ├── reality.go            # Reality parser + IsRealityURL detector
│   └── parser_test.go
├── pkg/fetcher/              # Remote subscription fetcher (Phase 8)
│   ├── fetcher.go            # HTTP fetch for subscription URLs
│   └── fetcher_test.go
├── configs/                  # Runtime / example configs (empty, for future use)
├── testdata/                 # Test fixtures (empty, for future use)
├── go.mod                    # module github.com/amiralavi/viberay
├── Makefile                  # build, test, coverage, cross-compile
├── .gitignore
├── TODO.md                   # Full 10-phase roadmap with checkboxes
└── AGENTS.md                 # This file
```

---

## Key Architectural Decisions

1. **Tagged-union for configs.** `models.ProxyConfig` has 5 pointer fields (`SS`, `VMess`, `VLess`, `Trojan`, `Reality`). Only one is non-nil at a time. Accessor methods (`Protocol()`, `Addr()`, `Name()`, `String()`) handle dispatch. It also stores the original input URI in the `Raw` field.

2. **Parser → Tester → Output pipeline.** Data flows in one direction:
   ```
   raw input → parser.Parse() → []ProxyConfig
   → tester.Pipeline.Run() → []TestResult
   → output generator → files / stdout
   ```

3. **Output is raw URIs.** All formatters (table, JSON, CSV, markdown) output only working configs, each as the original input URI with latency appended. No reconstruction, no data loss.

4. **Depth-gated testing.** `models.TestDepth` controls how far the pipeline goes:
   - `quick` = TCP only
   - `standard` = TCP + TLS
   - `full` = TCP + TLS + Protocol handshake
   - `comprehensive` = full + Xray proxy test

5. **Validation is separate from parsing.** Parsers extract fields; `validator.go` checks format constraints. Parsers call validators inline so a malformed URI fails fast.

6. **No external dependencies (yet).** Everything uses the Go standard library. If you need SOCKS5, YAML, or progress bars, check `TODO.md` "Dependencies (to evaluate)" first.

---

## Conventions

- **Error handling:** Use sentinel errors from `internal/errors`. Wrap with `fmt.Errorf("%w: ...", err)` for context.
- **Logging:** Use `log/slog` (already initialized in `main.go`). Avoid `log.Println`.
- **Tests:** Table-driven tests preferred. Network tests must respect `testing.Short()`.
- **Models:** All structs have `json:` and `yaml:` tags for future serialization.

---

## How to Add a New Protocol

1. Add `ProtocolXXX` constant in `internal/models/config.go`
2. Add `XXXConfig` struct in `internal/models/config.go`
3. Add pointer field to `models.ProxyConfig` + update accessor methods
4. Add parser in `pkg/parser/xxx.go` + wire into `ParseSingle()` (make sure to set `Raw` on the returned `ProxyConfig`)
5. Add validator rules in `pkg/parser/validator.go` if needed
6. Add protocol probe in `internal/tester/protocol.go`
7. Add xray outbound builder in `internal/tester/xray_config.go`
8. Update `ConfigPriority()` in `internal/tester/pipeline.go`
9. Add tests in `pkg/parser/parser_test.go` and `internal/tester/tester_test.go`

---

## Phases Completed / Remaining

See `TODO.md` for the full checklist. As of this writing:
- ✅ Phase 0 — Scaffolding
- ✅ Phase 1 — Data Models
- ✅ Phase 2 — Parser Engine
- ✅ Phase 3 — Testing Engine
- ✅ Phase 4 — Concurrency & Resources
- ✅ Phase 5 — AI Orchestrator (heuristic decision layer)
- ✅ Phase 6 — Output Generator
- ✅ Phase 7 — Error Handling & Resilience
- ✅ Phase 8 — CLI Interface
- ⬜ Phase 9 — Testing & Quality
- ⬜ Phase 10 — Polish & Ship

---

## Gotchas

- `ProxyConfig` uses value receivers on accessor methods — don't take its address expecting mutation.
- `ProxyConfig.Raw` stores the original input URI exactly as received. All formatters use this field for output, so the output is guaranteed to match the input for working configs.
- `xray_pool.go` `WriteXrayConfig` uses `os.CreateTemp` with `workDir`. If `workDir` is empty, it writes to the system temp dir.
- The `Pool.Submit` select can race between semaphore send and `ctx.Done()`. Workers should always check `ctx.Err()` before doing real work.
- `testViaSOCKS5` in `xray.go` only does a SOCKS5 greeting, not a full CONNECT + HTTP request. This is intentional to avoid importing `golang.org/x/net/proxy`.
- `orchestrator.Decide()` is intentionally minimal — a single function, not a subsystem. All CLI flags override heuristics.
- `ResilientRunner` in `resilience.go` caps backoff at 30s (`if wait > 30*time.Second`). This prevents runaway waits on repeated failures.
- Checkpoint files are written on `SIGINT`/`SIGTERM` only if there are remaining unprocessed configs. The file contains both completed results and the remaining configs list, so a future run could theoretically resume (resume logic not yet implemented).
- Xray crash auto-restart is deferred because the current `XrayRunner` starts a fresh process per config test anyway. Pool reuse of xray processes (`xray_pool.go`) would be the right place to add restart logic when Phase 4 pooling is more heavily used.
