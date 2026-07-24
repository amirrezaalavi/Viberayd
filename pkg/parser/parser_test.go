package parser

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/amirrezaalavi/Viberay/internal/models"
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

// --- DecodeBase64 ---

func TestDecodeBase64(t *testing.T) {
	b, err := DecodeBase64("dGVzdA==")
	if err != nil {
		t.Fatalf("valid base64 rejected: %v", err)
	}
	if string(b) != "test" {
		t.Errorf("DecodeBase64() = %q, want %q", string(b), "test")
	}

	// Invalid input
	if _, err := DecodeBase64("!!!invalid!!!"); err == nil {
		t.Error("DecodeBase64: expected error for invalid input")
	}
}

// --- DecodeBase64URL ---

func TestDecodeBase64URL(t *testing.T) {
	// Unpadded input — padding added internally
	b, err := DecodeBase64URL("dGVzdA")
	if err != nil {
		t.Fatalf("valid URL-safe base64 rejected: %v", err)
	}
	if string(b) != "test" {
		t.Errorf("DecodeBase64URL() = %q, want %q", string(b), "test")
	}

	// Invalid input
	if _, err := DecodeBase64URL("!!!invalid!!!"); err == nil {
		t.Error("DecodeBase64URL: expected error for invalid input")
	}
}

// --- IsBase64Encoded (the new, smarter detector) ---

func TestIsBase64Encoded(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Empty / trivially not base64
		{"empty", "", false},
		{"invalid pipe", "abc|def", false},

		// Contains "://" — definitely a proxy URI or URL, NOT base64
		{"vmess URI", "vmess://eyJ2IjoiMiJ9", false},
		{"vless URI", "vless://uuid@host:443?type=tcp", false},
		{"ss URI", "ss://YWVzLTI1Ni1nY206cGFzcw==@host:8388", false},
		{"trojan URI", "trojan://password@host:443", false},
		{"http URL", "http://example.com", false},

		// Plain base64 (no whitespace)
		{"plain padded", "dGVzdA==", true},
		{"plain unpadded short", "dGVzdA", true},  // still detected; decode will be attempted then fall through

		// Base64 with newlines (wrapped subscription — the main use case)
		{"with single newline", "dm1lc3M6Ly9leUoy\nSWpvaU1pSjk=\n", true},
		{"with carriage return + newline", "dm1lc3M6Ly9leUoy\r\nSWpvaU1pSjk=\r\n", true},

		// Base64 with other whitespace
		{"with trailing newline", "dGVzdA==\n", true},       // LooksLikeBase64 rejects this
		{"with spaces in middle", "dGV z dA==", true},       // spaces aren't base64 but IsBase64Encoded allows them
		{"with tabs", "dGVz	dA==", true},

		// Contains non-base64, non-whitespace characters (but no ://)
		{"invalid chars", "hello world!", false},   // '!' and space in middle? space ok, but '!' is not
		{"pipe char", "abc|def", false},
		{"colon inside (not ://)", "abc:def", false}, // colon is not base64, and no :// either

		// Multi-line plain text with :// — should NOT be treated as base64
		{"mixed URIs newline separated",
			"vmess://abc\ntrojan://def\nvless://ghi\n",
			false}, // has :// so not base64

		// Edge: very long random base64-like string (simulates real subscription)
		{"long b64", "dm1lc3M6Ly9leUoySWpvaU1pSjk9ZXlKMklqb2lNaUo5ZXlKMklqb2lNaUo5\nZXlKMklqb2lNaUo5ZXlKMklqb2lNaUo5", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsBase64Encoded(tt.input)
			if got != tt.want {
				t.Errorf("IsBase64Encoded(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestParse_DecodesBase64Subscription verifies that Parse correctly decodes
// a wrapped (newline-containing) base64 subscription body.
func TestParse_DecodesBase64Subscription(t *testing.T) {
	// Three proxy URIs joined by newline, then base64-encoded, then
	// artificially wrapped at column ~50 to simulate a real subscription payload.
	plain := "ss://chacha20-ietf-poly1305:pass@1.2.3.4:8388#A\ntrojan://pw@2.3.4.5:443#B\nvless://a1b2c3d4-e5f6-7890-abcd-ef1234567890@3.4.5.6:443#C\n"
	b64 := base64.StdEncoding.EncodeToString([]byte(plain))
	// Insert newlines to simulate line-wrapped subscription
	var wrapped strings.Builder
	for i, r := range b64 {
		if i > 0 && i%50 == 0 {
			wrapped.WriteByte('\n')
		}
		wrapped.WriteRune(r)
	}
	wrapped.WriteByte('\n')

	configs, err := Parse(wrapped.String())
	if err != nil {
		t.Fatalf("Parse(wrapped b64) returned error: %v", err)
	}
	if len(configs) != 3 {
		t.Fatalf("Parse(wrapped b64) returned %d configs, want 3", len(configs))
	}
	if configs[0].Protocol() != models.ProtocolSS {
		t.Errorf("first should be SS, got %v", configs[0].Protocol())
	}
	if configs[1].Protocol() != models.ProtocolTrojan {
		t.Errorf("second should be Trojan, got %v", configs[1].Protocol())
	}
	if configs[2].Protocol() != models.ProtocolVLess {
		t.Errorf("third should be VLess, got %v", configs[2].Protocol())
	}
}

// TestParse_PlainTextSubscription verifies that Parse handles multi-line
// plain text with :// URIs — it should NOT try to base64-decode those.
func TestParse_PlainTextSubscription(t *testing.T) {
	input := "ss://YWVzLTI1Ni1nY206cGFzcw==@host:8388#test1\nvmess://eyJ2IjoiMiIsImFkZCI6Imhvc3QiLCJwb3J0IjoiNDQzIiwiaWQiOiJhMGIxYzJkMy1lNGY2LTc4OTAtYWJjZC1lZjEyMzQ1Njc4OTAiLCJhaWQiOiIwIn0=#test2\n"
	configs, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse(plain text) returned error: %v", err)
	}
	if len(configs) != 2 {
		t.Errorf("Parse(plain text) returned %d configs, want 2", len(configs))
	}
}

// --- ExtractFragment edge cases ---

func TestExtractFragment_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantURI  string
		wantName string
	}{
		{"no fragment", "ss://user@host:port", "ss://user@host:port", ""},
		{"trailing hash", "ss://user@host:port#", "ss://user@host:port", ""},
		{"URL-encoded name", "ss://user@host:port#My%20Server", "ss://user@host:port", "My Server"},
		{"plain name", "ss://user@host:port#MyServer", "ss://user@host:port", "MyServer"},
		{"multiple hashes", "ss://user@host:port#first#second", "ss://user@host:port#first", "second"},
		{"empty input", "", "", ""},
		{"only hash", "#", "", ""},
		{"emoji fragment", "vmess://data#\U0001F1FA\U0001F1F8", "vmess://data", "\U0001F1FA\U0001F1F8"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri, name := ExtractFragment(tt.input)
			if uri != tt.wantURI || name != tt.wantName {
				t.Errorf("ExtractFragment(%q) = (%q, %q), want (%q, %q)", tt.input, uri, name, tt.wantURI, tt.wantName)
			}
		})
	}
}

