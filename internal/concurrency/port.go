package concurrency

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// PortManager allocates ports from a contiguous range with availability
// checks, auto-release, and conflict resolution.
type PortManager struct {
	base  int
	limit int

	mu       sync.Mutex
	allocated map[int]bool
	next     int
}

// NewPortManager creates a manager for ports [base, base+limit).
func NewPortManager(base, limit int) *PortManager {
	return &PortManager{
		base:      base,
		limit:     limit,
		allocated: make(map[int]bool),
		next:      base,
	}
}

// Allocate returns the next available port in the range.
// It checks local binding and skips occupied ports.
func (pm *PortManager) Allocate() (int, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	start := pm.next
	for {
		port := pm.next
		pm.next++
		if pm.next >= pm.base+pm.limit {
			pm.next = pm.base
		}

		if pm.allocated[port] {
			if pm.next == start {
				return 0, fmt.Errorf("port range %d-%d exhausted", pm.base, pm.base+pm.limit-1)
			}
			continue
		}

		if !isPortAvailable(port) {
			if pm.next == start {
				return 0, fmt.Errorf("port range %d-%d exhausted", pm.base, pm.base+pm.limit-1)
			}
			continue
		}

		pm.allocated[port] = true
		return port, nil
	}
}

// Release marks a port as free.
func (pm *PortManager) Release(port int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	delete(pm.allocated, port)
}

// Reserve marks a port as allocated without checking availability.
func (pm *PortManager) Reserve(port int) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if port < pm.base || port >= pm.base+pm.limit {
		return fmt.Errorf("port %d out of range [%d,%d)", port, pm.base, pm.base+pm.limit)
	}
	if pm.allocated[port] {
		return fmt.Errorf("port %d already allocated", port)
	}
	pm.allocated[port] = true
	return nil
}

// isPortAvailable tries to listen on the port with a short timeout.
func isPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}

// StaggeredAllocator wraps PortManager with a configurable stagger delay
// to reduce collisions when many workers start simultaneously.
type StaggeredAllocator struct {
	*PortManager
	Delay time.Duration
}

// NewStaggeredAllocator creates an allocator with a default 10ms stagger.
func NewStaggeredAllocator(base, limit int) *StaggeredAllocator {
	return &StaggeredAllocator{
		PortManager: NewPortManager(base, limit),
		Delay:       10 * time.Millisecond,
	}
}

// AllocateStaggered returns a port after an optional stagger delay.
func (sa *StaggeredAllocator) AllocateStaggered() (int, error) {
	if sa.Delay > 0 {
		time.Sleep(sa.Delay)
	}
	return sa.Allocate()
}
