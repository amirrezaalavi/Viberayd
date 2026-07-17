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
| `quick` | **TCP** | Baseline network connectivity, port accessibility, and latency. |
| `standard` | **TCP + TLS** | Quick + certificates, ALPN, cipher suites, SNI, and TLS handshake latency. |
| `full` | **TCP + TLS + Protocol** | Standard + light protocol payload checks and parameter/cipher validations. |
| `comprehensive` | **TCP + TLS + Protocol + Proxy** | Full + launching xray-core locally, routing standard SOCKS5 requests through the proxy backend to verify authentic end-to-end data delivery. |

---

## Xray Process Routing (`xray.go`)

At `comprehensive` depth, Viberay does a real proxy handshake:
1. Generates a custom xray JSON configuration using `xray_config.go` with a SOCKS5 inbound routing to the selected proxy outbound.
2. Starts `xray` using the path defined by the `-xray-bin` flag.
3. Performs a SOCKS5 protocol greeting (`testViaSOCKS5`) on the local loopback port.
4. Cleans up the background process and temporary JSON configuration files.

---

## Resilient Execution (`resilience.go`)

The `ResilientRunner` handles transient network interruptions:
- **Retry Bounds:** Up to the limits defined by `MaxRetries` with random jittered exponential backoffs (capped at `30s` to prevent runner hangs).
- **Parallelism Throttling:** If it encounters system resource exhausted errors (`ErrResourceExhaust`), it automatically decreases the running worker concurrency.

---

## Conventions & Gotchas

- **SOCKS5 Greeting limit:** The `testViaSOCKS5` method checks the SOCKS5 greeting/greeting handshakes to ensure authentication works, but does not execute a full HTTP GET query. This avoids importing third-party network libraries.
- **Sorting Priorities:** Configs are sorted using `ConfigPriority()` (Reality > VMess > VLess > Trojan > SS) to test high-signal, secure configs first.
- **Port Manager Dependency:** In `comprehensive` testing, a unique local SOCKS5 port must be acquired from the `PortManager` before initiating the Xray subprocess.