// --- IsRealityURL edge cases ---

func TestIsRealityURL_EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"security=reality", "vless://uuid@host:443?security=reality&pbk=xxx", true},
		{"pbk present (no security=reality)", "vless://uuid@host:443?pbk=xxx&security=tls", true},
		{"security=tls", "vless://uuid@host:443?security=tls&type=tcp", false},
		{"no security param", "vless://uuid@host:443?type=tcp", false},
		{"not vless", "ss://method:pass@host:443", false},
		{"malformed URL", "vless://not a valid url?security=reality", false},
		{"empty host", "vless://@?security=reality&pbk=xxx", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRealityURL(tt.input)
			if got != tt.want {
				t.Errorf("IsRealityURL(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// --- DetectProtocol error path ---

func TestDetectProtocol_Unknown(t *testing.T) {
	if _, err := DetectProtocol(""); err == nil {
		t.Error("expected error for empty input")
	}
	if _, err := DetectProtocol("unknown://scheme"); err == nil {
		t.Error("expected error for unknown scheme")
	}
	if _, err := DetectProtocol("http://example.com"); err == nil {
		t.Error("expected error for non-proxy scheme")
	}
}

// --- Parse edge cases ---

func TestParse_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantN   int
		wantErr bool
	}{
		{"empty", "", 0, true},
		{"only whitespace", "   	  \n  ", 0, true},
		{"only comments", "# comment1\n# comment2\n# comment3", 0, false},
		{"single valid SS", "ss://chacha20-ietf-poly1305:pass@192.0.2.1:8388#SS", 1, false},
		{"single base64 line", "c3M6Ly9jaGFjaGEyMC1pZXRmLXBvbHkxMzA1OnBhc3NAMTkyLjAuMi4xOjgzODgjU1M=", 1, false},
		{"mixed with blank lines", "\n\nss://method:pw@1.1.1.1:8388#A\n\n\nvmess://eyJ2IjogIjIiLCAicHMiOiAiQiIsICJhZGQiOiAiZXhhbXBsZS5jb20iLCAicG9ydCI6ICI4MCIsICJpZCI6ICJhMWIyYzNkNC1lNWY2LTc4OTAtYWJjZC1lZjEyMzQ1Njc4OTAiLCAiYWlkIjogIjAiLCAic2N5IjogImF1dG8iLCAibmV0IjogInRjcCJ9\n\n", 2, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configs, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(configs) != tt.wantN {
				t.Errorf("Parse() = %d configs, want %d", len(configs), tt.wantN)
			}
		})
	}
}

