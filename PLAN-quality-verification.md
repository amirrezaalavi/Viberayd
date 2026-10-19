# Viberayd Quality Verification Shift — Plan

**Status:** in progress. P1 + parser/protocol fixes done, verified live, deployed.
**Branch:** `fix/test-quality-verification` off upstream `main` `b07c639`; pushed to `amirrezaalavi/Viberayd` (push to `yolka-wiz/Viberayd` is denied — that token is not a collaborator there).
**Deployed:** `a472f92` on `core@192.168.4.120` (`viberayd.service`), backups `*.bak-20261019-*` in `/opt/viberayd` and `/opt/viberayd-src`.
**Scope (confirmed):** Viberayd first, then Viberoxy.
**Artifact under test (confirmed):** `working.txt` lines.
**Box usage (confirmed):** build + restart the real services on `core@192.168.4.120`.
**Acceptance (confirmed):** a config is "working" only if a real HTTP 204 goes through the proxy **and** the exact line written to `working.txt` re-parses in a strict client parser, and it stays up across N probes.

---

## 0. Results so far (all measured on the box)

| Check | Before | After |
|---|---|---|
| `/sub` lines carrying a latency suffix | 29/29 | **0/29** |
| `/sub` strict round-trip (parser) | 28/29 (1 malformed vmess) | **25/25, 0 malformed** |
| End-to-end re-check of served links | 3/25 failed to even bind | **25/25 pass** (after listener fix) |
| Parse loss on a real 128-link sample | 33 malformed | **4 malformed** (all invalid UUIDs) |
| Live configs recovered from the same 4 feeds | 647,193 | **649,000** (+1,807) |
| hysteria2/tuic classification | invalid xray outbound, always "failed" | explicit "cannot be tested" fast-fail |
| Served links stable 3/3 (live set, 25 links) | 16/25 (64%) | **16/22 (73%)** on the live gated set; 5 pass 2/3, 1 passes 1/3, none 0/3 |
| Cycle duration (649k candidates, full depth) | 4m42s (single pass) | 49m57s with the gate, **16m33s** with the xray-stage cap |

Stability gate: re-checking the live served set three times showed 16 links
passing 3/3, 5 passing 2/3 and 4 passing 1/3 (none 0/3) — a single sample is
not evidence of a usable tunnel. `DAEMON_STABILITY_PROBES` (default 2) now
requires two independent end-to-end passes. On the 128-link sample the gate
cut working from 7 to 5 and all 5 then passed 3/3 re-checks.

While wiring this I found a second real bug: `XrayTest` wrapped the whole
config in **one** `DAEMON_TIMEOUT` context although each stage has its own
timeout, so the extra probe was cancelled mid-flight (sample fell to 1
working, all failing with "context deadline exceeded"). `Pipeline.Budget()`
now sizes the context from the stages, and the gate behaved as designed.

**Cycle time, and the cost of the cap (measured).** With `DAEMON_STABILITY_PROBES=2`
the sweep went from ~4m42s to **49m57s**: the per-config context was widened to
`Pipeline.Budget()` (~70s) so the probes would not be cancelled, which also let
every TCP-reachable-but-dead candidate burn its full stage timeouts (3353
"failed" × ~9s at parallel 10). `DAEMON_MAX_XRAY_TESTS` (default 1200) now caps
how many TCP survivors reach the deep stages, keeping the fastest handshakes:
the next cycle took **16m33s** (survivors 5748 → kept 1200, slowest kept 115ms).
The trade-off is real and unfixed: 4548 survivors were dropped on handshake
latency alone, and the hash tie-break is stable, so a config that is
consistently slow to connect may never be tested. A rotating cursor over the
dropped set (keep N by latency, fill the rest round-robin) is the next change
if that bias matters.

