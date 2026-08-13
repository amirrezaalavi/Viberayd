package tester

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/amirrezaalavi/Viberayd/internal/models"
)

// TCPResult holds the outcome of a TCP connectivity test.
type TCPResult struct {
	Success bool          `json:"success"`
	Latency time.Duration `json:"latency_ms"`
	Error   string        `json:"error,omitempty"`
}

// TestTCP performs a raw TCP connection to the config's server:port.
func TestTCP(ctx context.Context, addr string, timeout time.Duration) TCPResult {
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	start := time.Now()
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	elapsed := time.Since(start)

	if err != nil {
		return TCPResult{
			Success: false,
			Latency: elapsed,
			Error:   fmt.Sprintf("tcp connect: %v", err),
		}
	}
	_ = conn.Close()

	return TCPResult{
		Success: true,
		Latency: elapsed,
	}
}

// TestTCPForConfig is a convenience wrapper that extracts the address.
func TestTCPForConfig(ctx context.Context, cfg models.ProxyConfig, timeout time.Duration) TCPResult {
	addr := cfg.Addr()
	if addr == "" {
		return TCPResult{Success: false, Error: "no address in config"}
	}
	return TestTCP(ctx, addr, timeout)
}
