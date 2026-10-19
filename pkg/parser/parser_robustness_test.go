package parser

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestDecodeBase64AlphabetsAndPadding(t *testing.T) {
	// Bytes that differ between the standard and URL-safe alphabets.
	raw := []byte{0xfb, 0xff, 0x3e, 0x3f, 0x01, 0x02}
	cases := map[string]string{
		"standard padded":   base64.StdEncoding.EncodeToString(raw),
		"standard unpadded": base64.RawStdEncoding.EncodeToString(raw),
		"url padded":        base64.URLEncoding.EncodeToString(raw),
		"url unpadded":      base64.RawURLEncoding.EncodeToString(raw),
	}
	for name, encoded := range cases {
		got, err := DecodeBase64(encoded)
		if err != nil {
			t.Errorf("%s: DecodeBase64(%q) error: %v", name, encoded, err)
			continue
		}
		if string(got) != string(raw) {
			t.Errorf("%s: DecodeBase64(%q) = %x, want %x", name, encoded, got, raw)
		}
	}

	if _, err := DecodeBase64("!!!!not-base64!!!!"); err == nil {
		t.Error("DecodeBase64 accepted a non-base64 string")
	}
}

func TestLooksLikeBase64AcceptsUnpadded(t *testing.T) {
	for _, s := range []string{"YWVzLTI1Ni1nY206cGFzcw==", "YWVzLTI1Ni1nY206cGFzcw", "abc-def_ghi"} {
		if !LooksLikeBase64(s) {
			t.Errorf("LooksLikeBase64(%q) = false, want true", s)
		}
	}
	// len%4 == 1 can never be valid base64.
	if LooksLikeBase64("YWJjZ") {
		t.Error("LooksLikeBase64 accepted a len%4==1 string")
	}
	if LooksLikeBase64("not base64!") {
		t.Error("LooksLikeBase64 accepted invalid characters")
	}
}

func TestParseVMessAcceptsNumericPortAndAid(t *testing.T) {
	// "port":443 and "aid":0 as JSON numbers.
	payload := base64.StdEncoding.EncodeToString([]byte(
		`{"v":"2","ps":"n","add":"1.2.3.4","port":443,"id":"a0b1c2d3-e4f5-6789-abcd-ef0123456789","aid":0,"net":"tcp"}`))
	cfg, err := ParseSingle("vmess://" + payload)
	if err != nil {
		t.Fatalf("numeric port/aid rejected: %v", err)
	}
	if cfg.VMess == nil || cfg.VMess.Port != 443 || cfg.VMess.Server != "1.2.3.4" {
		t.Fatalf("unexpected config: %+v", cfg.VMess)
	}
}

func TestParseVMessRepairsGluedPayload(t *testing.T) {
	payload := base64.StdEncoding.EncodeToString([]byte(
		`{"v":"2","ps":"n","add":"1.2.3.4","port":"443","id":"a0b1c2d3-e4f5-6789-abcd-ef0123456789","aid":"0","net":"tcp"}`))
	// The defect seen in real feeds: the next entry's scheme word glued onto
	// the payload ("<base64>vless") instead of a newline.
	glued := payload + "vless"
	cfg, err := ParseSingle("vmess://" + glued)
	if err != nil {
		t.Fatalf("glued payload not repaired: %v", err)
	}
	if cfg.VMess == nil || cfg.VMess.Server != "1.2.3.4" {
		t.Fatalf("unexpected config: %+v", cfg.VMess)
	}

	// A payload that is genuinely broken must still fail.
	if _, err := ParseSingle("vmess://" + payload[:len(payload)-12]); err == nil {
		t.Error("truncated payload was accepted")
	}
}

func TestParseSSUnpaddedBase64Userinfo(t *testing.T) {
	cred := "aes-256-gcm:pass"
	unpadded := strings.TrimRight(base64.StdEncoding.EncodeToString([]byte(cred)), "=")
	if unpadded == "" || strings.Contains(unpadded, "=") {
		t.Fatalf("fixture is not unpadded: %q", unpadded)
	}
	cfg, err := ParseSingle("ss://" + unpadded + "@2.2.2.2:8388#SS")
	if err != nil {
		t.Fatalf("unpadded ss userinfo rejected: %v", err)
	}
	if cfg.SS == nil || cfg.SS.Method != "aes-256-gcm" || cfg.SS.Password != "pass" {
		t.Fatalf("unexpected ss config: %+v", cfg.SS)
	}
}
