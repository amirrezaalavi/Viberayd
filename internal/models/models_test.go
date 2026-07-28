package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestProtocol_IsValid(t *testing.T) {
	tests := []struct {
		p       Protocol
		want    bool
	}{
		{ProtocolSS, true},
		{ProtocolVMess, true},
		{ProtocolVLess, true},
		{ProtocolTrojan, true},
		{ProtocolReality, true},
		{ProtocolWireGuard, true},
		{ProtocolTUIC, true},
		{ProtocolHysteria2, true},
		{ProtocolSocks5, true},
		{Protocol("unknown"), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.p), func(t *testing.T) {
			if got := tt.p.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBaseConfig_Addr(t *testing.T) {
	b := BaseConfig{Server: "192.0.2.1", Port: 443}
	if got := b.Addr(); got != "192.0.2.1:443" {
		t.Errorf("Addr() = %s, want 192.0.2.1:443", got)
	}
}

func TestProxyConfig_Protocol(t *testing.T) {
	tests := []struct {
		name string
		pc   ProxyConfig
		want Protocol
	}{
		{"SS", ProxyConfig{SS: &SSConfig{}}, ProtocolSS},
		{"VMess", ProxyConfig{VMess: &VMessConfig{}}, ProtocolVMess},
		{"VLess", ProxyConfig{VLess: &VLessConfig{}}, ProtocolVLess},
		{"Trojan", ProxyConfig{Trojan: &TrojanConfig{}}, ProtocolTrojan},
		{"Reality", ProxyConfig{Reality: &RealityConfig{}}, ProtocolReality},
		{"WireGuard", ProxyConfig{WireGuard: &WireGuardConfig{}}, ProtocolWireGuard},
		{"TUIC", ProxyConfig{TUIC: &TUICConfig{}}, ProtocolTUIC},
		{"Hysteria2", ProxyConfig{Hysteria2: &Hysteria2Config{}}, ProtocolHysteria2},
		{"Socks5", ProxyConfig{Socks5: &Socks5Config{}}, ProtocolSocks5},
		{"empty", ProxyConfig{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pc.Protocol(); got != tt.want {
				t.Errorf("Protocol() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProxyConfig_Name(t *testing.T) {
	tests := []struct {
		name string
		pc   ProxyConfig
		want string
	}{
		{"SS", ProxyConfig{SS: &SSConfig{BaseConfig: BaseConfig{Name: "ss-name"}}}, "ss-name"},
		{"VMess", ProxyConfig{VMess: &VMessConfig{BaseConfig: BaseConfig{Name: "vmess-name"}}}, "vmess-name"},
		{"VLess", ProxyConfig{VLess: &VLessConfig{BaseConfig: BaseConfig{Name: "vless-name"}}}, "vless-name"},
		{"Trojan", ProxyConfig{Trojan: &TrojanConfig{BaseConfig: BaseConfig{Name: "trojan-name"}}}, "trojan-name"},
		{"Reality", ProxyConfig{Reality: &RealityConfig{BaseConfig: BaseConfig{Name: "reality-name"}}}, "reality-name"},
		{"WireGuard", ProxyConfig{WireGuard: &WireGuardConfig{BaseConfig: BaseConfig{Name: "wg-name"}}}, "wg-name"},
		{"TUIC", ProxyConfig{TUIC: &TUICConfig{BaseConfig: BaseConfig{Name: "tuic-name"}}}, "tuic-name"},
		{"Hysteria2", ProxyConfig{Hysteria2: &Hysteria2Config{BaseConfig: BaseConfig{Name: "hy2-name"}}}, "hy2-name"},
		{"Socks5", ProxyConfig{Socks5: &Socks5Config{BaseConfig: BaseConfig{Name: "s5-name"}}}, "s5-name"},
		{"empty", ProxyConfig{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pc.Name(); got != tt.want {
				t.Errorf("Name() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProxyConfig_String(t *testing.T) {
	tests := []struct {
		name     string
		pc       ProxyConfig
		contains string
	}{
		{"SS", ProxyConfig{SS: &SSConfig{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 8388}, Method: "aes"}}, "SS[1.2.3.4:8388"},
		{"VMess", ProxyConfig{VMess: &VMessConfig{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 443}, UUID: "abc-uuid"}}, "VMess[1.2.3.4:443"},
		{"VLess", ProxyConfig{VLess: &VLessConfig{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 443}, UUID: "abc-uuid"}}, "VLess[1.2.3.4:443"},
		{"Trojan", ProxyConfig{Trojan: &TrojanConfig{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 443}}}, "Trojan[1.2.3.4:443]"},
		{"Reality", ProxyConfig{Reality: &RealityConfig{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 443}, UUID: "abc-uuid"}}, "Reality[1.2.3.4:443"},
		{"WireGuard", ProxyConfig{WireGuard: &WireGuardConfig{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 51820}}}, "WireGuard[1.2.3.4:51820"},
		{"TUIC", ProxyConfig{TUIC: &TUICConfig{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 443}}}, "TUIC[1.2.3.4:443"},
		{"Hysteria2", ProxyConfig{Hysteria2: &Hysteria2Config{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 443}}}, "Hysteria2[1.2.3.4:443"},
		{"Socks5", ProxyConfig{Socks5: &Socks5Config{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 1080}}}, "Socks5[1.2.3.4:1080"},
		{"empty", ProxyConfig{}, "empty"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pc.String()
			if !contains(got, tt.contains) {
				t.Errorf("String() = %q, want contains %q", got, tt.contains)
			}
		})
	}
}

func TestProxyConfig_Base(t *testing.T) {
	tests := []struct {
		name     string
		pc       ProxyConfig
		wantHost string
		wantPort int
	}{
		{"SS", ProxyConfig{SS: &SSConfig{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 8388}}}, "1.2.3.4", 8388},
		{"VMess", ProxyConfig{VMess: &VMessConfig{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 443}}}, "1.2.3.4", 443},
		{"VLess", ProxyConfig{VLess: &VLessConfig{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 443}}}, "1.2.3.4", 443},
		{"Trojan", ProxyConfig{Trojan: &TrojanConfig{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 443}}}, "1.2.3.4", 443},
		{"Reality", ProxyConfig{Reality: &RealityConfig{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 443}}}, "1.2.3.4", 443},
		{"WireGuard", ProxyConfig{WireGuard: &WireGuardConfig{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 51820}}}, "1.2.3.4", 51820},
		{"TUIC", ProxyConfig{TUIC: &TUICConfig{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 443}}}, "1.2.3.4", 443},
		{"Hysteria2", ProxyConfig{Hysteria2: &Hysteria2Config{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 443}}}, "1.2.3.4", 443},
		{"Socks5", ProxyConfig{Socks5: &Socks5Config{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 1080}}}, "1.2.3.4", 1080},
		{"empty", ProxyConfig{}, "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := tt.pc.Base()
			if base.Server != tt.wantHost || base.Port != tt.wantPort {
				t.Errorf("Base() = %+v, want server=%q port=%d", base, tt.wantHost, tt.wantPort)
			}
		})
	}
}