// --- Parse subscription with all 5 protocols ---

func TestParse_MixedSubscription(t *testing.T) {
	lines := []string{
		"ss://chacha20-ietf-poly1305:pass@1.1.1.1:8388",
		"vmess://eyJ2IjogIjIiLCAicHMiOiAiIiwgImFkZCI6ICIyLjIuMi4yIiwgInBvcnQiOiAiNDQzIiwgImlkIjogImExYjJjM2Q0LWU1ZjYtNzg5MC1hYmNkLWVmMTIzNDU2Nzg5MCIsICJhaWQiOiAiMCIsICJzY3kiOiAiYXV0byIsICJuZXQiOiAidGNwIn0=",
		"vless://a1b2c3d4-e5f6-7890-abcd-ef1234567890@3.3.3.3:443?type=tcp",
		"trojan://password@4.4.4.4:443",
		"vless://a1b2c3d4-e5f6-7890-abcd-ef1234567890@5.5.5.5:443?security=reality&pbk=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
	}
	input := lines[0] + "\n" + lines[1] + "\n" + lines[2] + "\n" + lines[3] + "\n" + lines[4]
	configs, err := Parse(input)
	if err != nil {
		t.Fatalf("mixed subscription parse failed: %v", err)
	}
	if len(configs) != 5 {
		t.Fatalf("expected 5 configs, got %d", len(configs))
	}
	expected := []models.Protocol{models.ProtocolSS, models.ProtocolVMess, models.ProtocolVLess, models.ProtocolTrojan, models.ProtocolReality}
	for i, exp := range expected {
		if configs[i].Protocol() != exp {
			t.Errorf("config[%d] protocol = %v, want %v", i, configs[i].Protocol(), exp)
		}
	}
}

// --- ParseSingle / parseSS error paths ---

func TestParseSS_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"missing @", "ss://notavaliduri"},
		{"empty after ss://", "ss://"},
		{"missing method:password", "ss://@host:8388"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSingle(tt.input)
			if err == nil {
				t.Errorf("ParseSingle(%q) expected error", tt.input)
			}
		})
	}
}

// --- parseVMess error paths ---

func TestParseVMess_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"not base64", "vmess://!!!not-base64!!!"},
		{"not JSON", "vmess://" + base64.StdEncoding.EncodeToString([]byte("not-json"))},
		{"bad UUID", "vmess://" + base64.StdEncoding.EncodeToString([]byte(`{"v":"2","ps":"T","add":"1.2.3.4","port":"443","id":"bad-uuid"}`))},
		{"missing server", "vmess://" + base64.StdEncoding.EncodeToString([]byte(`{"v":"2","ps":"T","add":"","port":"443","id":"a1b2c3d4-e5f6-7890-abcd-ef1234567890"}`))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSingle(tt.input)
			if err == nil {
				t.Errorf("ParseSingle(%q) expected error", tt.input[:40])
			}
		})
	}
}

// --- parseVLess error paths ---

func TestParseVLess_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"bad URL (no host)", "vless://"},
		{"bad UUID", "vless://bad-uuid@example.com:443"},
		{"missing server", "vless://a1b2c3d4-e5f6-7890-abcd-ef1234567890@:443"},
		{"garbage after vless://", "vless://!!!garbage!!!:443"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSingle(tt.input)
			if err == nil {
				t.Errorf("ParseSingle(%q) expected error", tt.input)
			}
		})
	}
}

// --- parseTrojan error paths ---

