package tester

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/amiralavi/viberay/internal/models"
)

// --- Test helpers ---

// startTCPEcho starts a goroutine that accepts one TCP connection, reads
// a few bytes, writes them back, then closes. The listener is cleaned up
// via t.Cleanup.
func startTCPEcho(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen tcp echo: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return // listener closed
			}
			go func(c net.Conn) {
				defer c.Close()
				_ = c.SetDeadline(time.Now().Add(2 * time.Second))
				buf := make([]byte, 128)
				n, _ := c.Read(buf)
				if n > 0 {
					_, _ = c.Write(buf[:n])
				}
			}(conn)
		}
	}()
	return ln
}

// startTLSListener starts a goroutine that accepts TLS connections and
// echoes back data. Uses self-signed cert. Listener cleaned up via
// t.Cleanup.
func startTLSListener(t *testing.T) net.Listener {
	t.Helper()
	cert := generateSelfSignedCert(t)
	config := &tls.Config{Certificates: []tls.Certificate{cert}}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", config)
	if err != nil {
		t.Fatalf("tls listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				_ = c.SetDeadline(time.Now().Add(2 * time.Second))
				buf := make([]byte, 128)
				n, _ := c.Read(buf)
				if n > 0 {
					_, _ = c.Write(buf[:n])
				}
			}(conn)
		}
	}()
	return ln
}

// generateSelfSignedCert creates a self-signed certificate for 127.0.0.1.
func generateSelfSignedCert(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	return tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
	}
}

// proxyConfigForAddr creates a minimal VLess ProxyConfig pointing to
// server:port with TLS optionally enabled.
func proxyConfigForAddr(server string, port int, tlsEnabled bool) models.ProxyConfig {
	return models.ProxyConfig{
		VLess: &models.VLessConfig{
			BaseConfig: models.BaseConfig{
				Server:   server,
				Port:     port,
				Protocol: models.ProtocolVLess,
			},
			TLSConfig: models.TLSConfig{
				Enabled:     tlsEnabled,
				SkipVerify:  true,
				SNI:         server,
				Fingerprint: "chrome",
			},
			UUID:       "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			Encryption: "none",
		},
	}
}

// --- Integration tests ---

// TestPipeline_TCPOpen verifies StatusSuccess when the host is reachable (TCP open).
func TestPipeline_TCPOpen(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ln := startTCPEcho(t)
	addr := ln.Addr().(*net.TCPAddr)

	p := NewPipeline(models.DepthQuick)
	p.TCPTimeout = 2 * time.Second
	cfg := proxyConfigForAddr("127.0.0.1", addr.Port, false)
	res := p.Run(context.Background(), cfg, 0)

	if res.Status != models.StatusSuccess {
		t.Fatalf("expected StatusSuccess for open TCP, got %q: errors=%v", res.Status, res.Errors)
	}
}

// TestPipeline_TCPClosed verifies StatusFailed when the port is closed.
func TestPipeline_TCPClosed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	p := NewPipeline(models.DepthQuick)
	p.TCPTimeout = 500 * time.Millisecond
	cfg := proxyConfigForAddr("127.0.0.1", 1, false)
	res := p.Run(context.Background(), cfg, 0)

	if res.Status != models.StatusFailed {
		t.Fatalf("expected StatusFailed for closed port, got %q", res.Status)
	}
}

// TestPipeline_TLSHandshakeFail verifies StatusFailed when TLS handshake
// fails (e.g. connecting to a plain TCP server with TLS enabled).
func TestPipeline_TLSHandshakeFail(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	// Start a plain TCP echo server (no TLS)
	ln := startTCPEcho(t)
	addr := ln.Addr().(*net.TCPAddr)

	p := NewPipeline(models.DepthStandard)
	p.TCPTimeout = 2 * time.Second
	p.TLSTimeout = 2 * time.Second

	// Config with TLS enabled pointing to the plain TCP server
	cfg := proxyConfigForAddr("127.0.0.1", addr.Port, true)
	res := p.Run(context.Background(), cfg, 0)

	if res.Status != models.StatusFailed {
		t.Fatalf("expected StatusFailed when TLS handshake fails, got %q: errors=%v", res.Status, res.Errors)
	}
	if len(res.Errors) == 0 {
		t.Fatal("expected error about TLS handshake failure")
	}
}

// TestIntegrationPipeline_DepthQuick verifies that depth=quick only runs TCP and
// does NOT attempt TLS or protocol stages.
func TestIntegrationPipeline_DepthQuick(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	// Use an actual TLS server to verify depth=quick skips TLS
	ln := startTLSListener(t)
	addr := ln.Addr().(*net.TCPAddr)

	p := NewPipeline(models.DepthQuick)
	p.TCPTimeout = 2 * time.Second
	cfg := proxyConfigForAddr("127.0.0.1", addr.Port, true)
	res := p.Run(context.Background(), cfg, 0)

	if res.Status != models.StatusSuccess {
		t.Fatalf("expected StatusSuccess for quick depth, got %q: errors=%v", res.Status, res.Errors)
	}
	if res.Stage != models.StageCompleted {
		t.Fatalf("expected StageCompleted after quick depth, got %q", res.Stage)
	}
	// TLS latency should be zero because TLS stage was never attempted
	if res.Latencies.TLS != 0 {
		t.Error("expected zero TLS latency for quick depth")
	}
}

// TestPipeline_Cancellation verifies that Pipeline.Run respects
// ctx.Done() cancellation.
func TestPipeline_Cancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	// Start a TCP server that accepts but hangs before echoing
	var wg sync.WaitGroup
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	wg.Add(1)
	go func() {
		defer wg.Done()
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		// Accept connection but never respond — hold it open
		_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
		buf := make([]byte, 128)
		_, _ = conn.Read(buf) // wait for data
		// Don't write back — hangs the TLS/protocol stage
		conn.Close()
	}()
	t.Cleanup(func() { wg.Wait() })

	addr := ln.Addr().(*net.TCPAddr)
	pipe := NewPipeline(models.DepthFull)
	pipe.TCPTimeout = 5 * time.Second
	pipe.TLSTimeout = 5 * time.Second

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel almost immediately so the pipeline sees a done context
	time.AfterFunc(10*time.Millisecond, cancel)

	cfg := proxyConfigForAddr("127.0.0.1", addr.Port, true)
	res := pipe.Run(ctx, cfg, 0)

	// The pipeline should respond to cancellation
	if res.Status != models.StatusFailed && res.Status != models.StatusError {
		t.Fatalf("expected StatusFailed or StatusError after cancellation, got %q", res.Status)
	}
}

// TestXrayBinaryNotRequired verifies integration tests skip gracefully
// when xray binary is not installed.
func TestXrayBinaryNotRequired(t *testing.T) {
	_, err := exec.LookPath("xray")
	if err != nil {
		t.Skip("xray binary not found — tests that require xray will be skipped")
	}
}