**Key artifacts**
- `cmd/qualitycheck` — strict-parse + optional end-to-end verifier; non-zero exit on any annotated/malformed/failing line.
- `internal/tester/xray.go` — listener poll instead of a 300ms sleep; per-target fallback while xray is alive; xray stderr captured; 2xx required.
- `internal/parser/*` (via `pkg/parser`) — unpadded/base64url decode, numeric vmess `port`/`aid`, bounded repair of glued vmess payloads.
- `wip/deployed-20260912` — the ~290 lines of uncommitted work found in `/opt/viberayd-src` (the `CheckDomains` content check), preserved and merged rather than dropped.

---

## 1. What recon found (hand-verified evidence)

Live stack on the box:

| Component | Path / branch | Role |
|---|---|---|
| `viberayd` | `/opt/viberayd-src` @ `b07c639` | tests configs, writes `/opt/viberayd/working.txt`, serves `/sub` on `:8084`, API `:8085` |
| `viberoxy` | `/opt/viberoxy-src` @ `ad87bdf` | consumes `/sub`, speed-tests, promotes to WAN slots, proxy `:10908` |
| `viber-console` | `/opt/viber-console-src` | UI |

Local `Viberayd/` is byte-identical to upstream `main` `b07c639`.
Live state: 647,185 configs, 29 lines in `working.txt` (API says 23 working).

### Section 3 — the served sharelink is corrupted (confirmed)
- `internal/daemon/loop.go:226` builds the line as `fmt.Sprintf("%s %dms", entry.Raw, entry.LatencyMs)` — **latency is appended inside the URI**.
- `internal/daemon/http.go:98` base64-encodes that file verbatim as `/sub`.
- Live check: 29/29 `/sub` lines end in ` <n>ms`. A strict base64 decode of the single `vmess://` line **fails at byte 320** (Go's `base64.StdEncoding`, which Viberoxy uses) → the config is silently dropped by every strict consumer. `vless`/`ss` names get ` 6380ms` glued into the fragment.
- Sorting in `writeOutputFile` depends on re-parsing that suffix (`parseLatencyMs`), even though `entry.LatencyMs` is already available.
- `internal/output/exporter.go:77 configToURI` is a lossy reconstructor (its own comment: "not guaranteed to be round-trippable"); dead code today, latent landmine.

### Section 1 — not a full test
- `internal/daemon/config.go` default `Depth: "standard"` = TCP + TLS only. No proxy test at all.
- `internal/tester/protocol.go` — the `full` level's "protocol handshake" re-dials TCP and validates fields; it is **not** a handshake (the file's own comments say so). Only `comprehensive` runs a real proxied request.
- `internal/tester/xray_config.go` emits `tuic` and `hysteria2` outbounds that xray-core cannot load → those ~7,978 configs can only fail (false negatives), never pass.

### Section 2 — protocol coverage / wrong tests
- Viberoxy `internal/xraycfg/build.go` maps `hysteria2`/`tuic`/`wireguard`/unknown → **`freedom` (direct)**. Guarded by `IsXraySupported` at `main.go:328` and `:957`, so no live leak today, but those protocols are silently untestable and the guard is the only thing preventing a leak.
- `buildStreamSettings` gaps: `xhttp` validates path/host/mode then never emits `xhttpSettings`; `kcp`/`quic`/`h2` silently become `tcp`; grpc reads `path` instead of `serviceName`; vmess ignores `scy` / `skip-cert-verify`.
- Viberoxy's tester itself is sound (real download + exit-IP probes through a temp xray).

---

## 2. Objectives

1. **No false positives.** Nothing reaches `working.txt` that a strict client cannot parse *and* actually use.
2. **Fewer false negatives.** Every protocol/transport we claim to support is tested with an outbound that real clients can also use.
3. **Reproducible evidence.** A harness that fails loudly when a served line is not a valid sharelink.
4. **Measurable before/after.** A false-positive/false-negative rate computed the same way on both builds.

**Non-goals:** rewriting the pipeline, changing the state machine, UI work, performance tuning beyond what fast iteration needs.

---

## 3. Phases (status)

