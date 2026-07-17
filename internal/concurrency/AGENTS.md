# Concurrency Package — Agent Guide

This package manages VibeRay's multi-threaded system limits. It contains the worker pool, a local port binder, and a reusable Xray-core process pool.

## Purpose

High-performance testing requires running dozens of network and protocol handshakes concurrently without colliding. The concurrency layer coordinates thread limits, local SOCKS5 proxy port binds, and manages Xray background process lifecycles.

---

## Files

- `pool.go`: Bounded Go worker pool (`Pool`) implementing graceful shutdown, semaphore-based thread control, and worker-level error propagation.
- `port.go`: Safe port allocator (`PortManager` and `StaggeredAllocator`) that verifies system-level TCP port availability before allocation to prevent SOCKS5 collision.
- `xray_pool.go`: Reusable Xray process manager (`XrayPool`) that maintains hot background processes and config generator scripts.
- `concurrency_test.go`: Parallel stress testing, port allocation/stagger race validation, and pool resource bounds.

---

## Key Components & APIs

### 1. Bounded Worker Pool (`Pool`)

Uses a channel-based semaphore to cap concurrent Go routines.
```go
type Pool struct {
    workers   int
    semaphore chan struct{}
    wg        sync.WaitGroup
    // ... error aggregation
}
```
- **`NewPool(workers int) *Pool`**: Instantiates the worker pool. Defaults to `min(CPU * 2, 100)` workers.
- **`Submit(ctx, WorkerFunc)`**: Enqueues a task for parallel execution. Blocks if all slots are occupied.
- **`Wait()`**: Blocks until the WaitGroup drains.
- **`Err() error`**: Returns the first worker error (synchronized via `sync.Once`).

### 2. Port Manager (`PortManager` & `StaggeredAllocator`)

Manages unique localhost ports for launching local Xray SOCKS5 inbound listeners.
- **`Allocate() (int, error)`**: Increments round-robin through its assigned port pool. Runs a quick `net.Listen` check on loopback (`127.0.0.1:port`) to confirm availability.
- **`Release(port)`**: Deletes the port registration from the allocation map, making it available for reuse.
- **`StaggeredAllocator`**: Extends PortManager with a short delay (default `10ms`) on allocations, preventing high-concurrency races when spinning up multiple listeners at once.

### 3. Xray Instance Pool (`XrayPool`)

Allows warm Xray processes to be cached and reused rather than launching a heavy sub-process from scratch for every config.
- **`Acquire(ctx, port, startFn)`**: Checks for dead or expired processes, and returns an idle warm `XrayInstance`. Otherwise, runs `startFn` to spin up a new process.
- **`Release(instance)`**: Returns the instance to the pool if still alive; otherwise terminates it cleanly.
- **`Shutdown()`**: Forces termination on all idle and active Xray processes and removes temporary configuration JSON files on disk.

---

## Conventions & Gotchas

- **Worker Context Interruption:** When enqueuing, workers should check `ctx.Err() != nil` *before* doing expensive tasks, in case the parent context was canceled (e.g. via Ctrl+C).
- **Graceful Port Release:** Always ensure `defer pm.Release(port)` is used after a port is successfully allocated to prevent leaks.
- **System Temp Files:** `WriteXrayConfig` creates temporary JSON configuration files on disk. The `XrayInstance.Stop()` method automatically removes these, but calling `XrayPool.Shutdown()` is recommended at program exit.
