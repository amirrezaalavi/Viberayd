# Orchestrator Package — Agent Guide

This package forms VibeRay's heuristic decision layer. It analyzes the runtime context (system capability and batch input profile) and automatically configures ideal concurrency, depth, and retry strategies.

## Purpose

Instead of asking users to manually supply tuning flags (concurrency, depth, timeouts, caching options) for every run, the orchestrator acts as a smart runtime decision engine. However, any user-supplied CLI flags will override the orchestrator's recommendation.

---

## Files

- `decision.go`: Key structures (`UserPreferences`), context builder (`BuildContext`), and the central decision algorithm (`Decide()`).
- `decision_test.go`: Comprehensive unit and regression testing confirming heuristics for various batch sizes, protocols (e.g., Reality triggers), and concurrency caps.

---

## Decision Logic & Heuristics

The `Decide` function evaluates the parsed context (`models.TestContext`) against user constraints:

### 1. Depth Selection (`decideDepth`)
- **0 configs**: defaults to `quick` (TCP only).
- **1–10 configs**: defaults to `comprehensive` (TCP + TLS + Protocol + Xray SOCKS5 verification).
- **11–100 configs**: defaults to `full` (TCP + TLS + Protocol).
- **101–500 configs**: defaults to `standard` (TCP + TLS).
- **500+ configs**: defaults to `quick` (TCP only).
- **Reality Premium Bonus**: If any Reality proxy is present in the batch, the depth is bumped by exactly **one level** (up to a cap of `comprehensive`) since Reality configs represent higher-signal modern infrastructure.

### 2. Output Formatting (`decideStyle`)
- **≤ 100 configs**: defaults to `table`.
- **> 100 configs**: defaults to `csv` (optimized for large streams).

### 3. Concurrency Limits (`decideConcurrency`)
- Calculates workers dynamically based on CPU core count: `min(cpu * 2, 100)`.

### 4. Retry Policy (`decideRetry`)
- **≤ 100 configs**: 2 retries (with exponential backoff).
- **> 100 configs**: 1 retry (fail fast to speed up batch runs).

### 5. Caching Strategy (`decideCache`)
- Automatically enabled if duplicate server/ports represent **more than 5%** of the batch input, reducing repetitive DNS/TCP requests.

### 6. Handshake Timeout (`decideTimeout`)
- Defaults to `5s`.
- Automatically increases to `7s` if any Reality configuration is present to accommodate slower cryptographic handshakes.

---

## Key Functions & APIs

- **`BuildContext(configs []models.ProxyConfig, parseErrors int) TestContext`**: Instantiates a system info analyzer (detects CPU count, available RAM, runtime OS, and checks if `xray` exists in path) and inspects input duplicate ratios.
- **`Decide(ctx models.TestContext, prefs UserPreferences) (OrchestratorDecision, error)`**: Analyzes context and user preference overrides, producing the final runtime directive structure.

---

## Conventions & Gotchas

- **CLI Overrides:** UserPreferences uses pointer or zero-value checks. A non-zero preference or non-nil pointer (like `Cache *bool`) explicitly overrides corresponding heuristics.
- **Linear Duplicate Scan:** Duplicate count is currently checked using a simple linear tracker. It runs in $O(N)$ with respect to the input array size (which is highly optimized in Go).
