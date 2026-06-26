# VibeRay — AI-Driven Xray Configuration Testing System

## 🏗️ Phase 0: Project Scaffolding
- [x] Initialize Go module (`go mod init github.com/...`)
- [x] Set up directory structure: `cmd/`, `internal/`, `pkg/`, `configs/`, `testdata/`
- [x] Add `main.go` entry point with CLI flag parsing
- [x] Set up structured logging (slog)
- [x] Add `.gitignore`, `Makefile`, basic CI config
- [x] Define project-wide error types and sentinel errors

---

## 📦 Phase 1: Data Models & Config Types
- [x] Define `ProxyConfig` base struct (server, port, protocol, name, remarks)
- [x] Implement `SSConfig` — method, password, plugin, plugin-opts
- [x] Implement `VMessConfig` — uuid, aid, security, network, tls, path, host, sni
- [x] Implement `VLessConfig` — uuid, flow, encryption, network, tls, path, host, sni
- [x] Implement `TrojanConfig` — password, flow, network, tls, sni, host
- [x] Implement `RealityConfig` — uuid, public_key, short_id, flow, fp, spiderX, sni
- [x] Define `TestResult` struct — success/fail, latency breakdowns, errors, metrics
- [x] Define `TestContext` struct — system info, input stats, runtime state
- [x] Implement `ValidationResult` struct with warnings and errors
- [x] Add JSON/YAML marshaling tags to all models

---

## 🔍 Phase 2: Parser Engine
- [x] Build protocol detector — identify SS, VMess, VLess, Trojan, Reality from URI prefix
- [x] Implement encoding detector — base64 vs plain text
- [x] Implement fragment extractor — parse `#name` suffix
- [x] **SS parser** — `ss://` URI (base64 and plain formats)
- [x] **VMess parser** — `vmess://` base64 JSON format
- [x] **VLess parser** — `vless://` URL format with params
- [x] **Trojan parser** — `trojan://` URL format with params
- [x] **Reality detection** — identify Reality configs from VLess params (reality settings)
- [x] Implement batch input parser — base64-encoded subscription lists
- [x] Build unified `Parse(input string) ([]ProxyConfig, error)` entry point

### Validation Layer
- [x] UUID format validator (8-4-4-4-12)
- [x] Port range validator (1–65535)
- [x] Public key format validator (Reality, base64)
- [x] TLS fingerprint validator (chrome, firefox, etc.)
- [x] Flow control validator (xtls-rprx-vision, etc.)
- [x] Network type validator (tcp, kcp, ws, grpc, http, quic)
- [x] Security type validator (auto, aes-128-gcm, chacha20-poly1305, none, zero)
- [x] Plugin validator (obfs, v2ray-plugin, etc.)

---

## 🧪 Phase 3: Testing Engine

### TCP Layer
- [x] Implement raw TCP connectivity test (configurable timeout 3–5s)
- [x] Measure connect latency
- [x] Detect connection refused, timeout, DNS failures

### TLS Layer
- [x] Implement TLS handshake test
- [x] Measure TLS handshake latency
- [x] Validate certificate chain
- [x] Support custom fingerprints (Reality fp values)
- [x] Detect TLS errors (cert mismatch, protocol errors)

### Protocol Layer
- [x] Implement protocol-specific handshake for each type
- [x] SS: test method negotiation
- [x] VMess: test AEAD auth handshake
- [x] VLess: test version negotiation
- [x] Trojan: test password auth handshake
- [x] Reality: test REALITY handshake with public key + short ID

### Xray Proxy Test
- [x] Generate Xray config JSON for each protocol type
- [x] Manage Xray process lifecycle (start, monitor, stop)
- [x] Route test traffic through Xray instance
- [x] Measure actual proxy latency and bandwidth
- [x] Test against target domains (google.com, cloudflare.com)

### Orchestration
- [x] Implement test pipeline (TCP → TLS → Protocol → Proxy)
- [x] Add adaptive depth selection (quick/standard/full/comprehensive)
- [x] Implement priority ordering (Reality > VMess > VLess > Trojan > SS)

---

## ⚡ Phase 4: Concurrency & Resource Management

### Port Manager
- [x] Implement port allocator (base 10820, range 100)
- [x] Port availability checker
- [x] Auto-release on test completion
- [x] Conflict resolution with 2s timeout
- [x] Staggered allocation to avoid collisions

### Thread/Worker Pool
- [x] Configurable worker pool (min of cpu*2, 100 max)
- [x] Semaphore-based resource control
- [x] Work queue for batch processing
- [x] Graceful shutdown and drain

