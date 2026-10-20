package main

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSplitLines(t *testing.T) {
	body := "\r\nvless://a@b:443#A\r\n# comment\n\n  trojan://c@d:443#B  \n"
	got := splitLines(body)
	want := []string{"vless://a@b:443#A", "trojan://c@d:443#B"}
	if len(got) != len(want) {
		t.Fatalf("splitLines() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("splitLines()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestLatencySuffixDetection(t *testing.T) {
	cases := []struct {
		line string
		want bool
	}{
		{"vless://a@b:443#A 123ms", true},
		{"vmess://eyJhIjoxfQ== 55ms", true},
		{"vless://a@b:443#A", false},
		{"vless://a@b:443#name-with-123ms-inside", false},
		{"trojan://p@h:443?security=tls#123ms", false}, // no separating space
	}
	for _, tc := range cases {
		if got := latencySuffix.MatchString(tc.line); got != tc.want {
			t.Errorf("latencySuffix(%q) = %v, want %v", tc.line, got, tc.want)
		}
	}
}

func TestLoadPlainAndBase64(t *testing.T) {
	dir := t.TempDir()

	plain := "vless://a@b:443#A\n"
	plainPath := filepath.Join(dir, "plain.txt")
	if err := os.WriteFile(plainPath, []byte(plain), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := load(plainPath, time.Second)
	if err != nil {
		t.Fatalf("load plain: %v", err)
	}
	if got != plain {
		t.Errorf("load plain = %q, want %q", got, plain)
	}

	b64Path := filepath.Join(dir, "sub.txt")
	encoded := base64.StdEncoding.EncodeToString([]byte(plain))
	if err := os.WriteFile(b64Path, []byte(encoded), 0644); err != nil {
		t.Fatal(err)
	}
	got, err = load(b64Path, time.Second)
	if err != nil {
		t.Fatalf("load base64: %v", err)
	}
	if got != plain {
		t.Errorf("load base64 = %q, want %q", got, plain)
	}
}

func TestSummarize(t *testing.T) {
	results := []lineResult{
		{RawOK: true},
		{RawOK: true, LatencyMs: true},
		{RawOK: false, ParseErr: "boom"},
		{RawOK: true, E2E: true, E2EOK: true},
		{RawOK: true, E2E: true, E2EOK: false},
	}
	malformed, e2eFailed := summarize(results)
	if malformed != 1 {
		t.Errorf("malformed = %d, want 1", malformed)
	}
	if e2eFailed != 1 {
		t.Errorf("e2eFailed = %d, want 1", e2eFailed)
	}
	if n := countLatency(results); n != 1 {
		t.Errorf("countLatency = %d, want 1", n)
	}
	tested, passed := countE2E(results)
	if tested != 2 || passed != 1 {
		t.Errorf("countE2E = (%d, %d), want (2, 1)", tested, passed)
	}
}
