package parser

import (
	"testing"

	"github.com/amirrezaalavi/Viberayd/internal/models"
)

// --- ValidatePort (string) ---
// This is NOT tested in parser_test.go — new coverage for ValidatePort.

func TestValidatePort(t *testing.T) {
	tests := []struct {
		name    string
		port    string
		wantErr bool
	}{
		{"valid low", "1", false},
		{"valid mid", "8080", false},
		{"valid high", "65535", false},
		{"zero", "0", true},
		{"negative", "-1", true},
		{"too high", "65536", true},
		{"non-numeric", "abc", true},
		{"empty", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePort(tt.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePort(%q) error = %v, wantErr %v", tt.port, err, tt.wantErr)
			}
		})
	}
}

// --- ValidateFingerprint ---

func TestValidateFingerprint(t *testing.T) {
	tests := []struct {
		name    string
		fp      string
		wantErr bool
	}{
		{"empty", "", false},
		{"chrome", "chrome", false},
		{"firefox", "firefox", false},
		{"safari", "safari", false},
		{"ios", "ios", false},
		{"android", "android", false},
		{"edge", "edge", false},
		{"360", "360", false},
		{"qq", "qq", false},
		{"random", "random", false},
		{"randomized", "randomized", false},
		{"invalid", "not-a-fingerprint", true},
		{"mixed case", "Chrome", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFingerprint(tt.fp)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFingerprint(%q) error = %v, wantErr %v", tt.fp, err, tt.wantErr)
			}
		})
	}
}

// --- ValidateNetwork ---

func TestValidateNetwork(t *testing.T) {
	tests := []struct {
		name    string
		net     string
		wantErr bool
	}{
		{"tcp", "tcp", false},
		{"kcp", "kcp", false},
		{"ws", "ws", false},
		{"grpc", "grpc", false},
		{"http", "http", false},
		{"h2", "h2", false},
		{"quic", "quic", false},
		{"xhttp", "xhttp", false},
		{"empty", "", true},
		{"unknown", "foo", true},
		{"udp", "udp", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNetwork(tt.net)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateNetwork(%q) error = %v, wantErr %v", tt.net, err, tt.wantErr)
			}
		})
	}
}

// --- ValidateSecurity ---

func TestValidateSecurity(t *testing.T) {
	tests := []struct {
		name     string
		security string
		wantErr  bool
	}{
		{"auto", "auto", false},
		{"none", "none", false},
		{"zero", "zero", false},
		{"aes-128-gcm", "aes-128-gcm", false},
		{"chacha20-poly1305", "chacha20-poly1305", false},
		{"empty", "", true},
		{"unknown", "aes-256-gcm", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSecurity(tt.security)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSecurity(%q) error = %v, wantErr %v", tt.security, err, tt.wantErr)
			}
		})
	}
}

// --- ValidatePlugin ---

func TestValidatePlugin(t *testing.T) {
	tests := []struct {
		name    string
		plugin  string
		wantErr bool
	}{
		{"empty", "", false},
		{"obfs-local", "obfs-local", false},
		{"v2ray-plugin", "v2ray-plugin", false},
		{"simple-obfs", "simple-obfs", false},
		{"unknown", "bad-plugin", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePlugin(tt.plugin)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePlugin(%q) error = %v, wantErr %v", tt.plugin, err, tt.wantErr)
			}
		})
	}
}

// --- ValidateHost ---

func TestValidateHost(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		wantErr bool
	}{
		{"valid hostname", "example.com", false},
		{"IPv4", "1.2.3.4", false},
		{"IPv6", "::1", false},
		{"empty", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHost(tt.host)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHost(%q) error = %v, wantErr %v", tt.host, err, tt.wantErr)
			}
		})
	}
}

// --- ValidateIPOrHost ---

