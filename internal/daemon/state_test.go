package daemon

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewState(t *testing.T) {
	s := NewState()
	if s.Version != 1 {
		t.Errorf("Version = %d, want 1", s.Version)
	}
	if len(s.Configs) != 0 {
		t.Errorf("Configs = %d, want 0", len(s.Configs))
	}
}

func TestLoadStateMissingFile(t *testing.T) {
	s, err := LoadState("nonexistent.json")
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if s == nil {
		t.Fatal("LoadState returned nil")
	}
	if s.Version != 1 {
		t.Errorf("Version = %d, want 1", s.Version)
	}
}

func TestLoadStateValid(t *testing.T) {
	content := `{
		"version": 1,
		"updated_at": "2026-07-25T12:00:00Z",
		"configs": {
			"abc123": {
				"raw": "vmess://...",
				"host": "1.2.3.4",
				"port": 443,
				"protocol": "vmess",
				"source_url": "https://sub.example.com/list",
				"first_seen": "2026-07-20T10:00:00Z",
				"last_tested": "2026-07-25T11:55:00Z",
				"last_success": "2026-07-25T11:55:00Z",
				"success_count": 42,
				"fail_count": 3,
				"state": "working",
				"latency_ms": 124
			}
		}
	}`
	path := stateTempFile(t, content)
	defer os.Remove(path)

	s, err := LoadState(path)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}

	entry, ok := s.Configs["abc123"]
	if !ok {
		t.Fatal("expected config abc123")
	}
	if entry.Raw != "vmess://..." {
		t.Errorf("Raw = %q, want vmess://...", entry.Raw)
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
	if entry.State != StateWorking {
		t.Errorf("State = %q, want working", entry.State)
	}
	if entry.SuccessCount != 42 {
		t.Errorf("SuccessCount = %d, want 42", entry.SuccessCount)
	}
	if entry.LatencyMs != 124 {
		t.Errorf("LatencyMs = %d, want 124", entry.LatencyMs)
	}
}

func TestLoadStateInvalidJSON(t *testing.T) {
	path := stateTempFile(t, "{invalid")
	defer os.Remove(path)

	_, err := LoadState(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	s := NewState()
	s.Configs["hash1"] = &ConfigEntry{
		Raw:          "vless://a@b:443?encryption=none#test",
		Host:         "b",
		Port:         443,
		Protocol:     "vless",
		SourceURL:    "https://sub.example.com",
		FirstSeen:    time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC),
		LastTested:   time.Date(2026, 7, 25, 11, 55, 0, 0, time.UTC),
		LastSuccess:  time.Date(2026, 7, 25, 11, 55, 0, 0, time.UTC),
		SuccessCount: 10,
		FailCount:    1,
		State:        StateWorking,
		LatencyMs:    55,
	}

	if err := SaveState(path, s); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	loaded, err := LoadState(path)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}

	if len(loaded.Configs) != 1 {
		t.Fatalf("got %d configs, want 1", len(loaded.Configs))
	}

	entry := loaded.Configs["hash1"]
	if entry == nil {
		t.Fatal("missing hash1")
	}
	if entry.Raw != "vless://a@b:443?encryption=none#test" {
		t.Errorf("Raw = %q", entry.Raw)
	}
	if entry.State != StateWorking {
		t.Errorf("State = %q", entry.State)
	}
	if entry.SuccessCount != 10 {
		t.Errorf("SuccessCount = %d, want 10", entry.SuccessCount)
	}
	if entry.LatencyMs != 55 {
		t.Errorf("LatencyMs = %d, want 55", entry.LatencyMs)
	}
}

func TestSelectCandidates(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	retestInterval := 30 * time.Minute

	s := NewState()
	s.Configs["u"] = &ConfigEntry{State: StateUnknown}
	s.Configs["w_due"] = &ConfigEntry{State: StateWorking, LastTested: now.Add(-1 * time.Hour)}
	s.Configs["w_fresh"] = &ConfigEntry{State: StateWorking, LastTested: now.Add(-5 * time.Minute)}
	s.Configs["f"] = &ConfigEntry{State: StateFailed}
	s.Configs["x"] = &ConfigEntry{State: StateUnreachable}

	t.Run("keep_successful=true", func(t *testing.T) {
		candidates := SelectCandidates(s, retestInterval, true, now)
		want := map[string]bool{"u": true, "w_due": true, "f": true, "x": true}
		if len(candidates) != len(want) {
			t.Fatalf("got %d candidates, want %d: %v", len(candidates), len(want), candidates)
		}
		for _, h := range candidates {
			if !want[h] {
				t.Errorf("unexpected candidate: %s", h)
			}
		}
	})

	t.Run("keep_successful=false", func(t *testing.T) {
		candidates := SelectCandidates(s, retestInterval, false, now)
		want := map[string]bool{"u": true, "f": true, "x": true}
		if len(candidates) != len(want) {
			t.Fatalf("got %d candidates, want %d: %v", len(candidates), len(want), candidates)
		}
		for _, h := range candidates {
			if !want[h] {
				t.Errorf("unexpected candidate: %s", h)
			}
		}
	})
}

