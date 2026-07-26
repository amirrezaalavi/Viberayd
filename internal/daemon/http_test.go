package daemon

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHandleSub(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "working.txt")
	os.WriteFile(outputPath, []byte("vless://a@b:443#A 100ms\ntrojan://p@c:443#B 50ms\n"), 0644)

	cfg := DefaultConfig()
	cfg.Daemon.OutputFile = outputPath
	cfg.HTTP.SubPath = "/sub"
	d := &Daemon{Config: &cfg, State: NewState()}

	hs := &httpServer{daemon: d}
	req := httptest.NewRequest(http.MethodGet, "/sub", nil)
	w := httptest.NewRecorder()
	hs.handleSub(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	decoded, err := decodeBase64Str(w.Body.String())
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !strings.Contains(decoded, "vless://a@b:443#A") {
		t.Errorf("missing vless config in: %s", decoded)
	}
}

func TestHandleSubNotFound(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Daemon.OutputFile = "/nonexistent/working.txt"
	d := &Daemon{Config: &cfg, State: NewState()}

	hs := &httpServer{daemon: d}
	req := httptest.NewRequest(http.MethodGet, "/sub", nil)
	w := httptest.NewRecorder()
	hs.handleSub(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandleSubMethodNotAllowed(t *testing.T) {
	cfg := DefaultConfig()
	d := &Daemon{Config: &cfg, State: NewState()}
	hs := &httpServer{daemon: d}

	req := httptest.NewRequest(http.MethodPost, "/sub", nil)
	w := httptest.NewRecorder()
	hs.handleSub(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", w.Code)
	}
}

func TestHandleHealth(t *testing.T) {
	hs := &httpServer{daemon: &Daemon{State: NewState()}}
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	hs.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("status = %q, want ok", resp["status"])
	}
}

func TestHandleURLsList(t *testing.T) {
	dir := t.TempDir()
	urlsPath := filepath.Join(dir, "urls.txt")
	os.WriteFile(urlsPath, []byte("https://sub1.example.com\nhttps://sub2.example.com\n"), 0644)

	cfg := DefaultConfig()
	cfg.Daemon.URLsFile = urlsPath
	hs := &httpServer{daemon: &Daemon{Config: &cfg}}

	req := httptest.NewRequest(http.MethodGet, "/api/urls", nil)
	w := httptest.NewRecorder()
	hs.handleURLs(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var urls []string
	json.NewDecoder(w.Body).Decode(&urls)
	if len(urls) != 2 {
		t.Fatalf("got %d urls, want 2", len(urls))
	}
	if urls[0] != "https://sub1.example.com" {
		t.Errorf("urls[0] = %q", urls[0])
	}
}

func TestHandleURLsAdd(t *testing.T) {
	dir := t.TempDir()
	urlsPath := filepath.Join(dir, "urls.txt")
	os.WriteFile(urlsPath, []byte("https://old.example.com\n"), 0644)

	cfg := DefaultConfig()
	cfg.Daemon.URLsFile = urlsPath
	hs := &httpServer{daemon: &Daemon{Config: &cfg}}

	body := bytes.NewReader([]byte(`{"url": "https://new.example.com"}`))
	req := httptest.NewRequest(http.MethodPost, "/api/urls", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	hs.handleURLs(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", w.Code)
	}

	data, _ := os.ReadFile(urlsPath)
	content := string(data)
	if !strings.Contains(content, "https://new.example.com") {
		t.Errorf("missing new url in %s", content)
	}
}

func TestHandleURLsAddEmpty(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig()
	cfg.Daemon.URLsFile = filepath.Join(dir, "urls.txt")
	hs := &httpServer{daemon: &Daemon{Config: &cfg}}
	body := bytes.NewReader([]byte(`{"url": ""}`))
	req := httptest.NewRequest(http.MethodPost, "/api/urls", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	hs.handleURLs(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandleURLsDelete(t *testing.T) {
	dir := t.TempDir()
	urlsPath := filepath.Join(dir, "urls.txt")
	os.WriteFile(urlsPath, []byte("https://a.example.com\nhttps://b.example.com\nhttps://c.example.com\n"), 0644)

	cfg := DefaultConfig()
	cfg.Daemon.URLsFile = urlsPath
	hs := &httpServer{daemon: &Daemon{Config: &cfg}}

	req := httptest.NewRequest(http.MethodDelete, "/api/urls/2", nil)
	w := httptest.NewRecorder()
	hs.handleURLsByID(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	data, _ := os.ReadFile(urlsPath)
	if strings.Contains(string(data), "https://b.example.com") {
		t.Errorf("b.example.com should have been removed")
	}
	if !strings.Contains(string(data), "https://a.example.com") {
		t.Errorf("a.example.com should remain")
	}
	if !strings.Contains(string(data), "https://c.example.com") {
		t.Errorf("c.example.com should remain")
	}
}

func TestHandleStats(t *testing.T) {
	s := NewState()
	s.Configs["a"] = &ConfigEntry{State: StateWorking}
	s.Configs["b"] = &ConfigEntry{State: StateWorking}
	s.Configs["c"] = &ConfigEntry{State: StateFailed}
	s.Configs["d"] = &ConfigEntry{State: StateUnknown}
	s.Configs["e"] = &ConfigEntry{State: StateUnreachable}

	d := &Daemon{State: s}
	hs := &httpServer{daemon: d}

	req := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	w := httptest.NewRecorder()
	hs.handleStats(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var stats map[string]interface{}
	json.NewDecoder(w.Body).Decode(&stats)

	if stats["total"].(float64) != 5 {
		t.Errorf("total = %v, want 5", stats["total"])
	}
	if stats["working"].(float64) != 2 {
		t.Errorf("working = %v, want 2", stats["working"])
	}
	if stats["failed"].(float64) != 1 {
		t.Errorf("failed = %v, want 1", stats["failed"])
	}
	if stats["unreachable"].(float64) != 1 {
		t.Errorf("unreachable = %v, want 1", stats["unreachable"])
	}
}

func TestHandleCycleTrigger(t *testing.T) {
	d := &Daemon{State: NewState(), trigger: make(chan struct{}, 1)}
	hs := &httpServer{daemon: d}

	req := httptest.NewRequest(http.MethodPost, "/api/cycle/trigger", nil)
	w := httptest.NewRecorder()
	hs.handleCycleTrigger(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	select {
	case <-d.trigger:
	default:
		t.Error("trigger was not signaled")
	}
}

func TestHandleCycleTriggerMethodNotAllowed(t *testing.T) {
	hs := &httpServer{daemon: &Daemon{State: NewState()}}
	req := httptest.NewRequest(http.MethodGet, "/api/cycle/trigger", nil)
	w := httptest.NewRecorder()
	hs.handleCycleTrigger(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", w.Code)
	}
}

func TestHandleConfigs(t *testing.T) {
	s := NewState()
	for i := 0; i < 10; i++ {
		h := sha256Of(string(rune('a' + i)))
		s.Configs[h] = &ConfigEntry{Raw: "test", State: StateWorking}
	}

	hs := &httpServer{daemon: &Daemon{State: s}}

	req := httptest.NewRequest(http.MethodGet, "/api/configs?page=1&per_page=5", nil)
	w := httptest.NewRecorder()
	hs.handleConfigs(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp struct {
		Page    int            `json:"page"`
		PerPage int            `json:"per_page"`
		Total   int            `json:"total"`
		Configs []*ConfigEntry `json:"configs"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Page != 1 {
		t.Errorf("page = %d, want 1", resp.Page)
	}
	if resp.PerPage != 5 {
		t.Errorf("per_page = %d, want 5", resp.PerPage)
	}
	if resp.Total != 10 {
		t.Errorf("total = %d, want 10", resp.Total)
	}
	if len(resp.Configs) != 5 {
		t.Errorf("configs = %d, want 5", len(resp.Configs))
	}
}

func TestHandleConfigsInvalidPage(t *testing.T) {
	hs := &httpServer{daemon: &Daemon{State: NewState()}}
	req := httptest.NewRequest(http.MethodGet, "/api/configs?page=-1&per_page=200", nil)
	w := httptest.NewRecorder()
	hs.handleConfigs(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp struct {
		Page    int            `json:"page"`
		PerPage int            `json:"per_page"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Page != 1 {
		t.Errorf("page = %d, want 1", resp.Page)
	}
	if resp.PerPage != 50 {
		t.Errorf("per_page = %d, want 50 (max clamped)", resp.PerPage)
	}
}

func TestStartStopHTTPServers(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig()
	cfg.HTTP.Enabled = true
	cfg.HTTP.Port = 0
	cfg.HTTP.APIPort = 0
	cfg.Daemon.OutputFile = filepath.Join(dir, "working.txt")

	d := &Daemon{
		Config:  &cfg,
		State:   NewState(),
		trigger: make(chan struct{}, 1),
	}

	err := d.StartHTTPServers()
	if err != nil {
		t.Fatalf("StartHTTPServers: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := d.StopHTTPServers(ctx); err != nil {
		t.Fatalf("StopHTTPServers: %v", err)
	}
}

func decodeBase64Str(s string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}
