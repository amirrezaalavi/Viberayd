package daemon

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/amirrezaalavi/Viberayd/internal/models"
)

func TestCountByState(t *testing.T) {
	s := NewState()
	s.Configs["a"] = &ConfigEntry{State: StateWorking}
	s.Configs["b"] = &ConfigEntry{State: StateWorking}
	s.Configs["c"] = &ConfigEntry{State: StateFailed}
	s.Configs["d"] = &ConfigEntry{State: StateUnknown}

	if n := countByState(s, StateWorking); n != 2 {
		t.Errorf("working count = %d, want 2", n)
	}
	if n := countByState(s, StateFailed); n != 1 {
		t.Errorf("failed count = %d, want 1", n)
	}
	if n := countByState(s, StateUnknown); n != 1 {
		t.Errorf("unknown count = %d, want 1", n)
	}
}

func TestCountByStateEmpty(t *testing.T) {
	s := NewState()
	if n := countByState(s, StateWorking); n != 0 {
		t.Errorf("got %d, want 0", n)
	}
}

func TestWriteOutputFile(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "working.txt")

	d := &Daemon{
		Config: &Config{
			Daemon: DaemonConfig{
				OutputFile: outputPath,
			},
		},
		State: NewState(),
	}

	d.State.Configs["a"] = &ConfigEntry{
		Raw:       "vless://a@host:443?encryption=none#A",
		State:     StateWorking,
		LatencyMs: 124,
	}
	d.State.Configs["b"] = &ConfigEntry{
		Raw:       "vmess://base64stuff",
		State:     StateWorking,
		LatencyMs: 55,
	}
	d.State.Configs["c"] = &ConfigEntry{
		Raw:   "trojan://p@host2:443?security=tls#C",
		State: StateFailed,
	}

	d.writeOutputFile()

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	lines := string(data)
	if contains(lines, "trojan://") {
		t.Errorf("output should not contain failed config")
	}
	if !contains(lines, "124ms") {
		t.Errorf("output should include latency: %s", lines)
	}
	if !contains(lines, "vless://a@host:443") {
		t.Errorf("missing vless config")
	}
	if !contains(lines, "vmess://base64stuff 55ms") {
		t.Errorf("missing vmess config with latency: %s", lines)
	}
}

func TestWriteOutputFileNoLatency(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "working.txt")

	d := &Daemon{
		Config: &Config{
			Daemon: DaemonConfig{
				OutputFile: outputPath,
			},
		},
		State: NewState(),
	}

	d.State.Configs["a"] = &ConfigEntry{
		Raw:   "vless://x@y:443#test",
		State: StateWorking,
	}

	d.writeOutputFile()

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	if contains(string(data), "0ms") {
		t.Errorf("should not write 0ms latency")
	}
	if string(data) != "vless://x@y:443#test\n" {
		t.Errorf("unexpected output: %q", string(data))
	}
}

func TestWriteOutputFileEmptyOutput(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "working.txt")

	d := &Daemon{
		Config: &Config{
			Daemon: DaemonConfig{
				OutputFile: outputPath,
			},
		},
		State: NewState(),
	}

	d.State.Configs["a"] = &ConfigEntry{
		Raw:   "vless://a@b:443#test",
		State: StateFailed,
	}

	d.writeOutputFile()

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	if len(data) != 0 {
		t.Errorf("expected empty file, got %q", string(data))
	}
}

func TestWriteOutputFileEmptyPath(t *testing.T) {
	d := &Daemon{
		Config: &Config{
			Daemon: DaemonConfig{},
		},
		State: NewState(),
	}
	d.State.Configs["a"] = &ConfigEntry{Raw: "test", State: StateWorking}

	d.writeOutputFile() // should not panic
}

func TestMergeWithParsed(t *testing.T) {
	d := &Daemon{
		State:   NewState(),
		configs: make(map[string]models.ProxyConfig),
	}

	cfg := models.ProxyConfig{
		VMess: &models.VMessConfig{
			BaseConfig: models.BaseConfig{
				Server:   "1.2.3.4",
				Port:     443,
				Protocol: models.ProtocolVMess,
			},
		},
		Raw: "vmess://abc123",
	}
	hash := sha256Of(cfg.Raw)

	d.mergeWithParsed(map[string]models.ProxyConfig{hash: cfg}, "https://sub.example.com")

	if len(d.State.Configs) != 1 {
		t.Fatalf("got %d state configs, want 1", len(d.State.Configs))
	}
	if d.State.Configs[hash].State != StateUnknown {
		t.Errorf("state = %q, want unknown", d.State.Configs[hash].State)
	}
	if _, ok := d.configs[hash]; !ok {
		t.Errorf("parsed config not cached")
	}
}

