package tester

import (
	"context"
	"encoding/json"
	"fmt"
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

	var target string
	var code int
	var elapsed time.Duration
	var testErr error
	err := xr.withTempXray(ctx, cfg, socksPort, timeout, func(proxyAddr string) error {
		target = xr.TargetURLs[0]

		start := time.Now()
		code, testErr = testViaSOCKS5(proxyAddr, target, timeout)
		elapsed = time.Since(start)

		if testErr != nil {
			// Try fallback target
			if len(xr.TargetURLs) > 1 {
				target = xr.TargetURLs[1]
				start = time.Now()
				code, testErr = testViaSOCKS5(proxyAddr, target, timeout)
				elapsed = time.Since(start)
			}
		}
		return testErr
	})
	if err != nil {
		return XrayResult{
			Success: false,
			Latency: elapsed,
			Error:   fmt.Sprintf("proxy test: %v", err),
			Target:  target,
		}
	}

	if code < 200 || code > 299 {
		return XrayResult{
			Success:  false,
			Latency:  elapsed,
			Error:    fmt.Sprintf("proxy test: bad status %d", code),
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

	// Start xray process
	cmd := exec.CommandContext(ctx, xr.BinPath, "-c", cfgPath)
	cmd.Stdout = nil
	cmd.Stderr = nil

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

	// Wait briefly for xray to bind
	time.Sleep(300 * time.Millisecond)

	return fn("127.0.0.1:" + strconv.Itoa(socksPort))
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
