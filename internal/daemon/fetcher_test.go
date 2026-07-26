package daemon

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/amirrezaalavi/Viberay/internal/models"
)

func TestLoadURLs(t *testing.T) {
	content := "https://sub1.example.com\nhttps://sub2.example.com\n"
	path := fetcherTempFile(t, content)
	defer os.Remove(path)

	urls, err := LoadURLs(path)
	if err != nil {
		t.Fatalf("LoadURLs: %v", err)
	}
	if len(urls) != 2 {
		t.Fatalf("got %d urls, want 2", len(urls))
	}
	if urls[0] != "https://sub1.example.com" {
		t.Errorf("urls[0] = %q", urls[0])
	}
}

func TestLoadURLsSkipsCommentsAndEmpty(t *testing.T) {
	content := "# this is a comment\n\nhttps://sub.example.com\n  \n"
	path := fetcherTempFile(t, content)
	defer os.Remove(path)

	urls, err := LoadURLs(path)
	if err != nil {
		t.Fatalf("LoadURLs: %v", err)
	}
	if len(urls) != 1 {
		t.Fatalf("got %d urls, want 1: %v", len(urls), urls)
	}
	if urls[0] != "https://sub.example.com" {
		t.Errorf("urls[0] = %q", urls[0])
	}
}

func TestLoadURLsMissingFile(t *testing.T) {
	_, err := LoadURLs("nonexistent.txt")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadURLsEmptyFile(t *testing.T) {
	path := fetcherTempFile(t, "")
	defer os.Remove(path)

	urls, err := LoadURLs(path)
	if err != nil {
		t.Fatalf("LoadURLs: %v", err)
	}
	if len(urls) != 0 {
		t.Errorf("got %d urls, want 0", len(urls))
	}
}

func TestFetchAndParse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("vmess://eyJhZGQiOiIxLjIuMy40IiwiYWlkIjoiMCIsImhvc3QiOiIiLCJpZCI6ImFiY2RlZmFiLTEyMzQtNTY3OC05YWJjLWRlZjAxMjM0NTY3OCIsIm5ldCI6InRjcCIsInBhdGgiOiIvIiwicG9ydCI6IjQ0MyIsInBzIjoiVGVzdC1WbWVzcyIsInRscyI6InRscyIsInR5cGUiOiJub25lIiwidiI6IjIifQ=="))
	}))
	defer server.Close()

	result, err := FetchAndParse([]string{server.URL}, 10*time.Second)
	if err != nil {
		t.Fatalf("FetchAndParse: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("got %d configs, want 1", len(result))
	}

	for hash, cfg := range result {
		if cfg.Protocol() != models.ProtocolVMess {
			t.Errorf("Protocol = %q, want vmess", cfg.Protocol())
		}
		if cfg.Raw == "" {
			t.Errorf("Raw should not be empty")
		}
		if len(hash) != 64 {
			t.Errorf("hash length = %d, want 64", len(hash))
		}
	}
}

func TestFetchAndParseDeduplicates(t *testing.T) {
	body := "vmess://eyJhZGQiOiIxLjIuMy40IiwiYWlkIjoiMCIsImhvc3QiOiIiLCJpZCI6ImFiY2RlZmFiLTEyMzQtNTY3OC05YWJjLWRlZjAxMjM0NTY3OCIsIm5ldCI6InRjcCIsInBhdGgiOiIvIiwicG9ydCI6IjQ0MyIsInBzIjoiVGVzdC1WbWVzcyIsInRscyI6InRscyIsInR5cGUiOiJub25lIiwidiI6IjIifQ=="
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body + "\n" + body))
	}))
	defer server.Close()

	result, err := FetchAndParse([]string{server.URL}, 10*time.Second)
	if err != nil {
		t.Fatalf("FetchAndParse: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("got %d configs, want 1 (deduplicated)", len(result))
	}
}

func TestFetchAndParseMultipleURLs(t *testing.T) {
	s1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("vmess://eyJhZGQiOiIxLjIuMy40IiwiYWlkIjoiMCIsImhvc3QiOiIiLCJpZCI6ImFiY2RlZmFiLTEyMzQtNTY3OC05YWJjLWRlZjAxMjM0NTY3OCIsIm5ldCI6InRjcCIsInBhdGgiOiIvIiwicG9ydCI6IjQ0MyIsInBzIjoiVGVzdC1WbWVzcyIsInRscyI6InRscyIsInR5cGUiOiJub25lIiwidiI6IjIifQ=="))
	}))
	defer s1.Close()

	s2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("trojan://password@5.6.7.8:443?security=tls#Test-Trojan"))
	}))
	defer s2.Close()

	result, err := FetchAndParse([]string{s1.URL, s2.URL}, 10*time.Second)
	if err != nil {
		t.Fatalf("FetchAndParse: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("got %d configs, want 2", len(result))
	}
}