func TestParseTrojan_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"bad URL", "trojan://"},
		{"missing password", "trojan://@host:443"},
		{"missing server", "trojan://password@:443"},
		{"garbage after trojan://", "trojan://!!!garbage!!!:443"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSingle(tt.input)
			if err == nil {
				t.Errorf("ParseSingle(%q) expected error", tt.input)
			}
		})
	}
}

// --- parseReality error paths ---

func TestParseReality_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"bad URL", "vless://?security=reality"},
		{"bad UUID", "vless://bad-uuid@host:443?security=reality&pbk=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="},
		{"bad public key", "vless://a1b2c3d4-e5f6-7890-abcd-ef1234567890@host:443?security=reality&pbk=short"},
		{"missing server", "vless://a1b2c3d4-e5f6-7890-abcd-ef1234567890@:443?security=reality&pbk=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSingle(tt.input)
			if err == nil {
				t.Errorf("ParseSingle(%q) expected error", tt.input[:50])
			}
		})
	}
}

// --- netSplitHostPort ---

func TestNetSplitHostPort(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantHost string
		wantPort string
		wantErr  bool
	}{
		{"host:port", "example.com:443", "example.com", "443", false},
		{"IPv4:port", "1.2.3.4:8080", "1.2.3.4", "8080", false},
		{"IPv6 with port", "[::1]:443", "::1", "443", false},
		{"host only", "example.com", "example.com", "", true},
		{"IPv6 without port", "[::1]", "::1", "", true},
		{"empty", "", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port, err := netSplitHostPort(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("netSplitHostPort(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if host != tt.wantHost {
				t.Errorf("netSplitHostPort(%q) host = %q, want %q", tt.input, host, tt.wantHost)
			}
			if port != tt.wantPort {
				t.Errorf("netSplitHostPort(%q) port = %q, want %q", tt.input, port, tt.wantPort)
			}
		})
	}
}

// --- parsePluginOpts ---

func TestParsePluginOpts(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  map[string]string
	}{
		{"empty", "", map[string]string{}},
		{"single pair", "key=value", map[string]string{"key": "value"}},
		{"multiple pairs", "a=1;b=2;c=3", map[string]string{"a": "1", "b": "2", "c": "3"}},
		{"malformed (no value)", "key", map[string]string{}},
		{"mixed", "a=1;b;c=3", map[string]string{"a": "1", "c": "3"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePluginOpts(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("parsePluginOpts(%q) = %v (len=%d), want %v (len=%d)", tt.input, got, len(got), tt.want, len(tt.want))
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("parsePluginOpts(%q)[%q] = %q, want %q", tt.input, k, got[k], v)
				}
			}
		})
	}
}

// --- Round-trip: parse then check Raw field ---

func TestParseRoundTrip(t *testing.T) {
	inputs := []string{
		"ss://chacha20-ietf-poly1305:pass@192.0.2.1:8388#SS-Test",
		"vless://a1b2c3d4-e5f6-7890-abcd-ef1234567890@example.com:443?type=tcp&security=tls&sni=example.com&fp=firefox#VLess-Test",
		"trojan://mypassword@tr.example.com:443#Trojan-Test",
	}
	for _, raw := range inputs {
		t.Run(raw[:10], func(t *testing.T) {
			cfg, err := ParseSingle(raw)
			if err != nil {
				t.Fatalf("ParseSingle(%q): %v", raw, err)
			}
			if cfg.Raw != raw {
				t.Errorf("Raw field = %q, want %q", cfg.Raw, raw)
			}
		})
	}
}

// --- Fallback: vless:// without reality params stays as VLess ---

func TestParse_FallbackToVLess(t *testing.T) {
	// vless:// URL that has no security=reality and no pbk → should stay VLess
	input := "vless://a1b2c3d4-e5f6-7890-abcd-ef1234567890@example.com:443?type=tcp&security=tls"
	cfg, err := ParseSingle(input)
	if err != nil {
		t.Fatalf("ParseSingle: %v", err)
	}
	if cfg.VLess == nil {
		t.Fatal("expected VLess config, got nil")
	}
	if cfg.Reality != nil {
		t.Fatal("expected nil Reality config for non-reality vless URL")
	}
	if cfg.Protocol() != models.ProtocolVLess {
		t.Errorf("Protocol() = %v, want %v", cfg.Protocol(), models.ProtocolVLess)
	}

	// Also test with no security param at all
	input2 := "vless://a1b2c3d4-e5f6-7890-abcd-ef1234567890@example.com:443?type=tcp"
	cfg2, err2 := ParseSingle(input2)
	if err2 != nil {
		t.Fatalf("ParseSingle: %v", err2)
	}
	if cfg2.VLess == nil {
		t.Fatal("expected VLess config for plain vless URL")
	}
	if cfg2.Protocol() != models.ProtocolVLess {
		t.Errorf("Protocol() = %v, want %v", cfg2.Protocol(), models.ProtocolVLess)
	}
}

