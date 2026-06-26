# VibeRay

An AI-driven proxy configuration testing system for Xray/V2Ray. VibeRay parses proxy URIs (Shadowsocks, VMess, VLESS, Trojan, Reality), validates them, runs staged connectivity tests, and generates formatted reports.

## Features

- **Protocol Support**: SS, VMess, VLESS, Trojan, Reality
- **Staged Testing**: TCP connectivity, TLS handshake, protocol-specific probes, Xray proxy test
- **Adaptive Depth**: Automatically selects test depth based on input size (quick/standard/full/comprehensive)
- **Concurrent Testing**: Worker pool with configurable parallelism
- **Result Caching**: LRU cache for duplicate server configurations
- **Resilience**: Exponential backoff retries, dynamic parallelism reduction, checkpoint save on interrupt
- **Multiple Output Formats**: JSON, CSV, table, markdown
- **Categorized Export**: Groups results into valid/failed/reality/legacy directories
- **Subscription Support**: Parse base64 subscription lists or fetch from remote URLs
- **Resume**: Save and resume test runs from checkpoint files

## Installation

```bash
# Clone the repository
git clone https://github.com/amiralavi/viberay.git
cd viberay

# Build
make build

# Or build directly
go build -o build/viberay ./cmd/viberay
```

## Usage

```bash
# Test configs from a file
viberay -input configs.txt

# Test from stdin
echo 'ss://YWVzLTI1Ni1nY206cGFzc3dvcmQ=@1.1.1.1:443#My-SS' | viberay -input -

# Fetch subscription from URL
viberay -input https://example.com/sub.txt

# Quick test with table output
viberay -input configs.txt -depth quick -output table

# Full test with JSON output and categorized export
viberay -input configs.txt -depth full -output json -out-dir ./results

# Quiet mode (output only, no progress)
viberay -input configs.txt -quiet

# Resume from checkpoint
viberay -resume checkpoint.json

# Override parallelism and timeout
viberay -input configs.txt -concurrency 20 -timeout 10s
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-input` | `-` | Input file, `-` for stdin, or URL |
| `-depth` | `auto` | Test depth: quick, standard, full, comprehensive, auto |
| `-output` | `auto` | Output style: json, csv, table, markdown, auto |
| `-concurrency` | `0` | Max parallel tests (0 = auto) |
| `-timeout` | `0` | Per-test timeout (0 = auto) |
| `-port-base` | `10820` | Base port for Xray SOCKS proxies |
| `-xray-bin` | `xray` | Path to Xray binary |
| `-retry` | `-1` | Max retries per config (-1 = auto) |
| `-out-dir` | `` | Directory for categorized exports |
| `-checkpoint-dir` | `` | Directory for checkpoint files |
| `-resume` | `` | Resume from checkpoint file |
| `-verbose` | `false` | Debug logging |
| `-quiet` | `false` | Suppress progress output |
| `-no-cache` | `false` | Disable result caching |

## Test Depth Levels

| Level | Stages | Use Case |
|-------|--------|----------|
| quick | TCP only | Large batches, simple health check |
| standard | TCP + TLS | Moderate batches with TLS configs |
| full | TCP + TLS + Protocol | Small batches, detailed validation |
| comprehensive | Full + Xray proxy | Critical configs, end-to-end test |

## Architecture

```
Input (file/stdin/URL)
  -> Parser (protocol detection, validation)
    -> Orchestrator (adaptive depth, concurrency, caching)
      -> Tester (TCP -> TLS -> Protocol -> Xray)
        -> Output (JSON/CSV/table/markdown + exports)
```

## Requirements

- Go 1.26 or later
- [Xray-core](https://github.com/XTLS/Xray-core) (optional, for comprehensive depth tests)

## License

MIT
