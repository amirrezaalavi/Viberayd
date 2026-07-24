package tester

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/amirrezaalavi/Viberay/internal/models"
)

func TestTestTCP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}
	ctx := context.Background()
	// Connect to a known public address that accepts TCP
	res := TestTCP(ctx, "1.1.1.1:53", 3*time.Second)
	if !res.Success {
		t.Logf("TCP to 1.1.1.1:53 failed (may be firewalled): %s", res.Error)
	}
}

func TestTestTCP_Timeout(t *testing.T) {
	ctx := context.Background()
	// RFC 5737 TEST-NET-1 192.0.2.0/24 — no host should respond on port 9999
	res := TestTCP(ctx, "192.0.2.1:9999", 500*time.Millisecond)
	if res.Success {
		t.Error("expected timeout/failure for unreachable address")
	}
	if res.Error == "" {
		t.Error("expected an error message")
	}
}

func TestPipeline_DepthQuick(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}
	p := NewPipeline(models.DepthQuick)
	cfg := models.ProxyConfig{
		SS: &models.SSConfig{
			BaseConfig: models.BaseConfig{Server: "1.1.1.1", Port: 53, Protocol: models.ProtocolSS},
			Method:     "aes-256-gcm",
			Password:   "test",
		},
	}
	res := p.Run(context.Background(), cfg, 10820)
	if res.Stage != models.StageCompleted && res.Stage != models.StageTCP {
		t.Errorf("expected quick to stop after TCP, got stage %s", res.Stage)
	}
}

func TestPipeline_DepthStandard_NonTLS(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}
	p := NewPipeline(models.DepthStandard)
	cfg := models.ProxyConfig{
		SS: &models.SSConfig{
			BaseConfig: models.BaseConfig{Server: "1.1.1.1", Port: 53, Protocol: models.ProtocolSS},
			Method:     "aes-256-gcm",
			Password:   "test",
		},
	}
	res := p.Run(context.Background(), cfg, 10820)
	// SS has no TLS, so standard should complete after TCP (TLS skipped)
	if res.Status == models.StatusError {
		t.Logf("Result: %+v", res)
	}
}

func TestPipeline_DepthFull_TLS(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}
	p := NewPipeline(models.DepthFull)
	cfg := models.ProxyConfig{
		Trojan: &models.TrojanConfig{
			BaseConfig: models.BaseConfig{Server: "1.1.1.1", Port: 443, Protocol: models.ProtocolTrojan},
			TLSConfig:  models.TLSConfig{Enabled: true, SNI: "cloudflare-dns.com"},
			Password:   "test",
		},
	}
	res := p.Run(context.Background(), cfg, 10820)
	// We don't have a real trojan server; protocol test will likely fail, but TLS may succeed
	if len(res.Errors) > 0 {
		t.Logf("Errors (expected for fake server): %v", res.Errors)
	}
}

func TestConfigPriority(t *testing.T) {
	tests := []struct {
		cfg  models.ProxyConfig
		want int
	}{
		{models.ProxyConfig{Reality: &models.RealityConfig{}}, 5},
		{models.ProxyConfig{VMess: &models.VMessConfig{}}, 4},
		{models.ProxyConfig{VLess: &models.VLessConfig{}}, 3},
		{models.ProxyConfig{Trojan: &models.TrojanConfig{}}, 2},
		{models.ProxyConfig{SS: &models.SSConfig{}}, 1},
		{models.ProxyConfig{}, 0},
	}
	for _, tt := range tests {
		if got := ConfigPriority(tt.cfg); got != tt.want {
			t.Errorf("priority for %v = %d, want %d", tt.cfg.Protocol(), got, tt.want)
		}
	}
}

func TestBuildXrayConfig_ValidJSON(t *testing.T) {
	cfg := models.ProxyConfig{
		VMess: &models.VMessConfig{
			BaseConfig: models.BaseConfig{Server: "example.com", Port: 443, Protocol: models.ProtocolVMess, Network: "ws"},
			TLSConfig:  models.TLSConfig{Enabled: true, SNI: "example.com", Fingerprint: "chrome"},
			UUID:       "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			AlterID:    0,
			Security:   "auto",
		},
	}
	m, err := buildXrayConfig(cfg, 10820)
	if err != nil {
		t.Fatalf("buildXrayConfig: %v", err)
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// Ensure it's valid JSON
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal round-trip: %v", err)
	}
	// Check structure (after JSON round-trip, slices are []interface{})
	inboundsRaw, ok := out["inbounds"].([]interface{})
	if !ok || len(inboundsRaw) == 0 {
		t.Fatal("missing inbounds")
	}
	inbound, ok := inboundsRaw[0].(map[string]interface{})
	if !ok {
		t.Fatal("inbound is not an object")
	}
	if int(inbound["port"].(float64)) != 10820 {
		t.Errorf("expected port 10820, got %v", inbound["port"])
	}
}

func TestBuildOutbound_Unsupported(t *testing.T) {
	_, err := buildOutbound(models.ProxyConfig{})
	if err == nil {
		t.Error("expected error for unsupported protocol")
	}
}

func TestBuildRealityOutbound(t *testing.T) {
	cfg := models.ProxyConfig{
		Reality: &models.RealityConfig{
			BaseConfig: models.BaseConfig{Server: "reality.example.com", Port: 443, Protocol: models.ProtocolReality},
			UUID:       "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			PublicKey:  "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
			ShortID:    "00112233",
			SpiderX:    "/download",
			Flow:       "xtls-rprx-vision",
			TLSConfig:  models.TLSConfig{SNI: "www.example.com", Fingerprint: "chrome"},
		},
	}
	out, err := buildOutbound(cfg)
	if err != nil {
		t.Fatalf("buildOutbound: %v", err)
	}
	stream, ok := out["streamSettings"].(map[string]any)
	if !ok {
		t.Fatal("missing streamSettings")
	}
	if stream["security"] != "reality" {
		t.Errorf("expected reality security, got %v", stream["security"])
	}
}
