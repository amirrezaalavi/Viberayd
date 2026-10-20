package tester

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/amirrezaalavi/Viberayd/internal/models"
)

// XrayResult holds the outcome of an Xray proxy test.
type XrayResult struct {
	Success  bool          `json:"success"`
	Latency  time.Duration `json:"latency_ms"`
	Error    string        `json:"error,omitempty"`
	Target   string        `json:"target"`
	HTTPCode int           `json:"http_code,omitempty"`
}

// XrayRunner manages an xray-core process for proxy testing.
type XrayRunner struct {
	BinPath    string
	WorkDir    string
	TargetURLs []string
}

// NewXrayRunner creates a runner with sensible defaults.
func NewXrayRunner(binPath string) *XrayRunner {
	if binPath == "" {
		binPath = "xray"
	}
	return &XrayRunner{
		BinPath:    binPath,
		TargetURLs: []string{"https://www.google.com/generate_204", "https://1.1.1.1/cdn-cgi/trace"},
	}
}

// TestXrayProxy starts an Xray instance with the given config, routes a test
// HTTP request through its SOCKS inbound, and measures latency. Success
// requires a 2xx status: block pages (403) and errors prove nothing about
// usable egress.
func (xr *XrayRunner) TestXrayProxy(ctx context.Context, cfg models.ProxyConfig, socksPort int, timeout time.Duration) XrayResult {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	var (
		target  string
		code    int
		elapsed time.Duration
		lastErr error
	)

	err := xr.withTempXray(ctx, cfg, socksPort, timeout, func(proxyAddr string) error {
		// Try every target while xray is still alive. The old code killed the
		// process first and then retried the fallback target against the dead
		// port, so a config whose primary target failed could never pass on
		// the fallback.
		for _, tgt := range xr.TargetURLs {
			target = tgt
			start := time.Now()
			c, err := testViaSOCKS5(proxyAddr, tgt, timeout)
			elapsed = time.Since(start)
			if err == nil {
				if c >= 200 && c < 300 {
					code = c
					return nil
				}
				lastErr = fmt.Errorf("bad status %d", c)
				continue
			}
			lastErr = err
		}
		return lastErr
	})
	if err != nil {
		return XrayResult{
			Success:  false,
			Latency:  elapsed,
			Error:    fmt.Sprintf("proxy test: %v", err),
			Target:   target,
			HTTPCode: code,
		}
	}

	return XrayResult{
		Success:  true,
		Latency:  elapsed,
		Target:   target,
		HTTPCode: code,
	}
}

// listenerWait bounds how long withTempXray waits for the xray SOCKS inbound
// to start accepting connections.
const listenerWait = 5 * time.Second

// waitForListener polls addr until a TCP connection is accepted, the deadline
// passes, or ctx is cancelled.
func waitForListener(ctx context.Context, addr string, max time.Duration) bool {
	deadline := time.Now().Add(max)
	dialer := &net.Dialer{Timeout: 250 * time.Millisecond}
	for {
		conn, err := dialer.DialContext(ctx, "tcp", addr)
		if err == nil {
			_ = conn.Close()
			return true
		}
		if ctx.Err() != nil || time.Now().After(deadline) {
			return false
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// tailText returns the last n bytes of s, trimmed, for compact error detail.
func tailText(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return "..." + s[len(s)-n:]
}

// withTempXray writes the xray config for cfg, spawns xray with a SOCKS5
// inbound on socksPort, waits for it to bind, runs fn with the SOCKS address
// ("127.0.0.1:port"), then kills xray and removes the config file. The whole
// stay is bounded by timeout.
func (xr *XrayRunner) withTempXray(ctx context.Context, cfg models.ProxyConfig, socksPort int, timeout time.Duration, fn func(proxyAddr string) error) error {
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Generate Xray JSON config
	xrayCfg, err := buildXrayConfig(cfg, socksPort)
	if err != nil {
		return fmt.Errorf("build config: %w", err)
	}

	cfgPath := filepath.Join(xr.WorkDir, fmt.Sprintf("xray-%d.json", socksPort))
	if xr.WorkDir == "" {
		cfgPath = fmt.Sprintf("xray-%d.json", socksPort)
	}
	if err := writeJSONFile(cfgPath, xrayCfg); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	defer os.Remove(cfgPath)

	// Start xray process. Capture stderr so a listener that never comes up can
	// be explained instead of surfacing as a bare "connection refused".
	cmd := exec.CommandContext(ctx, xr.BinPath, "-c", cfgPath)
	var stderr bytes.Buffer
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start xray: %w", err)
	}
	// Ensure cleanup
	go func() {
		<-ctx.Done()
		_ = cmd.Process.Kill()
	}()
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	// Wait for the listener instead of sleeping a fixed 300ms: under parallel
	// load xray can need longer to bind, and a fixed sleep makes the first
	// request fail with "connection refused" — a false negative for a config
	// that is perfectly usable.
	proxyAddr := "127.0.0.1:" + strconv.Itoa(socksPort)
	if !waitForListener(ctx, proxyAddr, listenerWait) {
		detail := tailText(stderr.String(), 300)
		if detail == "" {
			detail = "no stderr output"
		}
		return fmt.Errorf("xray never listened on %s: %s", proxyAddr, detail)
	}

	return fn(proxyAddr)
}

// testViaSOCKS5 performs a real HTTP GET through a SOCKS5 proxy.
func testViaSOCKS5(proxyAddr, targetURL string, timeout time.Duration) (int, error) {
	u := targetURL
	if !strings.HasPrefix(u, "http") {
		u = "http://" + u
	}

	proxyURL, err := url.Parse("socks5://" + proxyAddr)
	if err != nil {
		return 0, fmt.Errorf("invalid proxy address: %w", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: timeout,
	}

	resp, err := client.Get(u)
	if err != nil {
		return 0, fmt.Errorf("socks GET request failed: %w", err)
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

func netDialWithTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout(network, address, timeout)
}

func writeJSONFile(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