func TestValidateIPOrHost(t *testing.T) {
	tests := []struct {
		name      string
		host      string
		wantErr   bool
		skipShort bool // needs DNS lookup, skip in -short mode
	}{
		{"IPv4", "1.2.3.4", false, false},
		{"IPv6", "::1", false, false},
		{"hostname", "example.com", false, true}, // might do DNS lookup
		{"unresolvable", "this-domain-should-not-exist-abc123.test", false, true},
		{"empty", "", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipShort && testing.Short() {
				t.Skip("skipping network-dependent test in short mode")
			}
			err := ValidateIPOrHost(tt.host)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIPOrHost(%q) error = %v, wantErr %v", tt.host, err, tt.wantErr)
			}
		})
	}

	// Also verify that unresolvable host does NOT cause a hard error
	t.Run("unresolvable_does_not_fail", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping network-dependent test in short mode")
		}
		err := ValidateIPOrHost("nonexistent-domain-123456.xyz")
		if err != nil {
			t.Errorf("unresolvable host should not fail at parse time, got %v", err)
		}
	})
}

// --- ValidateBaseConfig ---

func TestValidateBaseConfig(t *testing.T) {
	tests := []struct {
		name          string
		cfg           models.BaseConfig
		wantValid     bool
		wantErrCount  int
		wantWarnCount int
	}{
		{
			name: "valid config",
			cfg: models.BaseConfig{
				Server:   "example.com",
				Port:     443,
				Protocol: models.ProtocolVLess,
				Network:  "tcp",
			},
			wantValid:     true,
			wantErrCount:  0,
			wantWarnCount: 0,
		},
		{
			name: "minimal valid - no protocol, no network",
			cfg: models.BaseConfig{
				Server: "example.com",
				Port:   443,
			},
			wantValid:     true,
			wantErrCount:  0,
			wantWarnCount: 0,
		},
		{
			name: "missing server",
			cfg: models.BaseConfig{
				Server:   "",
				Port:     443,
				Protocol: models.ProtocolSS,
			},
			wantValid:     false,
			wantErrCount:  1,
			wantWarnCount: 0,
		},
		{
			name: "bad port - zero",
			cfg: models.BaseConfig{
				Server:   "example.com",
				Port:     0,
				Protocol: models.ProtocolSS,
			},
			wantValid:     false,
			wantErrCount:  1,
			wantWarnCount: 0,
		},
		{
			name: "bad port - negative",
			cfg: models.BaseConfig{
				Server:   "example.com",
				Port:     -1,
				Protocol: models.ProtocolSS,
			},
			wantValid:     false,
			wantErrCount:  1,
			wantWarnCount: 0,
		},
		{
			name: "unknown protocol",
			cfg: models.BaseConfig{
				Server:   "example.com",
				Port:     443,
				Protocol: models.Protocol("unknown"),
			},
			wantValid:     false,
			wantErrCount:  1,
			wantWarnCount: 0,
		},
		{
			name: "unusual network",
			cfg: models.BaseConfig{
				Server:   "example.com",
				Port:     443,
				Protocol: models.ProtocolVLess,
				Network:  "foo",
			},
			wantValid:     true, // warning only, not an error
			wantErrCount:  0,
			wantWarnCount: 1,
		},
		{
			name: "multiple issues",
			cfg: models.BaseConfig{
				Server:   "",
				Port:     0,
				Protocol: models.Protocol("lol"),
				Network:  "bar",
			},
			wantValid:     false,
			wantErrCount:  3, // server, port, protocol
			wantWarnCount: 1, // network
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vr := ValidateBaseConfig(tt.cfg)
			if vr.Valid != tt.wantValid {
				t.Errorf("ValidateBaseConfig().Valid = %v, want %v", vr.Valid, tt.wantValid)
			}
			if len(vr.Errors) != tt.wantErrCount {
				t.Errorf("ValidateBaseConfig() error count = %d, want %d; errors=%v", len(vr.Errors), tt.wantErrCount, vr.Errors)
			}
			if len(vr.Warnings) != tt.wantWarnCount {
				t.Errorf("ValidateBaseConfig() warning count = %d, want %d; warnings=%v", len(vr.Warnings), tt.wantWarnCount, vr.Warnings)
			}
		})
	}
}