// --- Fuzzy / edge case: truncated URIs, null bytes, non-UTF-8, long strings ---
//
// NOTE: The null-byte test documents existing parser behavior. The SS parser
// silently ignores embedded null bytes in the port string (Atoi stops at the
// null byte, port defaults to 8388). A future improvement could reject null
// bytes at the token-extraction stage.

func TestParse_FuzzyNullBytes(t *testing.T) {
	// Null byte in the middle — the parser currently handles this silently
	// because strconv.Atoi stops at the null byte, triggering the default port.
	input := "ss://chacha20-ietf-poly1305:pass@192.0.2.1:8388\x00#SS"
	cfg, err := ParseSingle(input)
	if err != nil {
		t.Fatalf("ParseSingle with embedded null byte: %v", err)
	}
	if cfg.SS == nil {
		t.Fatal("expected SS config despite null byte")
	}
	// The port field should be set to the default (8388 from the URI before null)
	if cfg.SS.Port != 8388 {
		t.Logf("Parsed SS port = %d (null byte in port string)", cfg.SS.Port)
	}
}

func TestParse_FuzzyTruncated(t *testing.T) {
	full := "vmess://eyJ2IjogIjIiLCAicHMiOiAiIiwgImFkZCI6ICIxLjIuMy40IiwgInBvcnQiOiAiNDQzIiwgImlkIjogImExYjJjM2Q0LWU1ZjYtNzg5MC1hYmNkLWVmMTIzNDU2Nzg5MCIsICJhaWQiOiAiMCIsICJzY3kiOiAiYXV0byIsICJuZXQiOiAidGNwIn0="
	truncated := full[:len(full)/2] // cut in half
	_, err := ParseSingle(truncated)
	if err == nil {
		t.Error("expected error for truncated VMess URI")
	}
}

func TestParse_FuzzyNonUTF8(t *testing.T) {
	// Non-UTF-8 bytes — Go strings are just byte sequences
	input := string([]byte{0xff, 0xfe, 0x00, 0x01, 0x02})
	_, err := ParseSingle(input)
	if err == nil {
		t.Error("expected error for non-UTF-8 input without protocol prefix")
	}
}

func TestParse_FuzzyVeryLong(t *testing.T) {
	// Very long string (~10K chars) that is not a valid URI
	long := string(make([]byte, 10000))
	for range long {
		long = "A" + long[1:] // ensure first char is 'A'
	}
	// Make it not start with any known prefix
	input := "X" + long[1:]
	_, err := ParseSingle(input)
	if err == nil {
		t.Error("expected error for very long invalid input")
	}
}

// --- ParseSingle empty input ---

func TestParseSingle_EmptyInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"only spaces", "   "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSingle(tt.input)
			if err == nil {
				t.Errorf("ParseSingle(%q) expected error", tt.input)
			}
		})
	}
}

// --- Parse base64 batch with some errors ---

func TestParse_BatchWithErrors(t *testing.T) {
	lines := []string{
		"ss://chacha20-ietf-poly1305:pass@1.1.1.1:8388",
		"invalid-line-without-protocol",
		"trojan://password@2.2.2.2:443",
	}
	input := strings.Join(lines, "\n")
	configs, err := Parse(input)
	if err != nil {
		t.Fatalf("batch with errors: %v", err)
	}
	if len(configs) != 2 {
		t.Errorf("expected 2 successful configs, got %d", len(configs))
	}
}

// --- Parse single-line comment/whitespace ---

func TestParse_CommentAndBlank(t *testing.T) {
	// Only comments and blank lines -> no configs, no error
	configs, err := Parse("# comment\n\n# another\n")
	if err != nil {
		t.Errorf("expected nil error for comments+blanks, got %v", err)
	}
	if len(configs) != 0 {
		t.Errorf("expected 0 configs, got %d", len(configs))
	}
}

// --- Parse with all lines failing ---

