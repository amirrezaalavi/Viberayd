# Viberay Daemon

> **Status:** Active. Contributions welcome (see open issues/PRs). Additive changes preferred — extend, don't rewrite.  
> Next: [Viberoxy](https://github.com/amirrezaalavi/viberoxy) — proxy aggregation server built on top of Viberayd.

A long-running daemon that fetches proxy subscription URLs, tests each config (TCP → Xray), and serves the working ones as a subscription endpoint.

- **Supported protocols:** Shadowsocks, VMess, VLESS, Trojan, Reality, WireGuard, TUIC, Hysteria2, SOCKS5
- **Output:** Working sharelinks with latency, served as a subscription URL for Xray/V2Ray clients
- **Daemon mode:** Loops forever, re-tests configs periodically

---

## Table of Contents

- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Docker](#docker)
- [Configuration](#configuration)
- [CLI Flags](#cli-flags)
- [HTTP API](#http-api)
- [Test Depth Levels](#test-depth-levels)
- [Architecture](#architecture)
- [Project Structure](#project-structure)

---

## Prerequisites

- **Go 1.26+**
- **[Xray-core](https://github.com/XTLS/Xray-core)** (optional, required for `depth = comprehensive`)

---

## Installation

```bash
git clone <your-fork> viberay
cd viberay
go mod tidy
go build -o build/viberayd ./cmd/viberayd
```

This produces a single binary at `build/viberayd` (or `build/viberayd.exe` on Windows).

---

## Quick Start

### 1. Create a subscription URL list

```bash
echo "https://example.com/sub.txt" > urls.txt
```

### 2. Run the daemon

```bash
DAEMON_URLS_FILE=urls.txt DAEMON_OUTPUT_FILE=working.txt ./build/viberayd
```

### 4. Import the subscription in your client

Use `http://localhost:8080/sub` as a subscription URL in Xray, V2RayNG, Clash, etc.

---

## Docker

### Build the image

```bash
docker build -t viberayd:latest .
# or: make docker-build
```

The image includes xray-core v26.7.11 for comprehensive tests.

### Run the container

```bash
mkdir -p data
echo "https://your-subscription-url/sub" > data/urls.txt

docker run --rm -it \
  -e DAEMON_URLS_FILE=/work/urls.txt \
  -e DAEMON_OUTPUT_FILE=/work/working.txt \
  -v "$(pwd)/data:/work" \
  -p 8080:8080 \
  -p 8081:8081 \
  viberayd:latest
```

- Mounts `./data` to `/work` — urls.txt, state.json, working.txt all live here and persist across restarts
- Port 8080: subscription endpoint
- Port 8081: management API

### With docker-compose

```yaml
services:
  viberayd:
    build: .
    ports:
      - "8080:8080"
      - "8081:8081"
    volumes:
      - ./data:/work
```

---

## Configuration

All config via environment variables:

| Variable | Default | Description |
|---|---|---|
| `DAEMON_URLS_FILE` | `"urls.txt"` | File with subscription URLs (one per line, `#` for comments) |
| `DAEMON_OUTPUT_FILE` | `"working.txt"` | Working configs written here each cycle |
| `DAEMON_STATE_FILE` | `"state.json"` | Persisted state across restarts |
| `DAEMON_CYCLE_SLEEP` | `300` | Seconds between cycles (min 10) |
| `DAEMON_PARALLEL` | `10` | Concurrent Xray tests (1–20, clamped) |
| `DAEMON_TIMEOUT` | `10` | Per-test timeout in seconds |
| `DAEMON_DEPTH` | `"standard"` | Test depth: `quick`, `standard`, `full`, `comprehensive` |
| `DAEMON_KEEP_SUCCESSFUL` | `true` | Re-test working configs on subsequent cycles |
| `DAEMON_RETEST_INTERVAL` | `1800` | Seconds before re-testing a working config |
| `DAEMON_TCP_PING` | `true` | Fast TCP-connect prefilter before the xray test. Set `false` on networks that filter direct TCP to foreign hosts — otherwise everything is marked unreachable and never reaches the authoritative xray test |
| `HTTP_ENABLED` | `false` | Enable HTTP subscription + API server |
| `HTTP_PORT` | `8080` | Subscription endpoint port |
| `HTTP_SUB_PATH` | `"/sub"` | Path for subscription endpoint |
| `HTTP_API_PORT` | `8081` | Management API port |

---

## CLI Flags

| Flag | Default | Description |
|---|---|---|
| `-once` | `false` | Run a single test cycle and exit |

Example: `DAEMON_URLS_FILE=urls.txt ./viberayd -once`

---

## HTTP API

When `HTTP_ENABLED=true`, two HTTP servers start:

### Subscription endpoint (`:8080`)

| Method | Path | Description |
|---|---|---|
| `GET` | `/sub` | Returns base64 of `working.txt` (valid subscription for Xray) |

### Management API (`:8081`)

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/health` | Liveness probe: `{"status":"ok"}` |
| `GET` | `/api/urls` | List configured subscription URLs |
| `POST` | `/api/urls` | Add a URL: `{"url":"https://..."}` |
| `DELETE` | `/api/urls/{line}` | Remove URL by line number (1-indexed) |
| `GET` | `/api/stats` | Counts: total, working, failed, unreachable |
| `POST` | `/api/cycle/trigger` | Force immediate test cycle |
| `GET` | `/api/configs?page=1&per_page=50` | Paginated config list with states |
| `GET` | `/metrics` | Prometheus-format metrics: `viberayd_configs_total{state}`, `viberayd_build_info` (additive; for Grafana/Prometheus) |

---

## Test Depth Levels

| Level | Stages Run | When to use |
|---|---|---|
| `quick` | TCP only | Large batches, simple health check |
| `standard` | TCP + TLS | Default — good balance |
| `full` | TCP + TLS + Protocol handshake | Small batches, detailed validation |
| `comprehensive` | Full + Xray proxy test | Critical configs, end-to-end |

---

## Architecture

```
Daemon loop (every DAEMON_CYCLE_SLEEP seconds):
  urls.txt
    → HTTP fetch each subscription URL
    → Parse sharelinks (base64 or plain)
    → Deduplicate by SHA256
    → Merge into state.json (new = "unknown")
    → Select candidates due for testing
    → TCP ping filter (parallel, fast, removes unreachable)
    → Xray test pool (bounded by `parallel`, 1-20)
    → Apply results to state
    → Write working.txt
    → Sleep
```

The daemon uses a **two-stage filter**: a lightweight TCP ping pass (high concurrency, short timeout) removes dead hosts before expensive Xray tests.

---

## Project Structure

```
cmd/viberayd/              # daemon binary
internal/daemon/           # daemon logic (config, state, fetcher, tester, loop, http, signals)
internal/tester/           # testing pipeline (TCP → TLS → Protocol → Xray)
internal/concurrency/      # worker pool, port manager, xray process pool
internal/models/           # domain types (ProxyConfig, TestResult, TestDepth)
internal/cache/            # DNS + result caches
internal/errors/           # sentinel errors
internal/output/           # output formatters
internal/orchestrator/     # heuristic decision layer
internal/logging/          # logger setup
pkg/parser/                # sharelink parsers (SS, VMess, VLESS, Trojan, Reality)
pkg/fetcher/               # HTTP subscription fetcher
```
