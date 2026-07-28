# Viberayd Proxy Aggregation Server — Plan

## Overview

A server that ingests proxy sharelinks from subscriptions, tests them (latency + throughput + multi-domain health), selects the best one(s), runs them as Xray outbounds, and exposes a unified local proxy (SOCKS5/HTTP) for end users.

---

## 1. Share Link → Xray Config (DONE)

**Status: Complete — no reimplementation needed.**

The existing codebase at `internal/tester/xray_config.go` generates full, runnable Xray JSON configurations for all 5 protocols (Shadowsocks, VMess, VLESS, Trojan, Reality). Each generated config includes:

- SOCKS5 inbound on a specified port (127.0.0.1)
- Proxy outbound with full stream settings (tcp/ws/grpc/h2, TLS, Reality)
- Fallback direct + blackhole outbounds
- Routing rules

**What's needed for the server:** Instead of killing the Xray instance after a test, keep it running as a persistent proxy. The same `buildXrayConfig` function can be used — just don't stop the process.

---

## 2. Speed Testing

### Current state
The existing test pipeline does: TCP ping → TLS handshake → Protocol probe → Xray proxy test (simple HTTP GET via SOCKS5). This measures latency and basic connectivity, NOT throughput.

### What we need
A reliable speed test that measures **real throughput** (Mbps), not just ping latency.

### Approach: Multi-target throughput test

```
For each config that passed the basic tests (TCP + TLS + Protocol):
  1. Start Xray instance with the config as outbound
  2. Download test files from 3+ geographically distributed targets
     - speed.cloudflare.com (anycast, global)
     - self-hosted or known CDN files (10MB-100MB)
  3. Measure: latency (ms), download speed (Mbps), success rate
  4. Score = weighted combination of (latency × 0.3 + speed × 0.5 + reliability × 0.2)
  5. Kill Xray instance, record score
```

**Test targets:**
- `https://speed.cloudflare.com/__down?bytes=10000000` (10MB CF test)
- `https://proof.ovh.net/files/100Mb.dat` (OVH, Europe)
- `https://ash-speed.hetzner.com/100MB.bin` (Hetzner, Germany)
- User-configurable custom targets (via config)

**Measurement process:**
```go
func SpeedTest(ctx context.Context, proxyAddr string, targets []string, timeout time.Duration) SpeedResult {
    // For each target:
    //   start := time.Now()
    //   resp, err := http.Get(proxyAddr, targetURL)  // via SOCKS5
    //   bytesRead, _ := io.Copy(io.Discard, resp.Body)
    //   elapsed := time.Since(start)
    //   speedMbps = (bytesRead * 8) / elapsed.Seconds() / 1_000_000
    // Average across all targets, track failures
}
```

**Multi-domain health check (separate from speed):**
- Ping 5-10 known-good domains through the proxy
- Domains: google.com, youtube.com, github.com, cloudflare.com, wikipedia.org
- For each: TCP connect + TLS handshake + HTTP GET
- Score = percentage of successful checks
- Fail configs that can't reach basic internet

---

## 3. Proxy Server — Single Config (v1)

### Flow
```
1. Fetch subscriptions → parse sharelinks → deduplicate
2. Run lightweight filter (TCP ping) on all configs
3. Run full protocol + Xray test on survivors
4. Run speed test on top-K configs (based on latency from step 3)
5. Pick the best config (highest speed + lowest latency + best reliability)
6. Generate Xray config with:
   - SOCKS5 inbound on :1080
   - HTTP inbound on :8080
   - Selected config as the *only* outbound
   - All traffic routes through it
7. Start Xray as a persistent subprocess
8. Monitor health — if the config fails, fall back to the next-best
```

### Xray config for single-config mode
```json
{
  "inbounds": [
    { "tag": "socks-in", "protocol": "socks", "port": 1080,
      "settings": { "auth": "noauth", "udp": true }},
    { "tag": "http-in", "protocol": "http", "port": 8080,
      "settings": { "allowTransparent": false }}
  ],
  "outbounds": [
    { "tag": "proxy", "protocol": "vmess", ... },  // ← selected best config
    { "tag": "direct", "protocol": "freedom" },
    { "tag": "block", "protocol": "blackhole" }
  ],
  "routing": {
    "rules": [{ "type": "field", "outboundTag": "proxy", "network": "tcp,udp" }]
  }
}
```

### Health monitoring loop
```go
func (s *Server) healthLoop(ctx context.Context) {
    for {
        select {
        case <-ctx.Done(): return
        case <-time.After(30 * time.Second):
            if !s.checkProxyHealth() {
                s.switchToNextBest()
            }
        }
    }
}
```

---

## 4. Load Balancing — Multi-WAN (v2, post-implementation)

### Goal
Pick the top N configs by speed score, run them all as outbounds, and balance traffic across them without breaking sessions.

