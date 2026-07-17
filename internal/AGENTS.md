# Internal Directory — Agent Guide

The `internal/` directory contains VibeRay's core application packages. These components are private to the module and handle testing state, resources, orchestration heuristics, logging, error handling, and output generation.

## Directory Map

```
viberay/internal/
├── cache/            # LRU DNS & TestResult caches
├── concurrency/      # Worker, loopback port, and Xray subprocess pool managers
├── errors/           # Sentinel errors, classifications, and recovery recommendations
├── logging/          # slog text-logger helper
├── models/           # Unified tagged-union configuration and context schemas
├── orchestrator/     # AI/heuristic runtime decision engine
├── output/           # Formatter rendering and Ctrl+C checkpoint serialized savers
└── tester/           # incremental testing pipelines (TCP, TLS, Protocol, and Xray Proxy)
```

## Internal Architecture & Wiring

All packages in `internal/` are strictly wired around unidirectional data flow. The orchestration and lifecycle are coordinate as follows:

```mermaid
graph TD
    A[Parser Engine - pkg/parser] -->|[]ProxyConfig| B[Orchestrator - internal/orchestrator]
    B -->|BuildContext| C[Orchestrator Decision]
    C -->|Run Profile| D[Resilient Runner - internal/tester]
    D -->|Port Request| E[Port Manager - internal/concurrency]
    D -->|Cache Request| F[Result Cache - internal/cache]
    D -->|Test Stages| G[Execution Sockets/Xray subprocess]
    G -->|TestResult| H[Output Formatter - internal/output]
    H -->|Stdout| I[Raw URI with Latency]
```

## Key Interactions

1. **`models` is the Foundation:** All domain types are defined in `internal/models`. No package in `internal` or `pkg` should declare its own core proxy configuration or result types.
2. **Resilient Executions (`tester` + `concurrency` + `cache`):** During parallel runs, the `ResilientRunner` handles individual socket tests. It requests exclusive localhost ports from the `PortManager` (to run SOCKS5), skips redundant lookups via the DNS cache, and saves successful outputs to the `ResultCache`.
3. **Decoupled Error Recovery (`errors` + `tester`):** If a stage fails, the tester passes the error to `errors.Categorize()`. The output classification determines whether the runner retries (network error), skips (parse/schema error), or continues (lightweight handshake error).

## Conventions

- **Unwrap compatibility:** Avoid wrapping errors with plain strings; always use `%w` to preserve original sentinels for categorization matches.
- **Thread Safety:** Any shared resource (cache, port manager, process pool) must implement standard mutex safety (`sync.Mutex` or `sync.RWMutex`).
