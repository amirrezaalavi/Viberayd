package tester

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/amirrezaalavi/Viberay/internal/models"
)

// TLSResult holds the outcome of a TLS handshake test.
type TLSResult struct {
	Success      bool          `json:"success"`
	Latency      time.Duration `json:"latency_ms"`
	Version      string        `json:"version,omitempty"`
	CipherSuite  string        `json:"cipher_suite,omitempty"`
	Verified     bool          `json:"verified"`
	Error        string        `json:"error,omitempty"`
}

// fingerprintToCipherSuites maps common fingerprint names to cipher-suite preferences.
// In a full implementation this would drive uTLS / parrots; here we configure Go TLS
// to approximate the intent where possible.
var fingerprintToCipherSuites = map[string][]uint16{
	"chrome": {
		tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
	},
	"firefox": {
		tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_CHACHA20_POLY1305_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	},
}

// TestTLS performs a TLS handshake against the given address using the config's SNI.
func TestTLS(ctx context.Context, addr, sni string, skipVerify bool, fingerprint string, timeout time.Duration) TLSResult {
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	dialer := net.Dialer{Timeout: timeout}

	config := &tls.Config{
		ServerName:         sni,
		InsecureSkipVerify: skipVerify,
	}

	// Apply fingerprint approximation
	fp := strings.ToLower(fingerprint)
	if suites, ok := fingerprintToCipherSuites[fp]; ok {
		config.CipherSuites = suites
		config.PreferServerCipherSuites = false
	}

	start := time.Now()
	conn, err := tls.DialWithDialer(&dialer, "tcp", addr, config)
	elapsed := time.Since(start)

	if err != nil {
		return TLSResult{
			Success: false,
			Latency: elapsed,
			Error:   fmt.Sprintf("tls handshake: %v", err),
		}
	}
	defer conn.Close()

	state := conn.ConnectionState()
	ver := tlsVersionName(state.Version)
	cs := tls.CipherSuiteName(state.CipherSuite)

	return TLSResult{
		Success:     true,
		Latency:     elapsed,
		Version:     ver,
		CipherSuite: cs,
		Verified:    state.VerifiedChains != nil && len(state.VerifiedChains) > 0,
	}
}

// TestTLSForConfig is a convenience wrapper.
func TestTLSForConfig(ctx context.Context, cfg models.ProxyConfig, timeout time.Duration) TLSResult {
	addr := cfg.Addr()
	if addr == "" {
		return TLSResult{Success: false, Error: "no address in config"}
	}

	var sni string
	var skipVerify bool
	var fp string
	var enabled bool

	switch {
	case cfg.VMess != nil:
		sni = cfg.VMess.SNI
		skipVerify = cfg.VMess.SkipVerify
		fp = cfg.VMess.Fingerprint
		enabled = cfg.VMess.Enabled
	case cfg.VLess != nil:
		sni = cfg.VLess.SNI
		skipVerify = cfg.VLess.SkipVerify
		fp = cfg.VLess.Fingerprint
		enabled = cfg.VLess.Enabled
	case cfg.Trojan != nil:
		sni = cfg.Trojan.SNI
		skipVerify = cfg.Trojan.SkipVerify
		fp = cfg.Trojan.Fingerprint
		enabled = cfg.Trojan.Enabled
	case cfg.Reality != nil:
		sni = cfg.Reality.SNI
		skipVerify = cfg.Reality.SkipVerify
		fp = cfg.Reality.Fingerprint
		enabled = true // Reality is always TLS-like
	default:
		return TLSResult{Success: false, Error: "no TLS config in proxy"}
	}

	if !enabled {
		return TLSResult{Success: false, Error: "TLS not enabled for this config"}
	}

	return TestTLS(ctx, addr, sni, skipVerify, fp, timeout)
}

func tlsVersionName(v uint16) string {
	switch v {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("unknown (0x%04x)", v)
	}
}
