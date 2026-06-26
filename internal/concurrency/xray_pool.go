package concurrency

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"time"
)

// XrayInstance represents a pooled xray-core process.
type XrayInstance struct {
	Port      int
	ConfigPath string
	Cmd       *exec.Cmd
	CreatedAt time.Time
	LastUsed  time.Time
	healthy   bool
}

// IsRunning reports whether the process is still alive.
func (xi *XrayInstance) IsRunning() bool {
	if xi.Cmd == nil || xi.Cmd.Process == nil {
		return false
	}
	return xi.Cmd.Process.Signal(os.Signal(nil)) == nil
}

// Stop kills the process and cleans up.
func (xi *XrayInstance) Stop() {
	if xi.Cmd != nil && xi.Cmd.Process != nil {
		_ = xi.Cmd.Process.Kill()
		_ = xi.Cmd.Wait()
	}
	if xi.ConfigPath != "" {
		_ = os.Remove(xi.ConfigPath)
	}
}

// XrayPool manages reusable xray-core instances.
type XrayPool struct {
	binPath     string
	maxSize     int
	idleTimeout time.Duration

	mu        sync.Mutex
	available []*XrayInstance
	inUse     map[int]*XrayInstance
}

// NewXrayPool creates a pool. maxSize <= 0 defaults to 20.
func NewXrayPool(binPath string, maxSize int) *XrayPool {
	if maxSize <= 0 {
		maxSize = 20
	}
	return &XrayPool{
		binPath:     binPath,
		maxSize:     maxSize,
		idleTimeout: 2 * time.Minute,
		inUse:       make(map[int]*XrayInstance),
	}
}

// Acquire returns an existing idle instance or starts a new one.
// The caller must provide a function that generates a fresh config file.
func (xp *XrayPool) Acquire(ctx context.Context, port int, startFn func(port int) (*XrayInstance, error)) (*XrayInstance, error) {
	xp.mu.Lock()

	// Try to reuse an idle instance
	for i, xi := range xp.available {
		if !xi.IsRunning() {
			continue // dead, skip
		}
		if time.Since(xi.LastUsed) > xp.idleTimeout {
			xi.Stop()
			continue
		}
		// Remove from available
		xp.available = append(xp.available[:i], xp.available[i+1:]...)
		xp.inUse[xi.Port] = xi
		xi.LastUsed = time.Now()
		xp.mu.Unlock()
		return xi, nil
	}

	// Check capacity
	if len(xp.inUse) >= xp.maxSize {
		xp.mu.Unlock()
		return nil, fmt.Errorf("xray pool at capacity %d", xp.maxSize)
	}
	xp.mu.Unlock()

	// Start new instance
	xi, err := startFn(port)
	if err != nil {
		return nil, err
	}
	xi.Port = port
	xi.CreatedAt = time.Now()
	xi.LastUsed = time.Now()
	xi.healthy = true

	xp.mu.Lock()
	xp.inUse[port] = xi
	xp.mu.Unlock()

	return xi, nil
}

// Release returns an instance to the idle pool.
func (xp *XrayPool) Release(xi *XrayInstance) {
	xp.mu.Lock()
	defer xp.mu.Unlock()
	delete(xp.inUse, xi.Port)
	if xi.IsRunning() {
		xp.available = append(xp.available, xi)
	} else {
		xi.Stop()
	}
}

// Shutdown stops all instances.
func (xp *XrayPool) Shutdown() {
	xp.mu.Lock()
	defer xp.mu.Unlock()
	for _, xi := range xp.available {
		xi.Stop()
	}
	for _, xi := range xp.inUse {
		xi.Stop()
	}
	xp.available = nil
	xp.inUse = make(map[int]*XrayInstance)
}

// HealthCheck stops dead instances and logs status.
func (xp *XrayPool) HealthCheck() {
	xp.mu.Lock()
	defer xp.mu.Unlock()

	alive := 0
	for _, xi := range xp.available {
		if xi.IsRunning() {
			alive++
		} else {
			xi.Stop()
		}
	}
	slog.Debug("xray pool health check", "idle_alive", alive, "in_use", len(xp.inUse))
}

// StartXrayInstance is a helper that starts xray with a config file.
func StartXrayInstance(binPath, configPath string, port int) (*XrayInstance, error) {
	cmd := exec.Command(binPath, "-c", configPath)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start xray: %w", err)
	}
	// Wait briefly for bind
	time.Sleep(200 * time.Millisecond)
	return &XrayInstance{
		Port:       port,
		ConfigPath: configPath,
		Cmd:        cmd,
	}, nil
}

// WriteXrayConfig writes the JSON config to a temp file and returns its path.
func WriteXrayConfig(workDir string, cfg map[string]any) (string, error) {
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", err
	}
	f, err := os.CreateTemp(workDir, "xray-*.json")
	if err != nil {
		return "", err
	}
	if _, err := f.Write(b); err != nil {
		_ = f.Close()
		return "", err
	}
	return f.Name(), f.Close()
}
