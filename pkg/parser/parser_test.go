package parser

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/amiralavi/viberay/internal/models"
)

// --- Detector ---

func TestDetectProtocol(t *testing.T) {
	tests := []struct {
		input   string
		want    models.Protocol
		wantErr bool
	}{
		{"ss://YWVzLTI1Ni1nY206dGVzdA==@1.2.3.4:8388#Test", models.ProtocolSS, false},
		{"vmess://eyJ2IjogIjIiLCAicHMiOiAiVGVzdCIsICJhZGQiOiAiMS4yLjMuNCIsICJwb3J0IjogIjQ0MyIsICJpZCI6ICJhMWIyYzNkNC1lNWY2LTc4OTAtYWJjZC1lZjEyMzQ1Njc4OTAifQ==", models.ProtocolVMess, false},
		{"vless://a1b2c3d4-e5f6-7890-abcd-ef1234567890@1.2.3.4:443?type=tcp", models.ProtocolVLess, false},
		{"trojan://password@1.2.3.4:443", models.ProtocolTrojan, false},
		{"random://data", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.input[:8], func(t *testing.T) {
			got, err := DetectProtocol(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("DetectProtocol() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("DetectProtocol() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractFragment(t *testing.T) {
	uri, name := ExtractFragment("ss://xxx@1.2.3.4:8388#My%20Server")
	if name != "My Server" {
		t.Errorf("fragment decode failed: got %q", name)
	}
	if strings.Contains(uri, "#") {
		t.Error("URI should not contain #")
	}
}

func TestLooksLikeBase64(t *testing.T) {
	if !LooksLikeBase64("dGVzdA==") {
		t.Error("expected base64-looking string to match")
	}
	if LooksLikeBase64("hello world!") {
		t.Error("plain text should not match")
	}
	if LooksLikeBase64("abc") { // length not divisible by 4
		t.Error("bad length should not match")
	}
}

// --- Per-protocol parsing ---

func TestParseSS(t *testing.T) {
	// SIP002 plain
	cfg, err := ParseSingle("ss://chacha20-ietf-poly1305:pass@192.0.2.1:8388#SS-Test")
	if err != nil {
		t.Fatalf("parse SS plain: %v", err)
	}
	ss := cfg.SS
	if ss == nil {
		t.Fatal("expected SS config")
	}
	if ss.Method != "chacha20-ietf-poly1305" || ss.Password != "pass" {
		t.Errorf("method/password mismatch")
	}
	if ss.Server != "192.0.2.1" || ss.Port != 8388 {
		t.Errorf("addr mismatch: %s:%d", ss.Server, ss.Port)
	}
	if ss.Name != "SS-Test" {
		t.Errorf("name mismatch: %q", ss.Name)
	}

	// Base64 encoded method:password
	b64 := base64.StdEncoding.EncodeToString([]byte("aes-256-gcm:secret"))
	cfg2, err := ParseSingle("ss://" + b64 + "@192.0.2.2:1080")
	if err != nil {
		t.Fatalf("parse SS base64: %v", err)
	}
	if cfg2.SS.Method != "aes-256-gcm" || cfg2.SS.Password != "secret" {
		t.Errorf("base64 decode mismatch")
	}
}

func TestParseVMess(t *testing.T) {
	j := `{"v":"2","ps":"Test","add":"1.2.3.4","port":"443","id":"a1b2c3d4-e5f6-7890-abcd-ef1234567890","aid":"0","scy":"auto","net":"ws","tls":"tls","sni":"example.com","host":"cdn.example.com","path":"/ws","fp":"chrome"}`
	b64 := base64.StdEncoding.EncodeToString([]byte(j))
	cfg, err := ParseSingle("vmess://" + b64 + "#VMess-Test")
	if err != nil {
		t.Fatalf("parse VMess: %v", err)
	}
	v := cfg.VMess
	if v == nil {
		t.Fatal("expected VMess config")
	}
	if v.UUID != "a1b2c3d4-e5f6-7890-abcd-ef1234567890" {
		t.Errorf("UUID mismatch: %s", v.UUID)
	}
	if v.Port != 443 || v.Server != "1.2.3.4" {
		t.Errorf("addr mismatch")
	}
	if !v.TLSConfig.Enabled {
		t.Error("TLS should be enabled")
	}
	if v.TLSConfig.SNI != "example.com" || v.TLSConfig.Fingerprint != "chrome" {
		t.Errorf("TLS settings mismatch")
	}
	if v.Network != "ws" || v.Security != "auto" {
		t.Errorf("network/security mismatch")
	}
	if v.Name != "VMess-Test" {
		t.Errorf("name mismatch: %q", v.Name)
	}
}

func TestParseVLess(t *testing.T) {
	cfg, err := ParseSingle("vless://a1b2c3d4-e5f6-7890-abcd-ef1234567890@example.com:443?type=tcp&security=tls&sni=example.com&fp=firefox&flow=xtls-rprx-vision#VLess-Test")
	if err != nil {
		t.Fatalf("parse VLess: %v", err)
	}
	vl := cfg.VLess
	if vl == nil {
		t.Fatal("expected VLess config")
	}
	if vl.UUID != "a1b2c3d4-e5f6-7890-abcd-ef1234567890" {
		t.Errorf("UUID mismatch")
	}
	if !vl.TLSConfig.Enabled || vl.TLSConfig.SNI != "example.com" {
		t.Errorf("TLS settings mismatch")
	}
	if vl.Flow != "xtls-rprx-vision" || vl.Encryption != "none" {
		t.Errorf("flow/encryption mismatch: flow=%s enc=%s", vl.Flow, vl.Encryption)
	}
	if vl.Name != "VLess-Test" {
		t.Errorf("name mismatch")
	}
}

func TestParseTrojan(t *testing.T) {
	cfg, err := ParseSingle("trojan://mypassword@tr.example.com:443?type=ws&host=cdn.example.com&path=/path&fp=safari#Trojan-Test")
	if err != nil {
		t.Fatalf("parse Trojan: %v", err)
	}
	tr := cfg.Trojan
	if tr == nil {
		t.Fatal("expected Trojan config")
	}
	if tr.Password != "mypassword" {
		t.Errorf("password mismatch")
	}
	if !tr.TLSConfig.Enabled {
		t.Error("Trojan should always have TLS enabled")
	}
	if tr.Network != "ws" || tr.TLSConfig.Host != "cdn.example.com" {
		t.Errorf("network/host mismatch")
	}
	if tr.Name != "Trojan-Test" {
		t.Errorf("name mismatch")
	}
}

func TestParseReality(t *testing.T) {
	// VLess URL with security=reality + pbk
	pk := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=" // base64, 32 zero bytes → 44 chars
	cfg, err := ParseSingle("vless://a1b2c3d4-e5f6-7890-abcd-ef1234567890@reality.example.com:443?security=reality&pbk=" + pk + "&sid=00112233&spx=/download&fp=chrome&type=tcp&flow=xtls-rprx-vision&sni=www.example.com#Reality-Test")
	if err != nil {
		t.Fatalf("parse Reality: %v", err)
	}
	r := cfg.Reality
	if r == nil {
		t.Fatal("expected Reality config")
	}
	if r.Server != "reality.example.com" || r.Port != 443 {
		t.Errorf("addr mismatch")
	}
	if r.PublicKey != pk {
		t.Errorf("public key mismatch")
	}
	if r.ShortID != "00112233" || r.SpiderX != "/download" {
		t.Errorf("short_id/spider_x mismatch")
	}
	if r.Flow != "xtls-rprx-vision" {
		t.Errorf("flow mismatch")
	}
	if r.SNI != "www.example.com" || r.Fingerprint != "chrome" {
		t.Errorf("TLS settings mismatch")
	}
	if r.Name != "Reality-Test" {
		t.Errorf("name mismatch")
	}
}

// --- Batch parsing ---

func TestParseBatchBase64(t *testing.T) {
	lines := []string{
		"ss://chacha20-ietf-poly1305:pass@1.2.3.4:8388#A",
		"trojan://pw@2.3.4.5:443#B",
		"vless://a1b2c3d4-e5f6-7890-abcd-ef1234567890@3.4.5.6:443#C",
	}
	input := base64.StdEncoding.EncodeToString([]byte(strings.Join(lines, "\n")))
	configs, err := Parse(input)
	if err != nil {
		t.Fatalf("batch parse: %v", err)
	}
	if len(configs) != 3 {
		t.Fatalf("expected 3 configs, got %d", len(configs))
	}
	if configs[0].Protocol() != models.ProtocolSS {
		t.Errorf("first should be SS")
	}
	if configs[1].Protocol() != models.ProtocolTrojan {
		t.Errorf("second should be Trojan")
	}
	if configs[2].Protocol() != models.ProtocolVLess {
		t.Errorf("third should be VLess")
	}
}

func TestParseBatchPlain(t *testing.T) {
	input := `# Subscription
ss://method:pw@1.1.1.1:8388#First

vmess://eyJ2IjoiMiIsInBzIjoiU2Vjb25kIiwiYWRkIjoiMi4yLjIuMiIsInBvcnQiOiI0NDMiLCJpZCI6ImExYjJjM2Q0LWU1ZjYtNzg5MC1hYmNkLWVmMTIzNDU2Nzg5MCJ9

// comment line should be skipped if not URI
`
	configs, err := Parse(input)
	if err != nil {
		t.Fatalf("plain batch parse: %v", err)
	}
	if len(configs) != 2 {
		t.Fatalf("expected 2 configs, got %d", len(configs))
	}
}

// --- Validation ---

func TestValidateUUID(t *testing.T) {
	if err := ValidateUUID("a1b2c3d4-e5f6-7890-abcd-ef1234567890"); err != nil {
		t.Errorf("valid UUID rejected: %v", err)
	}
	if err := ValidateUUID("a1b2c3d4e5f67890abcdef1234567890"); err != nil {
		t.Errorf("valid 32-char UUID rejected: %v", err)
	}
	if err := ValidateUUID("not-a-uuid"); err == nil {
		t.Error("invalid UUID accepted")
	}
}

func TestValidatePortInt(t *testing.T) {
	if err := ValidatePortInt(1); err != nil {
		t.Error("port 1 rejected")
	}
	if err := ValidatePortInt(65535); err != nil {
		t.Error("port 65535 rejected")
	}
	if err := ValidatePortInt(0); err == nil {
		t.Error("port 0 accepted")
	}
	if err := ValidatePortInt(70000); err == nil {
		t.Error("port 70000 accepted")
	}
}

func TestValidatePublicKey(t *testing.T) {
	// 32 zero bytes → 44 chars base64 (43 A's + 1 pad)
	good := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	if err := ValidatePublicKey(good); err != nil {
		t.Errorf("valid key rejected: %v", err)
	}
	if err := ValidatePublicKey("short"); err == nil {
		t.Error("short key accepted")
	}
	if err := ValidatePublicKey("not valid base64!!!"); err == nil {
		t.Error("bad base64 accepted")
	}
}

func TestValidateFlow(t *testing.T) {
	if err := ValidateFlow("xtls-rprx-vision"); err != nil {
		t.Errorf("valid flow rejected: %v", err)
	}
	if err := ValidateFlow(""); err != nil {
		t.Errorf("empty flow rejected: %v", err)
	}
	if err := ValidateFlow("bad-flow"); err == nil {
		t.Error("invalid flow accepted")
	}
}

func TestIsRealityURL(t *testing.T) {
	if !IsRealityURL("vless://uuid@host:443?security=reality&pbk=xxx") {
		t.Error("expected reality detection")
	}
	if IsRealityURL("vless://uuid@host:443?security=tls") {
		t.Error("plain TLS misdetected as reality")
	}
}