### Xray Instance Pool
- [x] Pool Xray processes for reuse
- [x] Health check for pooled instances
- [x] Auto-restart crashed instances
- [x] Resource limit per instance (~100MB)

### Caching
- [x] DNS LRU cache (TTL: 300s)
- [x] TLS session cache (TTL: 600s)
- [x] Result cache for duplicate servers
- [x] Cache stats tracking and eviction policy

---

## 🤖 Phase 5: AI Orchestrator
- [x] System resource analyzer (CPU, memory, OS detection)
- [x] Input analyzer (count, protocol distribution, duplicates)
- [x] Concurrency decision engine — auto vs manual
- [x] Test depth decision engine — quick/standard/full/comprehensive
- [x] Output format decision engine — auto-select based on config count
- [x] Cache strategy decision — enable if duplicates > 5%
- [x] Retry policy decision — 1–3 retries with exponential backoff
- [x] Timeout adaptation (2–10s based on server response)

---

## 📤 Phase 6: Output Generator
- [x] Summary statistics builder (total, passed, failed, avg latency, success rate)
- [x] Per-config detailed result formatter
- [x] Error message aggregator
- [x] **JSON output** — full machine-readable dump
- [x] **CSV output** — summary metrics for batch processing
- [x] **Table output** — human-readable key metrics
- [x] **Markdown output** — documentation/reports
- [ ] **HTML dashboard output** — interactive view (stretch goal)
- [x] Categorized export: `valid/`, `failed/`, `reality/`, `legacy/`
- [x] Report file naming with timestamp

---

## 🚨 Phase 7: Error Handling & Resilience
- [ ] Error category taxonomy (parse, network, protocol, resource, runtime)
- [ ] Recovery strategies per category
- [ ] Exponential backoff retry (1s, 2s, 4s)
- [ ] Total per-config timeout (30s)
- [ ] Partial result saving / checkpoint recovery
- [ ] Failure categorization and aggregation
- [ ] Xray crash detection and auto-restart
- [ ] Reduce parallelism on resource exhaustion

---

## 📊 Phase 8: CLI Interface
- [ ] Input flags: file path, stdin, URL, subscription
- [ ] `--concurrency` flag (auto default, manual override)
- [ ] `--depth` flag (quick/standard/full/comprehensive)
- [ ] `--output` flag (json/csv/table/markdown/auto)
- [ ] `--timeout` flag
- [ ] `--port-base` flag
- [ ] `--xray-bin` flag
- [ ] `--retry` flag
- [ ] `--verbose` / `--debug` log level flags
- [ ] `--cache` on/off flag
- [ ] Progress bar / live stats display
- [ ] Graceful interrupt handling (Ctrl+C → save partial results)

---

## 🧪 Phase 9: Testing & Quality
- [ ] Unit tests for all parsers (SS, VMess, VLess, Trojan, Reality)
- [ ] Unit tests for validators
- [ ] Unit tests for port manager
- [ ] Integration tests for testing engine (with mock servers)
- [ ] Integration tests for Xray process management
- [ ] Benchmark tests for parser throughput
- [ ] Benchmark tests for concurrency scaling
- [ ] Edge case tests: malformed URIs, missing fields, invalid encodings
- [ ] Fuzz testing for parser inputs
- [ ] Achieve >80% code coverage

---

## 🚀 Phase 10: Polish & Ship
- [ ] README with usage examples
- [ ] Example input files (one per protocol)
- [ ] Dockerfile with Xray binary included
- [ ] CI pipeline (lint, test, build)
- [ ] Cross-platform build (linux, macOS, windows)
- [ ] Release binary packaging

---

## 📌 Dependencies (to evaluate)
- CLI: `cobra` or `urfave/cli`
- Logging: `zerolog` or `slog` (stdlib)
- YAML: `gopkg.in/yaml.v3`
- JSON: stdlib `encoding/json`
- Concurrency: stdlib `sync`, `golang.org/x/sync`
- TLS: stdlib `crypto/tls` (custom fingerprint support)
- Process management: stdlib `os/exec`
- Progress: `schollz/progressbar` or similar

---

## Priority Order
1. **Phase 0–1**: Scaffolding + models (foundation)
2. **Phase 2**: Parser engine (core value — parse all proxy types)
3. **Phase 3–4**: Testing engine + concurrency (core value — test configs)
4. **Phase 5**: AI orchestrator (intelligence layer)
5. **Phase 6–7**: Output + error handling (completeness)
6. **Phase 8–10**: CLI, tests, polish (ship-ready)
