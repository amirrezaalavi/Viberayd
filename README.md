# Viberay Daemon

> **Status:** Complete. This project is read-only.  
> Next: [Viberoxy](https://github.com/amirrezaalavi/viberoxy) — proxy aggregation server built on top of Viberayd.

A long-running daemon that fetches proxy subscription URLs, tests each config (TCP → Xray), and serves the working ones as a subscription endpoint.

- **Supported protocols:** Shadowsocks, VMess, VLESS, Trojan, Reality
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

### 2. Create a config file

```toml
version = 1

[daemon]
urls_file = "urls.txt"
output_file = "working.txt"
state_file = "state.json"
cycle_sleep = 300
parallel = 5
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

### 3. Run the daemon

```bash
./build/viberayd -config config.toml
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
cp config.toml data/config.toml
echo "https://your-subscription-url/sub" > data/urls.txt

docker run --rm -it \
  -v "$(pwd)/data:/work" \
  -p 8080:8080 \
  -p 8081:8081 \
  viberayd:latest
```

- Mounts `./data` to `/work` — config.toml, urls.txt, state.json, working.txt all live here and persist across restarts
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

### `config.toml`

| Field | Default | Description |
|---|---|---|
| `version` | `1` | Config file version |
| `urls_file` | `"urls.txt"` | File with subscription URLs (one per line, `#` for comments) |
| `output_file` | `"working.txt"` | Working configs written here each cycle |
| `state_file` | `"state.json"` | Persisted state across restarts |
| `cycle_sleep` | `300` | Seconds between cycles (min 10) |
| `parallel` | `10` | Concurrent Xray tests (1–20, clamped) |
| `timeout` | `10` | Per-test timeout in seconds |
| `depth` | `"standard"` | Test depth: `quick`, `standard`, `full`, `comprehensive` |
| `keep_successful` | `true` | Re-test working configs on subsequent cycles |
| `retest_interval` | `1800` | Seconds before re-testing a working config |
| `http.enabled` | `false` | Enable HTTP subscription + API server |
| `http.port` | `8080` | Subscription endpoint port |
| `http.sub_path` | `"/sub"` | Path for subscription endpoint |
| `http.api_port` | `8081` | Management API port |

---

## CLI Flags

| Flag | Default | Description |
|---|---|---|
| `-config` | `"config.toml"` | Path to config file |
| `-once` | `false` | Run a single test cycle and exit |

Example: `./viberayd -config myconfig.toml -once`

---

## HTTP API

When `http.enabled = true`, two HTTP servers start:

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
Daemon loop (every cycle_sleep seconds):
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
