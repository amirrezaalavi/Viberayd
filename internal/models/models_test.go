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

func TestProxyConfig_Addr(t *testing.T) {
	pc := ProxyConfig{VLess: &VLessConfig{BaseConfig: BaseConfig{Server: "example.com", Port: 443}}}
	if got := pc.Addr(); got != "example.com:443" {
		t.Errorf("Addr() = %s, want example.com:443", got)
	}
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