func TestProxyConfig_Addr(t *testing.T) {
	tests := []struct {
		name string
		pc   ProxyConfig
		want string
	}{
		{"SS", ProxyConfig{SS: &SSConfig{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 8388}}}, "1.2.3.4:8388"},
		{"VMess", ProxyConfig{VMess: &VMessConfig{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 443}}}, "1.2.3.4:443"},
		{"VLess", ProxyConfig{VLess: &VLessConfig{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 443}}}, "1.2.3.4:443"},
		{"Trojan", ProxyConfig{Trojan: &TrojanConfig{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 443}}}, "1.2.3.4:443"},
		{"Reality", ProxyConfig{Reality: &RealityConfig{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 443}}}, "1.2.3.4:443"},
		{"WireGuard", ProxyConfig{WireGuard: &WireGuardConfig{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 51820}}}, "1.2.3.4:51820"},
		{"TUIC", ProxyConfig{TUIC: &TUICConfig{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 443}}}, "1.2.3.4:443"},
		{"Hysteria2", ProxyConfig{Hysteria2: &Hysteria2Config{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 443}}}, "1.2.3.4:443"},
		{"Socks5", ProxyConfig{Socks5: &Socks5Config{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 1080}}}, "1.2.3.4:1080"},
		{"empty", ProxyConfig{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pc.Addr(); got != tt.want {
				t.Errorf("Addr() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPerConfigStrings(t *testing.T) {
	// Each per-config String() method should return a non-empty string
	cfg := SSConfig{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 8388}, Method: "aes"}
	if cfg.String() == "" {
		t.Error("SSConfig.String() returned empty")
	}
	cfg2 := VMessConfig{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 443}, UUID: "abc"}
	if cfg2.String() == "" {
		t.Error("VMessConfig.String() returned empty")
	}
	cfg3 := VLessConfig{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 443}, UUID: "abc"}
	if cfg3.String() == "" {
		t.Error("VLessConfig.String() returned empty")
	}
	cfg4 := TrojanConfig{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 443}}
	if cfg4.String() == "" {
		t.Error("TrojanConfig.String() returned empty")
	}
	cfg5 := RealityConfig{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 443}, UUID: "abc"}
	cfg6 := WireGuardConfig{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 51820}}
	cfg7 := TUICConfig{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 443}}
	cfg8 := Hysteria2Config{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 443}}
	cfg9 := Socks5Config{BaseConfig: BaseConfig{Server: "1.2.3.4", Port: 1080}}
	if cfg5.String() == "" {
		t.Error("RealityConfig.String() returned empty")
	}
	if cfg6.String() == "" {
		t.Error("WireGuardConfig.String() returned empty")
	}
	if cfg7.String() == "" {
		t.Error("TUICConfig.String() returned empty")
	}
	if cfg8.String() == "" {
		t.Error("Hysteria2Config.String() returned empty")
	}
	if cfg9.String() == "" {
		t.Error("Socks5Config.String() returned empty")
	}
}

