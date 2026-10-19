# Viberayd Quality Verification Shift — Plan

**Status:** proposed, awaiting go-ahead. No product code changed yet.
**Branch:** `fix/test-quality-verification` off upstream `main` `b07c639` (in `/home/netcon/playground/ai/Viberayd-fix`, remote `fork` → `yolka-wiz/Viberayd`).
**Scope (confirmed):** Viberayd first, then Viberoxy.
**Artifact under test (confirmed):** `working.txt` lines.
**Box usage (confirmed):** build + restart the real services on `core@192.168.4.120`.
**Acceptance (confirmed):** a config is "working" only if a real HTTP 204 goes through the proxy **and** the exact line written to `working.txt` re-parses in a strict client parser, and it stays up across N probes.

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

## 3. Phases

### P0 — Repro harness + golden corpus (no box changes)
Deliverables in `internal/qualitytest/` + `cmd/qualitycheck/`:
- `parse-serve-roundtrip`: for each link, assert `parse → render → serve` yields a line that `ParseSingle` accepts byte-exact (catches the latency suffix and any future re-encoding).
- `end-to-end`: start xray on a scratch port from the built JSON, `GET https://www.gstatic.com/generate_204` through SOCKS, require HTTP 204 within timeout; optionally N stability probes.
- `reference-verdict`: given a `working.txt` line, produce `usable | unparseable | no-traffic`.
- Golden corpus `testdata/quality/*.txt`: ~30 links selected from live `state.json`, covering every protocol and transport (ws/grpc/xhttp/tcp+http-header/reality/tls/none), each with an expected verdict recorded from a manual run.
- Subagent A audits the corpus selection; Subagent B independently re-derives expected verdicts to catch harness bias.

### P1 — Fix section 3 (served-sharelink corruption)
> **Contract conflict to resolve:** `AGENTS.md`/`README` document `working.txt` as `<raw-uri> <latency-ms>`, yet `http.go:98` serves that exact file as `/sub`. A file cannot be both a human report with latency and a valid subscription. Since **`working.txt` is the artifact you consume**, the plan is to make the file a pure sharelink list and move latency elsewhere; the documented format is updated in the same commit.
- `loop.go`: write **pure** `entry.Raw`; sort by `entry.LatencyMs` directly; drop `parseLatencyMs`.
- Keep latency in `state.json` (already there) / `/api/stats`; add `working.csv` (`uri,latency_ms`) if a flat latency file is still wanted — never inside the URI.
- Update `AGENTS.md`/`README` config reference for the new `working.txt` contract.
- Delete `configToURI` and its exporter usage (or route `Export` through `r.Config.Raw`).
- Tests: `/sub` round-trips every protocol with a strict decoder; latency-suffix regression test.
- *Alternative if you'd rather keep the documented format:* leave `working.txt` as `<uri> <latency>` and make `handleSub` emit clean URIs from `State`. Slightly more code, no change to the human file — tell me if you prefer this.

### P2 — Fix section 1 (test depth)
- Make `comprehensive` the default; when depth < `comprehensive`, configs may be marked `probe_passed` but **never** `working` unless an explicit opt-out flag is set (additive, keeps the state machine intact).
- Startup log + `/api/stats` must state the effective depth so a misconfigured deployment is obvious.
- Reconcile `working.txt` count vs `/api/stats` count (currently 29 vs 23).

### P3 — Fix section 2 (protocols / transports)
- Viberayd: classify `tuic`/`hysteria2` honestly — either a sing-box sidecar tester (flagged, staged) or "unsupported: never marked working". Verify `wireguard` really loads in xray.
- Port the transport matrix fixes (xhttp settings, grpc `serviceName`, kcp/quic/h2, vmess `scy`/skip-verify) with per-transport golden tests.
- Viberoxy (second): replace the `freedom` fallback with a hard error; fix the same transport gaps.

### P4 — Live validation on the box
- Isolated dev instance first: `/opt/viberayd-quality` (own `urls.txt`/`state.json`/`working.txt`, ports `:8094/:8095`, `parallel=20`, `timeout=6..8s`, `cycle_sleep=10`) with a curated ~2–3k-config sample for fast cycles.
- Baseline vs branch: compute FP/FN on the sample with `reference-verdict`.
- Final: back up `/opt/viberayd/{working.txt,state.json}` + current binary/unit, deploy my branch to `/opt/viberayd-src`, restart, confirm `/sub` is clean and Viberoxy ingests strictly more configs (vmess no longer dropped).

---

## 4. Evidence to produce

- Before/after table: FP rate, FN rate, per-protocol pass counts, `/sub` strict-parse rate (target 100%).
- Hand-checked samples: N chosen sharelinks per protocol, each shown as `line → strict parse → HTTP 204 result`.
- `go test ./...` + `go vet ./...` green; new golden tests fail on `b07c639` and pass on the branch.

## 5. Risks & rollback

| Risk | Mitigation |
|---|---|
| Breaking the live subscription | Dev instance on separate ports first; back up `working.txt`/`state.json`/binary/unit before the prod restart |
| Removing latency from `working.txt` surprises a consumer | Viberoxy only needs the URI; latency stays in `state.json`/API; add `working.csv` |
| Depth default change slows cycles | Sample-based dev runs; cost measured before the prod restart |
| sing-box sidecar scope creep | Staged behind a flag; default is honest "unsupported" classification |

## 6. Still needed from you

1. Approval to proceed (no code touched yet).
2. Permission to add a temporary isolated instance on the box (extra ports/dirs) before the real restart — the real restart is already approved.
3. Whether the 4 current `urls.txt` entries are the set to use, or a curated/known-good list.
4. Push target confirmation for the branch (your fork `yolka-wiz/Viberayd` vs keep local).