### P0 — Repro harness — DONE
`cmd/qualitycheck` reads a `/sub` URL or file, base64-decodes subscriptions,
and reports per line: strict round-trip, annotation (` 123ms`), and with `-e2e`
the real proxied HTTP result. Non-zero exit when any line is annotated,
malformed, or fails to carry traffic. Unit-tested.
(Still open: a checked-in golden corpus and independent subagent audits.)

### P1 — Section 3, served-sharelink corruption — DONE, deployed
- `loop.go` writes one verbatim `entry.Raw` per line; sorting uses `entry.LatencyMs`.
- Latency moved to a sibling `working.csv` + `state.json` + API.
- `exporter.go` no longer reconstructs URIs (`configToURI` deleted).
- `AGENTS.md` / `internal/tester/AGENTS.md` document the new contract.
- Live result: 25/25 strict parse, 0 annotations, 25/25 e2e after the listener fix.

### P2 — Section 1, test depth — MOSTLY DONE (partly by the recovered WIP)
- The recovered `wip/deployed-20260912` reworked depth: `quick` (no subprocess),
  `standard` (real xray proxy test, 2xx required), `full` (plus the multi-domain
  content check), `comprehensive` = alias of `full`.
- Default depth in `config.go` is still `standard`, which now includes the real
  proxy test — acceptable, but `full` is the stricter option.
- Open: reconcile `working.txt` count vs `/api/stats`, and state the effective
  depth in the API.

### P3 — Section 2, protocols / transports — PARTIAL
Done, verified against Xray 26.7.11:
- `tuic` and `hysteria2` are **not** xray-core protocols; the invalid outbounds
  were replaced with an explicit fast-fail (no wasted subprocess).
- `wireguard` is a real xray outbound and is still tested.
- Parser now recovers 30% more links from real feeds (unpadded/base64url,
  numeric vmess `port`/`aid`, glued payloads).
Open:
- Transport matrix: `xhttp` validates path/host/mode then never emits
  `xhttpSettings`; `kcp`/`quic`/`h2` silently become `tcp`; grpc reads `path`
  instead of `serviceName`; vmess ignores `scy` / `skip-cert-verify`.
- Optional sing-box sidecar so tuic/hysteria2 can actually be tested.

### P4 — Live validation — DONE
Isolated instance at `~/viberayd-quality` (own state, ports `:8094/:8095`, xray
ports `11100+` via the new `DAEMON_PORT_BASE`), then the real service.

---

## 4. Remaining work (next rounds, priority order)

1. **Bound each cycle.** `SelectCandidates` returns every `failed`/`unreachable`
   config every cycle: 649,000 candidates. A full cycle takes hours, so
   `working.txt` rarely refreshes and most configs are never reached — a direct
   cause of "not enough working configs". Add a due-time for failed/unreachable
   (like `RetestInterval` does for working) and/or a per-cycle candidate cap.
2. **Stability, not a single sample.** Measured 6/7, 6/7, 7/7 on identical
   links. `ConfigEntry.SuccessCount` already exists: require N consecutive
   successes before a config enters `working.txt`.
3. **Transport matrix** (P3 open items above) with per-transport tests.
4. **Viberoxy** (second repo): replace the `freedom` fallback with a hard error,
   port the same transport fixes, and re-verify it ingests the now-clean `/sub`.
5. **Golden corpus + independent audit** (subagent A selects links, subagent B
   re-derives expected verdicts) to keep the harness honest.
6. **Decide the push destination.** Pushes go to `amirrezaalavi/Viberayd`;
   `yolka-wiz/Viberayd` rejects the current token (not a collaborator).

## 5. Rollback

Old binary and data are on the box: `/opt/viberayd-src/viberayd.bak-20261019-*`,
`/opt/viberayd/{working.txt,working.csv,state.json}.bak-20261019-*`. Restore the
binary and `systemctl restart viberayd` to revert.

## 6. Still needed from you

1. Whether `standard` or `full` should be the default depth (cost vs strictness).
2. Whether to pursue a sing-box sidecar for tuic/hysteria2, or keep the honest
   fast-fail.
3. Confirmation of the push destination (this session pushes to
   `amirrezaalavi/Viberayd`).