// contains is a tiny helper to avoid importing strings just for one call.
func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestValidationResult(t *testing.T) {
	vr := ValidationResult{Valid: true}
	vr.AddWarning("old cipher")
	if !vr.Valid {
		t.Error("warning should not invalidate")
	}
	vr.AddError("bad port")
	if vr.Valid {
		t.Error("error should invalidate")
	}
	if len(vr.Errors) != 1 || len(vr.Warnings) != 1 {
		t.Fatalf("unexpected counts: errors=%d warnings=%d", len(vr.Errors), len(vr.Warnings))
	}
}

func TestSummary_String(t *testing.T) {
	s := Summary{Total: 100, Passed: 95, Failed: 3, Errors: 2, SuccessRatePct: 95.0, ConfigsPerSecond: 12.5}
	out := s.String()
	if out == "" {
		t.Error("Summary.String() returned empty")
	}
}

func TestTestDepth_IsValid(t *testing.T) {
	for _, d := range []TestDepth{DepthQuick, DepthStandard, DepthFull, DepthComprehensive} {
		if !d.IsValid() {
			t.Errorf("%q should be valid", d)
		}
	}
	if TestDepth("bad").IsValid() {
		t.Error("bad depth should be invalid")
	}
}

func TestOutputStyle_IsValid(t *testing.T) {
	for _, s := range []OutputStyle{StyleAuto, StyleJSON, StyleCSV, StyleTable, StyleMarkdown, StyleHTML} {
		if !s.IsValid() {
			t.Errorf("%q should be valid", s)
		}
	}
	if OutputStyle("bad").IsValid() {
		t.Error("bad style should be invalid")
	}
}

func TestJSONMarshaling(t *testing.T) {
	res := TestResult{
		ID:        "test-1",
		Status:    StatusSuccess,
		Stage:     StageCompleted,
		Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Latencies: LatencyBreakdown{Connect: 10 * time.Millisecond},
	}
	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var back TestResult
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.ID != res.ID || back.Status != res.Status {
		t.Errorf("round-trip mismatch")
	}
}