### Xray balancer approach
```json
{
  "outbounds": [
    { "tag": "proxy-0", "protocol": "vmess", ... },  // best
    { "tag": "proxy-1", "protocol": "vless", ... },  // second
    { "tag": "proxy-2", "protocol": "trojan", ... }, // third
    { "tag": "direct", "protocol": "freedom" },
    { "tag": "block", "protocol": "blackhole" }
  ],
  "routing": {
    "rules": [{
      "type": "field",
      "inboundTag": ["socks-in", "http-in"],
      "balancerTag": "wan-balancer"
    }],
    "balancers": [{
      "tag": "wan-balancer",
      "selector": ["proxy-"],
      "strategy": { "type": "leastPing" }
    }]
  }
}
```

### Session persistence
- Xray balancers work at the **connection level** — each new TCP connection is assigned to one outbound by the strategy
- A single TCP connection (HTTP/1.1 keep-alive, HTTP/2 stream) **stays on the same outbound** for its lifetime
- This means websites don't break: all packets for one TCP connection go through one WAN
- For HTTP/2 multiplexing: all streams share one TCP connection → all go through the same WAN ✓
- `leastPing` strategy dynamically routes new connections to the fastest WAN based on observatory data

### Adding/removing WANs dynamically
- When a new config is tested and scores well, add it to the outbounds and update the balancer
- When a config fails health checks, remove it from the outbounds
- Xray supports hot-reload via API (SIGHUP) or by rewriting config and restarting

---

## Architecture Diagram

```
┌──────────────┐     ┌───────────────────┐     ┌─────────────────┐
│  urls.txt    │────▶│  Fetcher + Parser │────▶│  State (DB/mem) │
└──────────────┘     └───────────────────┘     └─────────────────┘
                                                       │
                                                       ▼
                                              ┌─────────────────┐
                                              │  Candidate       │
                                              │  Selection       │
                                              └─────────────────┘
                                                       │
                                                       ▼
                                              ┌─────────────────┐
                                              │  TCP Ping Filter │
                                              └─────────────────┘
                                                       │
                                                       ▼
                                              ┌─────────────────┐
                                              │  Xray Test       │
                                              │  (protocol)      │
                                              └─────────────────┘
                                                       │
                                                       ▼
                                              ┌─────────────────┐
                                              │  Speed Test      │
                                              │  (throughput)    │
                                              └─────────────────┘
                                                       │
                                              ┌────────┴────────┐
                                              ▼                 ▼
                                     ┌──────────────┐  ┌──────────────┐
                                     │  Single WAN  │  │  Multi-WAN   │
                                     │  (pick best) │  │  (top N)     │
                                     └──────────────┘  └──────────────┘
                                           │                  │
                                           ▼                  ▼
                                     ┌──────────────┐  ┌──────────────┐
                                     │  Xray        │  │  Xray        │
                                     │  SOCKS5:1080 │  │  SOCKS5:1080 │
                                     │  HTTP:8080   │  │  HTTP:8080   │
                                     └──────────────┘  └──────┬───────┘
                                                              │
                                              ┌───────────────┴────────┐
                                              │  Balancer (leastPing) │
                                              │  proxy-0 proxy-1 ...  │
                                              └───────────────────────┘
```

---

## Implementation Steps

### Phase 1: Speed Test Module
- Implement `SpeedTest(ctx, proxyAddr, targets, timeout) → SpeedResult`
- Download test files from 3+ targets via SOCKS5
- Calculate Mbps, latency, reliability score
- Add multi-domain health check (5-10 domains)

### Phase 2: Proxy Server Mode
- New binary: `viberayd serve` (or subcommand)
- After testing, pick best config, start persistent Xray
- SOCKS5 inbound on :1080, HTTP inbound on :8080
- Health monitoring loop with failover
- Config hot-reload on SIGHUP

### Phase 3: Load Balancing
- Select top N configs
- Generate Xray config with multiple outbounds + balancer
- `leastPing` strategy with observatory
- Dynamic add/remove outbounds
- Graceful restart without dropping connections

### Phase 4: Observatory Integration
- Enable Xray's built-in observatory (ping to targets periodically)
- Feed observatory data into balancer decisions
- Automatic failover based on real-time health

---

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Config generation | Reuse existing `xray_config.go` | Already complete for all 5 protocols |
| Speed test method | HTTP download via SOCKS5 | Simple, uses existing infra, no new dependencies |
| Test file targets | speed.cloudflare.com + OVH + Hetzner + custom | Global coverage, large files available |
| Scoring | Weighted (latency × 0.3 + speed × 0.5 + reliability × 0.2) | Speed matters most for proxy use |
| Session persistence | Connection-level (Xray native) | TCP connections stay on one WAN, no session breakage |
| Balancer strategy | `leastPing` (v2) | Routes to fastest WAN dynamically |
| Config reload | SIGHUP or API | Standard pattern, no downtime |