func TestMergeWithParsedPreservesExisting(t *testing.T) {
	d := &Daemon{
		State:   NewState(),
		configs: make(map[string]models.ProxyConfig),
	}

	cfg := models.ProxyConfig{
		VMess: &models.VMessConfig{
			BaseConfig: models.BaseConfig{Server: "1.2.3.4", Port: 443},
		},
		Raw: "vmess://abc",
	}
	hash := sha256Of(cfg.Raw)

	d.State.Configs[hash] = &ConfigEntry{
		Raw:       "vmess://abc",
		State:     StateWorking,
		LatencyMs: 100,
	}
	d.configs[hash] = cfg

	newCfg := models.ProxyConfig{
		VMess: &models.VMessConfig{
			BaseConfig: models.BaseConfig{Server: "9.9.9.9", Port: 8080},
		},
		Raw: "vmess://abc",
	}

	d.mergeWithParsed(map[string]models.ProxyConfig{hash: newCfg}, "https://new.example.com")

	if d.State.Configs[hash].State != StateWorking {
		t.Errorf("state changed to %q, want working", d.State.Configs[hash].State)
	}
	if d.State.Configs[hash].LatencyMs != 100 {
		t.Errorf("latency changed to %d, want 100", d.State.Configs[hash].LatencyMs)
	}
}

func TestShutdown(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	outputPath := filepath.Join(dir, "working.txt")

	cfg := DefaultConfig()
	cfg.Daemon.StateFile = statePath
	cfg.Daemon.OutputFile = outputPath

	d := NewDaemon(&cfg)
	d.State.Configs["h1"] = &ConfigEntry{
		Raw:   "vless://a@b:443#A",
		State: StateWorking,
		LatencyMs: 50,
	}

	if err := d.shutdown(); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Errorf("state file not saved: %v", err)
	}
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("output file not saved: %v", err)
	}
}

func TestRunCycleNoURLs(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	outputPath := filepath.Join(dir, "working.txt")
	urlsPath := filepath.Join(dir, "urls.txt")

	os.WriteFile(urlsPath, []byte(""), 0644)

	cfg := DefaultConfig()
	cfg.Daemon.StateFile = statePath
	cfg.Daemon.OutputFile = outputPath
	cfg.Daemon.URLsFile = urlsPath
	cfg.Daemon.Parallel = 5
	cfg.Daemon.TimeoutSec = 5
	cfg.Daemon.Depth = "quick"

	d := NewDaemon(&cfg)

	err := d.runCycle(context.Background())
	if err != nil {
		t.Fatalf("RunCycle: %v", err)
	}

	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Errorf("state file not created")
	}
}

func TestRunCycleHandlesFetchFailure(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	outputPath := filepath.Join(dir, "working.txt")
	urlsPath := filepath.Join(dir, "urls.txt")

	os.WriteFile(urlsPath, []byte("http://127.0.0.1:1/sub\n"), 0644)

	cfg := DefaultConfig()
	cfg.Daemon.StateFile = statePath
	cfg.Daemon.OutputFile = outputPath
	cfg.Daemon.URLsFile = urlsPath
	cfg.Daemon.Parallel = 5
	cfg.Daemon.TimeoutSec = 1
	cfg.Daemon.Depth = "quick"
	cfg.Daemon.CycleSleepSec = 60
	cfg.Daemon.RetestIntervalSec = 3600

	d := NewDaemon(&cfg)

	err := d.runCycle(context.Background())
	if err != nil {
		t.Fatalf("RunCycle: %v", err)
	}

	stateData, _ := os.ReadFile(statePath)
	if len(stateData) == 0 {
		t.Error("state file is empty")
	}
}

func TestNewDaemonLoadsState(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")

	s := NewState()
	s.Configs["h1"] = &ConfigEntry{Raw: "test", State: StateWorking}
	SaveState(statePath, s)

	cfg := DefaultConfig()
	cfg.Daemon.StateFile = statePath

	d := NewDaemon(&cfg)
	if len(d.State.Configs) != 1 {
		t.Errorf("got %d configs, want 1", len(d.State.Configs))
	}
	if d.State.Configs["h1"].State != StateWorking {
		t.Errorf("state = %q, want working", d.State.Configs["h1"].State)
	}
}

func TestNewDaemonMissingState(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Daemon.StateFile = "/nonexistent/state.json"

	d := NewDaemon(&cfg)
	if len(d.State.Configs) != 0 {
		t.Errorf("got %d configs, want 0", len(d.State.Configs))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
