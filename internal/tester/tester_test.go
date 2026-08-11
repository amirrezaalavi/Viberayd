package tester

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/amirrezaalavi/Viberayd/internal/models"
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
		{models.ProxyConfig{Hysteria2: &models.Hysteria2Config{}}, 6},
		{models.ProxyConfig{TUIC: &models.TUICConfig{}}, 5},
		{models.ProxyConfig{WireGuard: &models.WireGuardConfig{}}, 3},
		{models.ProxyConfig{Socks5: &models.Socks5Config{}}, 1},
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

func TestBuildWireGuardOutbound(t *testing.T) {
		cfg := models.ProxyConfig{
			WireGuard: &models.WireGuardConfig{
				BaseConfig:   models.BaseConfig{Server: "wg.example.com", Port: 51820, Protocol: models.ProtocolWireGuard},
				PrivateKey:   "uFXxaZmMhYmBzA7jRdqjuPnlmgZcLExPUkZeiRGySH0=",
				PublicKey:    "HI1yN4GzQGQtc1cN1EJa0fPbYZaNoz+dYqNHxIk/WGQ=",
				LocalAddress: "10.0.0.2/32",
				MTU:          1420,
				Reserved:     "0,0,0",
				AllowedIPs:   "0.0.0.0/0,::/0",
			},
		}
		out, err := buildOutbound(cfg)
		if err != nil {
			t.Fatalf("buildOutbound wireguard: %v", err)
		}
		if out["protocol"] != "wireguard" {
			t.Errorf("protocol = %v, want wireguard", out["protocol"])
		}
		settings, ok := out["settings"].(map[string]any)
		if !ok {
			t.Fatal("missing settings")
		}
		if settings["secretKey"] != cfg.WireGuard.PrivateKey {
			t.Errorf("secretKey = %v", settings["secretKey"])
		}
		peers, ok := settings["peers"].([]map[string]any)
		if !ok {
			peersRaw, ok2 := settings["peers"].([]interface{})
			if !ok2 {
				t.Fatal("missing peers")
			}
			peers = make([]map[string]any, len(peersRaw))
			for i, p := range peersRaw {
				peers[i] = p.(map[string]any)
			}
		}
		if len(peers) != 1 {
			t.Fatalf("expected 1 peer, got %d", len(peers))
		}
		if peers[0]["publicKey"] != cfg.WireGuard.PublicKey {
			t.Errorf("peer publicKey = %v", peers[0]["publicKey"])
		}
	}

	func TestBuildTUICOutbound(t *testing.T) {
		cfg := models.ProxyConfig{
			TUIC: &models.TUICConfig{
				BaseConfig:        models.BaseConfig{Server: "tuic.example.com", Port: 443, Protocol: models.ProtocolTUIC},
				UUID:              "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
				Password:          "mypass",
				CongestionControl: "bbr",
				UDPRelayMode:      "native",
				TLSConfig:         models.TLSConfig{SNI: "example.com", Enabled: true},
			},
		}
		out, err := buildOutbound(cfg)
		if err != nil {
			t.Fatalf("buildOutbound tuic: %v", err)
		}
		if out["protocol"] != "tuic" {
			t.Errorf("protocol = %v, want tuic", out["protocol"])
		}
		stream, ok := out["streamSettings"].(map[string]any)
		if !ok {
			t.Fatal("missing streamSettings")
		}
		if stream["network"] != "quic" {
			t.Errorf("network = %v, want quic", stream["network"])
		}
		if stream["security"] != "tls" {
			t.Errorf("security = %v, want tls", stream["security"])
		}
	}

	func TestBuildHysteria2Outbound(t *testing.T) {
		cfg := models.ProxyConfig{
			Hysteria2: &models.Hysteria2Config{
				BaseConfig: models.BaseConfig{Server: "hy2.example.com", Port: 443, Protocol: models.ProtocolHysteria2},
				Auth:       "myauth",
				UpMbps:     100,
				DownMbps:   200,
				ObfsType:   "salamander",
				TLSConfig:  models.TLSConfig{SNI: "example.com", Enabled: true},
			},
		}
		out, err := buildOutbound(cfg)
		if err != nil {
			t.Fatalf("buildOutbound hysteria2: %v", err)
		}
		if out["protocol"] != "hysteria2" {
			t.Errorf("protocol = %v, want hysteria2", out["protocol"])
		}
		settings, ok := out["settings"].(map[string]any)
		if !ok {
			t.Fatal("missing settings")
		}
		if settings["auth"] != "myauth" {
			t.Errorf("auth = %v", settings["auth"])
		}
		if settings["up"] != "100 mbps" {
			t.Errorf("up = %v, want 100 mbps", settings["up"])
		}
		if settings["down"] != "200 mbps" {
			t.Errorf("down = %v, want 200 mbps", settings["down"])
		}
	}

	func TestBuildSocks5Outbound(t *testing.T) {
		cfg := models.ProxyConfig{
			Socks5: &models.Socks5Config{
				BaseConfig: models.BaseConfig{Server: "s5.example.com", Port: 1080, Protocol: models.ProtocolSocks5},
				Username:   "myuser",
				Password:   "mypass",
			},
		}
		out, err := buildOutbound(cfg)
		if err != nil {
			t.Fatalf("buildOutbound socks5: %v", err)
		}
		if out["protocol"] != "socks" {
			t.Errorf("protocol = %v, want socks", out["protocol"])
		}
		settings, ok := out["settings"].(map[string]any)
		if !ok {
			t.Fatal("missing settings")
		}
		servers, ok := settings["servers"].([]map[string]any)
		if !ok {
			serversRaw, ok2 := settings["servers"].([]interface{})
			if !ok2 {
				t.Fatal("missing servers")
			}
			servers = make([]map[string]any, len(serversRaw))
			for i, s := range serversRaw {
				servers[i] = s.(map[string]any)
			}
		}
		if len(servers) != 1 {
			t.Fatalf("expected 1 server, got %d", len(servers))
		}
		users, ok := servers[0]["users"].([]map[string]any)
		if !ok {
			usersRaw, ok2 := servers[0]["users"].([]interface{})
			if !ok2 {
				t.Fatal("missing users")
			}
			users = make([]map[string]any, len(usersRaw))
			for i, u := range usersRaw {
				users[i] = u.(map[string]any)
			}
		}
		if users[0]["user"] != "myuser" {
			t.Errorf("user = %v", users[0]["user"])
		}
		if users[0]["pass"] != "mypass" {
			t.Errorf("pass = %v", users[0]["pass"])
		}
	}

	func TestBuildSocks5Outbound_NoAuth(t *testing.T) {
		cfg := models.ProxyConfig{
			Socks5: &models.Socks5Config{
				BaseConfig: models.BaseConfig{Server: "s5.example.com", Port: 1080, Protocol: models.ProtocolSocks5},
			},
		}
		out, err := buildOutbound(cfg)
		if err != nil {
			t.Fatalf("buildOutbound socks5 noauth: %v", err)
		}
		settings, ok := out["settings"].(map[string]any)
		if !ok {
			t.Fatal("missing settings")
		}
		servers, ok := settings["servers"].([]map[string]any)
		if !ok {
			serversRaw, ok2 := settings["servers"].([]interface{})
			if !ok2 {
				t.Fatal("missing servers")
			}
			servers = make([]map[string]any, len(serversRaw))
			for i, s := range serversRaw {
				servers[i] = s.(map[string]any)
			}
		}
		if _, hasUsers := servers[0]["users"]; hasUsers {
			t.Error("expected no users for socks5 without auth")
		}
	}

	func TestBuildXrayConfig_AllProtocols(t *testing.T) {
		protocols := map[models.Protocol]models.ProxyConfig{
			models.ProtocolSS:        {SS: &models.SSConfig{BaseConfig: models.BaseConfig{Server: "ss.example.com", Port: 8388, Protocol: models.ProtocolSS}, Method: "aes-256-gcm", Password: "pass"}},
			models.ProtocolVMess:     {VMess: &models.VMessConfig{BaseConfig: models.BaseConfig{Server: "vmess.example.com", Port: 443, Protocol: models.ProtocolVMess}, UUID: "a1b2c3d4-e5f6-7890-abcd-ef1234567890", Security: "auto"}},
			models.ProtocolVLess:     {VLess: &models.VLessConfig{BaseConfig: models.BaseConfig{Server: "vless.example.com", Port: 443, Protocol: models.ProtocolVLess}, UUID: "a1b2c3d4-e5f6-7890-abcd-ef1234567890"}},
			models.ProtocolTrojan:    {Trojan: &models.TrojanConfig{BaseConfig: models.BaseConfig{Server: "tr.example.com", Port: 443, Protocol: models.ProtocolTrojan}, Password: "pass"}},
			models.ProtocolReality:   {Reality: &models.RealityConfig{BaseConfig: models.BaseConfig{Server: "reality.example.com", Port: 443, Protocol: models.ProtocolReality}, UUID: "a1b2c3d4-e5f6-7890-abcd-ef1234567890", PublicKey: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="}},
			models.ProtocolWireGuard: {WireGuard: &models.WireGuardConfig{BaseConfig: models.BaseConfig{Server: "wg.example.com", Port: 51820, Protocol: models.ProtocolWireGuard}, PrivateKey: "key", PublicKey: "pubkey", LocalAddress: "10.0.0.2/32"}},
			models.ProtocolTUIC:      {TUIC: &models.TUICConfig{BaseConfig: models.BaseConfig{Server: "tuic.example.com", Port: 443, Protocol: models.ProtocolTUIC}, UUID: "a1b2c3d4-e5f6-7890-abcd-ef1234567890", Password: "pass"}},
			models.ProtocolHysteria2: {Hysteria2: &models.Hysteria2Config{BaseConfig: models.BaseConfig{Server: "hy2.example.com", Port: 443, Protocol: models.ProtocolHysteria2}, Auth: "auth"}},
			models.ProtocolSocks5:    {Socks5: &models.Socks5Config{BaseConfig: models.BaseConfig{Server: "s5.example.com", Port: 1080, Protocol: models.ProtocolSocks5}, Username: "user", Password: "pass"}},
		}
		for proto, cfg := range protocols {
			t.Run(string(proto), func(t *testing.T) {
				m, err := buildXrayConfig(cfg, 10820)
				if err != nil {
					t.Fatalf("buildXrayConfig %s: %v", proto, err)
				}
				outbounds, ok := m["outbounds"].([]map[string]any)
				if !ok {
					obsRaw, ok2 := m["outbounds"].([]interface{})
					if !ok2 {
						t.Fatal("missing outbounds")
					}
					outbounds = make([]map[string]any, len(obsRaw))
					for i, ob := range obsRaw {
						outbounds[i] = ob.(map[string]any)
					}
				}
				if len(outbounds) < 1 {
					t.Fatal("no outbounds")
				}
				if outbounds[0]["tag"] != "proxy" {
					t.Errorf("outbound tag = %v, want proxy", outbounds[0]["tag"])
				}
				// Verify that it marshals to valid JSON
				b, err := json.Marshal(m)
				if err != nil {
					t.Fatalf("json.Marshal: %v", err)
				}
				var roundtrip map[string]any
				if err := json.Unmarshal(b, &roundtrip); err != nil {
					t.Fatalf("json.Unmarshal round-trip: %v", err)
				}
			})
		}
	}
