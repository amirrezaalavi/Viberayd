# Test Data

This directory contains real-world and synthetic test fixtures used by VibeRay tests and for manual experimentation.

## Directory layout

```
testdata/
├── working/proxies.txt        # 16 known-working configs (end-to-end tested 2026-07)
├── not-working/proxies.txt    # 15 known-failing configs
├── examples/                  # One minimal config per protocol (SS, VMess, VLess, Trojan, Reality)
│   ├── ss.txt
│   ├── vmess.txt
│   ├── vless.txt
│   ├── trojan.txt
│   └── reality.txt
└── subscriptions/             # 1,916-config subscription split into 5 chunks of ~400
    ├── chunk_000.txt
    ├── chunk_001.txt
    ├── chunk_002.txt
    ├── chunk_003.txt
    └── chunk_004.txt
```

## Usage

Quick sanity test against the working set (parser + quick TCP test):
```bash
viberay -input testdata/working/proxies.txt -depth quick -timeout 3s
```

Test the failure-detection path:
```bash
viberay -input testdata/not-working/proxies.txt -depth quick -timeout 2s -out-dir ./out
```

Process a chunked subscription (recommended over feeding the full 1,916-line `chunk`):
```bash
viberay -input testdata/subscriptions/chunk_000.txt -depth standard -concurrency 20
```

## Notes

- All configs here are scraped from public sources. Some hosts/ports are intentionally dead — the `not-working/` set is for verifying failure-handling, not for daily use.
- The subscription chunks are derived from a single ~800 KB subscription, split at 400 lines per file to keep individual runs under the 100-port cap and within the 30 s per-test timeout.
- `test.txt` at the repo root is the original un-split subscription (1,916 lines). It is preserved for one-shot bulk runs but the chunks are preferred for incremental work.
