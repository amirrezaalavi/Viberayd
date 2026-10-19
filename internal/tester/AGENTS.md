# Tester Package — Agent Guide

This package forms VibeRay's multi-stage testing core. It executes connectivity and handshake validation checks across 4 progressive depth levels.

## Purpose

To implement concrete validation stages (TCP, TLS, Protocol, and Xray-proxy routing). By running tests incrementally, nodes fail-fast, preventing CPU and port resources from being wasted on dead connections.

---

## Files

- `tcp.go`: Performs lightweight socket dialing (`net.DialTimeout`) to measure baseline connection latency.
- `tls.go`: Performs TLS client handshakes (`crypto/tls`) with certificate validation and custom browser fingerprint simulations.
- `protocol.go`: Runs lightweight protocol probes (verifies cipher lists, parameters, and checks destination TLS capabilities for Reality).
- `xray.go`: Spawns and manages xray-core subprocesses, verifying connectivity over local SOCKS5 proxy listeners.
- `xray_config.go`: Programmatically creates complete JSON configurations for the 5 protocol variants for xray-core outbound routers.
- `pipeline.go`: Connects the sequential stages (TCP → TLS → Protocol → Proxy) using a gate-controlled Depth level.
- `resilience.go`: Implements the `ResilientRunner` wrapping executions with randomized exponential backoff and dynamic parallelism throttling.
- `tester_test.go` and `resilience_test.go`: Unit tests mocking network sockets, TLS connections, and validation checks.

---

## Depth Levels & Stages (`pipeline.go`)

Each configuration is tested sequentially. It stops immediately at the depth limit:

| Depth Level | Stages Run | Validation Goals |
|---|---|---|
| `quick` | **TCP + TLS + Protocol** | Connectivity, TLS handshake, and light protocol checks — no subprocess spawned. |
| `standard` | **Quick + Xray proxy** | Plus a real HTTP request through a temporary xray SOCKS inbound; a **2xx** status is required (block pages and errors prove nothing). |
| `full` | **Standard + Domain content check + Stability** | Plus `CheckDomains` (quorum of domains returning 2xx with `DomainMinBytes`), then `StabilityProbes` independent end-to-end verifications. |
| `comprehensive` | alias of `full` | Normalized to `full`; kept so existing deployments and env files keep working. |

**Stability stage.** A single passing sample is not evidence of a usable
tunnel: re-checking the served set three times showed 36% of links failing at
least one run. After the depth-specific stages pass, `StabilityProbes-1`
additional full verifications run (`StageStability`) and any failure rejects
the config. `StabilityProbes=1` disables the gate.

A config is only marked `working` after the xray stage; `quick` never marks a
config working end-to-end and is intended for fast reachability sweeps.

---

## Xray Process Routing (`xray.go`)

Every depth that needs a real tunnel does a real proxied HTTP request:
1. Generates a custom xray JSON configuration using `xray_config.go` with a SOCKS5 inbound routing to the selected proxy outbound (`withTempXray`).
2. Starts `xray` and **polls the SOCKS listener** (up to `listenerWait`, 5s) instead of sleeping a fixed 300ms — a fixed sleep races under parallel load and produces "connection refused" false negatives.
3. Requests each target URL in turn while the process is alive; success requires a 2xx.
4. Cleans up the background process and the temporary JSON configuration file. If the listener never comes up, the captured xray stderr is included in the error.

---

## Resilient Execution (`resilience.go`)

The `ResilientRunner` handles transient network interruptions:
- **Retry Bounds:** Up to the limits defined by `MaxRetries` with random jittered exponential backoffs (capped at `30s` to prevent runner hangs).
- **Parallelism Throttling:** If it encounters system resource exhausted errors (`ErrResourceExhaust`), it automatically decreases the running worker concurrency.

---

## Conventions & Gotchas

- **Real HTTP, not a greeting:** `testViaSOCKS5` issues a full `http.Client` GET through the SOCKS5 proxy and requires a 2xx — a bare SOCKS greeting is not treated as proof of working egress.
- **Sorting Priorities:** Configs are sorted using `ConfigPriority()` (Reality > VMess > VLess > Trojan > SS) to test high-signal, secure configs first.
- **Port Manager Dependency:** Every depth that spawns xray acquires a unique local SOCKS5 port from the `PortManager` before starting the subprocess.