func TestFetchAndParseBadURL(t *testing.T) {
	result, err := FetchAndParse([]string{"http://127.0.0.1:1"}, 1*time.Second)
	if err != nil {
		t.Fatalf("FetchAndParse: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("got %d configs, want 0", len(result))
	}
}

func TestMergeIntoStateAddsNew(t *testing.T) {
	s := NewState()
	configs := map[string]models.ProxyConfig{
		"hash1": {
			VMess: &models.VMessConfig{
				BaseConfig: models.BaseConfig{
					Server: "1.2.3.4",
					Port:   443,
					Protocol: models.ProtocolVMess,
				},
			},
			Raw: "vmess://...",
		},
	}

	MergeIntoState(s, configs, "https://sub.example.com")

	if len(s.Configs) != 1 {
		t.Fatalf("got %d configs, want 1", len(s.Configs))
	}

	entry := s.Configs["hash1"]
	if entry == nil {
		t.Fatal("missing hash1")
	}
	if entry.State != StateUnknown {
		t.Errorf("State = %q, want unknown", entry.State)
	}
	if entry.Host != "1.2.3.4" {
		t.Errorf("Host = %q, want 1.2.3.4", entry.Host)
	}
	if entry.Port != 443 {
		t.Errorf("Port = %d, want 443", entry.Port)
	}
	if entry.Protocol != "vmess" {
		t.Errorf("Protocol = %q, want vmess", entry.Protocol)
	}
	if entry.SourceURL != "https://sub.example.com" {
		t.Errorf("SourceURL = %q", entry.SourceURL)
	}
	if entry.Raw != "vmess://..." {
		t.Errorf("Raw = %q", entry.Raw)
	}
}

func TestMergeIntoStatePreservesExisting(t *testing.T) {
	s := NewState()
	s.Configs["hash1"] = &ConfigEntry{
		Raw:   "vmess://old",
		Host:  "1.2.3.4",
		Port:  443,
		State: StateWorking,
	}

	newConfigs := map[string]models.ProxyConfig{
		"hash1": {
			VMess: &models.VMessConfig{
				BaseConfig: models.BaseConfig{
					Server: "9.9.9.9",
					Port:   8080,
				},
			},
			Raw: "vmess://new",
		},
	}

	MergeIntoState(s, newConfigs, "https://new-sub.example.com")

	entry := s.Configs["hash1"]
	if entry.Host != "1.2.3.4" {
		t.Errorf("Host = %q, want 1.2.3.4 (preserved)", entry.Host)
	}
	if entry.State != StateWorking {
		t.Errorf("State = %q, want working (preserved)", entry.State)
	}
	if entry.Raw != "vmess://old" {
		t.Errorf("Raw = %q, want vmess://old (preserved)", entry.Raw)
	}
}

func TestMergeIntoStateMultiple(t *testing.T) {
	s := NewState()
	s.Configs["existing"] = &ConfigEntry{Raw: "keep", State: StateWorking}

	configs := map[string]models.ProxyConfig{
		"new1": {
			VMess: &models.VMessConfig{
				BaseConfig: models.BaseConfig{Server: "1.2.3.4", Port: 443},
			},
			Raw: "vmess://new1",
		},
		"new2": {
			VLess: &models.VLessConfig{
				BaseConfig: models.BaseConfig{Server: "5.6.7.8", Port: 80},
			},
			Raw: "vless://new2",
		},
	}

	MergeIntoState(s, configs, "https://sub.example.com")

	if len(s.Configs) != 3 {
		t.Fatalf("got %d configs, want 3", len(s.Configs))
	}
	if s.Configs["existing"].State != StateWorking {
		t.Errorf("existing State = %q, want working", s.Configs["existing"].State)
	}
	if s.Configs["new1"].State != StateUnknown {
		t.Errorf("new1 State = %q, want unknown", s.Configs["new1"].State)
	}
	if s.Configs["new2"].Protocol != "vless" {
		t.Errorf("new2 Protocol = %q, want vless", s.Configs["new2"].Protocol)
	}
}

func TestSHA256Of(t *testing.T) {
	h1 := sha256Of("hello")
	h2 := sha256Of("hello")
	h3 := sha256Of("world")

	if h1 != h2 {
		t.Error("same input should produce same hash")
	}
	if h1 == h3 {
		t.Error("different input should produce different hash")
	}
	if len(h1) != 64 {
		t.Errorf("hash length = %d, want 64", len(h1))
	}
}

func fetcherTempFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "urls.txt")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	return path
}
