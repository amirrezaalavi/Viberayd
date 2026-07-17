# Cache Package — Agent Guide

This package provides TTL-based Least-Recently-Used (LRU) cache layers for DNS queries and test results, optimizing performance across large batches of duplicated server nodes.

## Purpose

Large subscription files often contain redundant nodes or repeatedly refer to the same backend IP addresses/domains. Caching these network transactions prevents duplicate TCP/DNS lookups and handshake delays, substantially speeding up execution.

---

## Files

- `dns.go`: LRU lookup cache for resolving hostnames to IP addresses with a custom TTL (default: 5 minutes, max size: 1000).
- `result.go`: LRU test results cache keyed by protocol, server, and port to immediately answer tests for duplicate nodes (default TTL: 10 minutes, max size: 500).
- `cache_test.go`: Validates concurrency safety, item expiration, eviction on exceeding cap boundaries, and hit/miss stats tracking.

---

## Key Structures & APIs

### 1. DNS LRU Cache (`DNSCache`)

Bypasses core operating system socket resolve lookups for repetitive domains.
- **`NewDNSCache(maxSize int, ttl time.Duration) *DNSCache`**: Instantiates a cache.
- **`Lookup(host string) ([]string, error)`**: Returns a cached array of resolved IP addresses if still valid. Otherwise, triggers a thread-safe `net.LookupHost` call and inserts the result back into the cache.

### 2. Test Result Cache (`ResultCache`)

Reuses completed test runs for duplicate server configs, allowing rapid skips during testing.
- **`KeyFor(cfg models.ProxyConfig)`**: Generates a composite key formatted as `protocol://server:port` (e.g. `reality://1.1.1.1:443`).
- **`Get(key string) (TestResult, bool)`**: Returns the cached test status and staging details if present and unexpired.
- **`Put(key, result)`**: Caches a successful test result, automatically pushing it to the front of the LRU sequence.
- **`Stats() (hits, misses, size, max)`**: Returns metric tracking statistics used by the orchestrator.

---

## Conventions & Gotchas

- **Double-Lock Safety:** Cache structs utilize `sync.Mutex` or `sync.RWMutex` to ensure thread safety across concurrent pool workers.
- **Eviction Strategy:** When the container reaches its capacity, the oldest un-accessed item is evicted utilizing the doubly-linked list (`container/list`) pointer.
- **Short-Circuit on Failures:** The result cache should typically store only successful results (`StatusSuccess`) so that temporary network issues on one node don't falsely cause duplicates to skip real network test validation.