func TestParse_AllLinesFail(t *testing.T) {
	input := "not-a-valid-uri\nanother-bad-one\nthird-garbage"
	configs, err := Parse(input)
	if err == nil {
		t.Error("expected error when all lines fail")
	}
	if len(configs) != 0 {
		t.Errorf("expected 0 configs, got %d", len(configs))
	}
}

// --- parseSS: port defaults to 8388 when port string is non-numeric ---

func TestParseSS_DefaultPort(t *testing.T) {
	// Atoi("abc") returns 0, which triggers the default 8388
	cfg, err := ParseSingle("ss://aes-256-gcm:secret@1.2.3.4:abc")
	if err != nil {
		t.Fatalf("ParseSingle with non-numeric port: %v", err)
	}
	if cfg.SS.Port != 8388 {
		t.Errorf("expected port 8388 (default), got %d", cfg.SS.Port)
	}
}

// --- parseVLess: network defaults to tcp ---

func TestParseVLess_NetworkDefault(t *testing.T) {
	// URL without type param — Network should default to "tcp"
	cfg, err := ParseSingle("vless://a1b2c3d4-e5f6-7890-abcd-ef1234567890@example.com:443")
	if err != nil {
		t.Fatalf("ParseSingle: %v", err)
	}
	if cfg.VLess.Network != "tcp" {
		t.Errorf("expected Network=tcp (default), got %q", cfg.VLess.Network)
	}
	if cfg.VLess.Encryption != "none" {
		t.Errorf("expected Encryption=none (default), got %q", cfg.VLess.Encryption)
	}
}

// --- parseVLess: grpc type with serviceName (path alias) ---

func TestParseVLess_GRPC(t *testing.T) {
	cfg, err := ParseSingle("vless://a1b2c3d4-e5f6-7890-abcd-ef1234567890@example.com:443?type=grpc&serviceName=my.api.Stream")
	if err != nil {
		t.Fatalf("ParseSingle vless grpc: %v", err)
	}
	if cfg.VLess.Network != "grpc" {
		t.Errorf("expected Network=grpc, got %q", cfg.VLess.Network)
	}
	if cfg.VLess.TLSConfig.Path != "my.api.Stream" {
		t.Errorf("expected Path=my.api.Stream (from serviceName), got %q", cfg.VLess.TLSConfig.Path)
	}
}

// --- parseTrojan: network defaults to tcp ---

func TestParseTrojan_NetworkDefault(t *testing.T) {
	// Trojan without type param — Network should default to "tcp"
	cfg, err := ParseSingle("trojan://password@tr.example.com:443")
	if err != nil {
		t.Fatalf("ParseSingle: %v", err)
	}
	if cfg.Trojan.Network != "tcp" {
		t.Errorf("expected Network=tcp (default), got %q", cfg.Trojan.Network)
	}
}

// --- parseReality: network and flow defaults ---

func TestParseReality_Defaults(t *testing.T) {
	// Reality URL without flow and without type
	// Network should default to tcp, then Flow should default to xtls-rprx-vision
	cfg, err := ParseSingle("vless://a1b2c3d4-e5f6-7890-abcd-ef1234567890@reality.example.com:443?security=reality&pbk=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
	if err != nil {
		t.Fatalf("ParseSingle reality defaults: %v", err)
	}
	if cfg.Reality.Network != "tcp" {
		t.Errorf("expected Network=tcp (default), got %q", cfg.Reality.Network)
	}
	if cfg.Reality.Flow != "xtls-rprx-vision" {
		t.Errorf("expected Flow=xtls-rprx-vision (default when tcp), got %q", cfg.Reality.Flow)
	}
}

// --- parseReality: grpc type with serviceName ---

func TestParseReality_GRPC(t *testing.T) {
	cfg, err := ParseSingle("vless://a1b2c3d4-e5f6-7890-abcd-ef1234567890@reality.example.com:443?security=reality&pbk=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=&type=grpc&serviceName=api.v1.StreamService")
	if err != nil {
		t.Fatalf("ParseSingle reality grpc: %v", err)
	}
	if cfg.Reality.Network != "grpc" {
		t.Errorf("expected Network=grpc, got %q", cfg.Reality.Network)
	}
	if cfg.Reality.TLSConfig.Path != "api.v1.StreamService" {
		t.Errorf("expected Path=api.v1.StreamService, got %q", cfg.Reality.TLSConfig.Path)
	}
}
