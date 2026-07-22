# Test Run Results

Recorded 2026-07-22 with `viberay -input <chunk> -depth quick -timeout 2s -concurrency 30 -quiet`.

| Chunk | Lines | Pass | Pass rate |
|-------|-------|------|-----------|
| chunk_000.txt | 400 | 155 | 38.8% |
| chunk_001.txt | 400 | 317 | 79.3% |
| chunk_002.txt | 400 | 364 | 91.0% |
| chunk_003.txt | 400 | 368 | 92.0% |
| chunk_004.txt | 316 | 191 | 60.4% |
| **Total** | **1,916** | **1,395** | **72.8%** |

## Observations

- `chunk_000` is heavily degraded — many dead VMess/WebSocket endpoints. Most configs there are `series-v1.samanehha.co` and similar subdomains that have since been rotated out.
- `chunk_002` and `chunk_003` are the freshest, with Reality + VLess+TCP endpoints dominating the working set.
- Median latency across all passing configs at `-depth quick` (TCP only) is well under 1 s.
- For deeper validation use `-depth standard` (TCP+TLS) or `-depth full` (adds protocol handshake) on individual chunks; expect 10-30 s per chunk on a fast network.

## Reproducing

```bash
# Quick baseline (TCP only)
for i in 000 001 002 003 004; do
  viberay -input testdata/subscriptions/chunk_$i.txt \
          -depth quick -timeout 2s -concurrency 30 -quiet \
    | wc -l
done

# Standard depth (TCP + TLS) — slower, more accurate
viberay -input testdata/subscriptions/chunk_002.txt -depth standard -timeout 5s -concurrency 20

# Save working configs to disk
viberay -input testdata/subscriptions/chunk_002.txt -depth standard \
        -out-dir ./out
```