func TestSelectCandidatesEmptyState(t *testing.T) {
	s := NewState()
	candidates := SelectCandidates(s, 30*time.Minute, true, time.Now())
	if len(candidates) != 0 {
		t.Errorf("got %d candidates, want 0", len(candidates))
	}
}

func TestApplyResultsUnknownToWorking(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)

	s := NewState()
	s.Configs["h"] = &ConfigEntry{State: StateUnknown, FailCount: 0, SuccessCount: 0}

	ApplyResults(s, []TestResult{{Hash: "h", Success: true, LatencyMs: 100}}, now)

	entry := s.Configs["h"]
	if entry.State != StateWorking {
		t.Errorf("State = %q, want working", entry.State)
	}
	if entry.SuccessCount != 1 {
		t.Errorf("SuccessCount = %d, want 1", entry.SuccessCount)
	}
	if entry.LatencyMs != 100 {
		t.Errorf("LatencyMs = %d, want 100", entry.LatencyMs)
	}
	if !entry.LastSuccess.Equal(now) {
		t.Errorf("LastSuccess = %v, want %v", entry.LastSuccess, now)
	}
	if !entry.LastTested.Equal(now) {
		t.Errorf("LastTested = %v, want %v", entry.LastTested, now)
	}
}

func TestApplyResultsUnknownToFailed(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)

	s := NewState()
	s.Configs["h"] = &ConfigEntry{State: StateUnknown}

	ApplyResults(s, []TestResult{{Hash: "h", Success: false}}, now)

	entry := s.Configs["h"]
	if entry.State != StateFailed {
		t.Errorf("State = %q, want failed", entry.State)
	}
	if entry.FailCount != 1 {
		t.Errorf("FailCount = %d, want 1", entry.FailCount)
	}
}

func TestApplyResultsWorkingToFailed(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)

	s := NewState()
	s.Configs["h"] = &ConfigEntry{State: StateWorking, SuccessCount: 5, FailCount: 0}

	ApplyResults(s, []TestResult{{Hash: "h", Success: false}}, now)

	entry := s.Configs["h"]
	if entry.State != StateFailed {
		t.Errorf("State = %q, want failed", entry.State)
	}
	if entry.FailCount != 1 {
		t.Errorf("FailCount = %d, want 1", entry.FailCount)
	}
	if entry.SuccessCount != 5 {
		t.Errorf("SuccessCount = %d, want 5 (unchanged)", entry.SuccessCount)
	}
}

func TestApplyResultsWorkingToWorking(t *testing.T) {
	now := time.Date(2026, 7, 25, 13, 0, 0, 0, time.UTC)

	s := NewState()
	s.Configs["h"] = &ConfigEntry{State: StateWorking, SuccessCount: 5, LatencyMs: 200}

	ApplyResults(s, []TestResult{{Hash: "h", Success: true, LatencyMs: 50}}, now)

	entry := s.Configs["h"]
	if entry.State != StateWorking {
		t.Errorf("State = %q, want working", entry.State)
	}
	if entry.SuccessCount != 6 {
		t.Errorf("SuccessCount = %d, want 6", entry.SuccessCount)
	}
	if entry.LatencyMs != 50 {
		t.Errorf("LatencyMs = %d, want 50", entry.LatencyMs)
	}
}

func TestApplyResultsMissingHash(t *testing.T) {
	s := NewState()
	// Should not panic or add entry
	ApplyResults(s, []TestResult{{Hash: "nonexistent", Success: true}}, time.Now())
	if len(s.Configs) != 0 {
		t.Errorf("Configs = %d, want 0", len(s.Configs))
	}
}

func TestApplyResultsMultiple(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)

	s := NewState()
	s.Configs["a"] = &ConfigEntry{State: StateUnknown}
	s.Configs["b"] = &ConfigEntry{State: StateUnknown}
	s.Configs["c"] = &ConfigEntry{State: StateWorking, SuccessCount: 3}

	results := []TestResult{
		{Hash: "a", Success: true, LatencyMs: 10},
		{Hash: "b", Success: false},
		{Hash: "c", Success: true, LatencyMs: 30},
	}
	ApplyResults(s, results, now)

	if s.Configs["a"].State != StateWorking {
		t.Errorf("a.State = %q, want working", s.Configs["a"].State)
	}
	if s.Configs["b"].State != StateFailed {
		t.Errorf("b.State = %q, want failed", s.Configs["b"].State)
	}
	if s.Configs["c"].State != StateWorking {
		t.Errorf("c.State = %q, want working", s.Configs["c"].State)
	}
	if s.Configs["c"].SuccessCount != 4 {
		t.Errorf("c.SuccessCount = %d, want 4", s.Configs["c"].SuccessCount)
	}
}

func TestLoadStateNilConfigs(t *testing.T) {
	content := `{"version": 1, "updated_at": "2026-07-25T12:00:00Z"}`
	path := stateTempFile(t, content)
	defer os.Remove(path)

	s, err := LoadState(path)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if s.Configs == nil {
		t.Fatal("Configs should not be nil")
	}
	if len(s.Configs) != 0 {
		t.Errorf("Configs = %d, want 0", len(s.Configs))
	}
}

func stateTempFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	return path
}
