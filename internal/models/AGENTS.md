# Models Package — Agent Guide

This package contains the core data types, configuration schemas, and context structs used throughout the VibeRay pipeline. It represents the single source of truth for all domain types.

## Purpose

To provide unified domain definitions for proxy configurations (using a tagged-union model), testing metrics, system metadata, validation results, and execution statistics. All structures are decorated with JSON and YAML annotations for forward-compatible serialization.

---

## Files

- `config.go`: Defines the core `Protocol` types, individual configuration schemas for all 5 supported protocols (Shadowsocks, VMess, VLESS, Trojan, Reality), and the unified `ProxyConfig` tagged-union struct.
- `result.go`: Defines the testing status, stage constants, timing breakdown structures, and execution aggregations (`TestResult`, `Summary`, and `ValidationResult`).
- `context.go`: Contains systemic metadata schemas for the AI heuristic decision engine (`TestContext`, `SystemInfo`, `InputStats`, `RuntimeState`, etc.) as well as the output decision schema `OrchestratorDecision`.
- `models_test.go`: Unit tests ensuring proper accessor performance, string formats, and JSON serialization.

---

## Protocol Configurations (`config.go`)

Proxy configs are structured as a **tagged-union** inside `ProxyConfig`. This guarantees there is no data loss or conversion ambiguity during execution.

```go
type ProxyConfig struct {
    SS      *SSConfig
    VMess   *VMessConfig
    VLess   *VLessConfig
    Trojan  *TrojanConfig
    Reality *RealityConfig
    Raw     string `json:"raw,omitempty" yaml:"raw,omitempty"` // Original un-parsed share link
}
```

### Protocol Types

- **Shadowsocks (`SSConfig`)**: Host, Port, Method, Password, Plugin, and Plugin Options.
- **VMess (`VMessConfig`)**: Host, Port, UUID, AlterID, Security, and TLS configuration.
- **VLESS (`VLessConfig`)**: Host, Port, UUID, Flow control, Encryption, and TLS.
- **Trojan (`TrojanConfig`)**: Host, Port, Password, Flow, and TLS.
- **Reality (`RealityConfig`)**: Host, Port, VLESS UUID, Flow, Public Key, Short ID, SpiderX target, and TLS.

### Accessor Methods on `ProxyConfig`

Since accessors are value receivers, **do not take the address of a `ProxyConfig` with the intention of mutating its inner pointers via accessors**.
- `Protocol() Protocol`: Returns the active protocol (e.g. `ss`, `vmess`, etc.).
- `Name() string`: Extracts the remarks/name segment.
- `Addr() string`: Formats host and port as a standard `"host:port"` dial-ready string.
- `Base() BaseConfig`: Returns the shared `BaseConfig` containing fields like Server, Port, Network, etc.
- `String() string`: Returns a brief human-readable identifier.

---

## Testing Stages & Results (`result.go`)

### Pipeline Stage Constants

```go
const (
    StageTCP       = "tcp"
    StageTLS       = "tls"
    StageProtocol  = "protocol"
    StageProxy     = "proxy"
    StageCompleted = "completed"
)
```

### Test Outcomes (`TestStatus`)

- `success`: Fully completed up to target depth without failure.
- `failed`: Handshake or ping failed during one of the active stages.
- `error`: Non-protocol system exceptions (e.g., port exhaustion, crashed binary).
- `skipped`: Dropped prior to running (duplicate, cached, or manually excluded).

---

## AI Orchestration Context (`context.go`)

- **`TestDepth`**: Sets pipeline gate-level (`quick`, `standard`, `full`, `comprehensive`).
- **`TestContext`**: Sent to `orchestrator.Decide()` to evaluate CPU counts, OS types, subscription duplicates, and past performance.
- **`OrchestratorDecision`**: The finalized runtime profile holding worker pool sizes, retry counts, per-config timeouts, and cache rules.

---

## Conventions & Gotchas

- **Raw Field Matters:** Always preserve the exact input string in `ProxyConfig.Raw`. The formatters use this field directly to prevent any data loss when outputting successful configs.
- **Value Receivers:** Accessors on `ProxyConfig` use value receivers. Directly editing fields should be done on the inner concrete configs.
