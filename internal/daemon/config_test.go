package daemon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Version != 1 {
		t.Errorf("Version = %d, want 1", cfg.Version)
	}
	if cfg.Daemon.URLsFile != "urls.txt" {
		t.Errorf("URLsFile = %q, want urls.txt", cfg.Daemon.URLsFile)
	}
	if cfg.Daemon.OutputFile != "working.txt" {
		t.Errorf("OutputFile = %q, want working.txt", cfg.Daemon.OutputFile)
	}
	if cfg.Daemon.StateFile != "state.json" {
		t.Errorf("StateFile = %q, want state.json", cfg.Daemon.StateFile)
	}
	if cfg.Daemon.CycleSleepSec != 300 {
		t.Errorf("CycleSleepSec = %d, want 300", cfg.Daemon.CycleSleepSec)
	}
	if cfg.Daemon.Parallel != 10 {
		t.Errorf("Parallel = %d, want 10", cfg.Daemon.Parallel)
	}
	if cfg.Daemon.TimeoutSec != 10 {
		t.Errorf("TimeoutSec = %d, want 10", cfg.Daemon.TimeoutSec)
	}
	if cfg.Daemon.Depth != "standard" {
		t.Errorf("Depth = %q, want standard", cfg.Daemon.Depth)
	}
	if cfg.Daemon.KeepSuccessful != true {
		t.Errorf("KeepSuccessful = %v, want true", cfg.Daemon.KeepSuccessful)
	}
	if cfg.Daemon.RetestIntervalSec != 1800 {
		t.Errorf("RetestIntervalSec = %d, want 1800", cfg.Daemon.RetestIntervalSec)
	}
	if cfg.HTTP.Enabled != false {
		t.Errorf("HTTP.Enabled = %v, want false", cfg.HTTP.Enabled)
	}
	if cfg.HTTP.Port != 8080 {
		t.Errorf("HTTP.Port = %d, want 8080", cfg.HTTP.Port)
	}
	if cfg.HTTP.SubPath != "/sub" {
		t.Errorf("HTTP.SubPath = %q, want /sub", cfg.HTTP.SubPath)
	}
	if cfg.HTTP.APIPort != 8081 {
		t.Errorf("HTTP.APIPort = %d, want 8081", cfg.HTTP.APIPort)
	}
}

func TestLoadConfigFull(t *testing.T) {
	content := `
version = 1

[daemon]
urls_file = "my_urls.txt"
output_file = "my_working.txt"
state_file = "my_state.json"
cycle_sleep = 60
parallel = 5
timeout = 15
depth = "comprehensive"
keep_successful = false
retest_interval = 3600

[http]
enabled = true
port = 9090
sub_path = "/my_sub"
api_port = 9091
`
	path := writeTempFile(t, content)
	defer os.Remove(path)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.Daemon.URLsFile != "my_urls.txt" {
		t.Errorf("URLsFile = %q, want my_urls.txt", cfg.Daemon.URLsFile)
	}
	if cfg.Daemon.OutputFile != "my_working.txt" {
		t.Errorf("OutputFile = %q, want my_working.txt", cfg.Daemon.OutputFile)
	}
	if cfg.Daemon.StateFile != "my_state.json" {
		t.Errorf("StateFile = %q, want my_state.json", cfg.Daemon.StateFile)
	}
	if cfg.Daemon.CycleSleepSec != 60 {
		t.Errorf("CycleSleepSec = %d, want 60", cfg.Daemon.CycleSleepSec)
	}
	if cfg.Daemon.Parallel != 5 {
		t.Errorf("Parallel = %d, want 5", cfg.Daemon.Parallel)
	}
	if cfg.Daemon.TimeoutSec != 15 {
		t.Errorf("TimeoutSec = %d, want 15", cfg.Daemon.TimeoutSec)
	}
	if cfg.Daemon.Depth != "comprehensive" {
		t.Errorf("Depth = %q, want comprehensive", cfg.Daemon.Depth)
	}
	if cfg.Daemon.KeepSuccessful != false {
		t.Errorf("KeepSuccessful = %v, want false", cfg.Daemon.KeepSuccessful)
	}
	if cfg.Daemon.RetestIntervalSec != 3600 {
		t.Errorf("RetestIntervalSec = %d, want 3600", cfg.Daemon.RetestIntervalSec)
	}
	if cfg.HTTP.Enabled != true {
		t.Errorf("HTTP.Enabled = %v, want true", cfg.HTTP.Enabled)
	}
	if cfg.HTTP.Port != 9090 {
		t.Errorf("HTTP.Port = %d, want 9090", cfg.HTTP.Port)
	}
	if cfg.HTTP.SubPath != "/my_sub" {
		t.Errorf("HTTP.SubPath = %q, want /my_sub", cfg.HTTP.SubPath)
	}
	if cfg.HTTP.APIPort != 9091 {
		t.Errorf("HTTP.APIPort = %d, want 9091", cfg.HTTP.APIPort)
	}
}

func TestLoadConfigPartial(t *testing.T) {
	content := `
version = 1

[daemon]
parallel = 3
timeout = 5
`
	path := writeTempFile(t, content)
	defer os.Remove(path)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.Daemon.Parallel != 3 {
		t.Errorf("Parallel = %d, want 3", cfg.Daemon.Parallel)
	}
	if cfg.Daemon.TimeoutSec != 5 {
		t.Errorf("TimeoutSec = %d, want 5", cfg.Daemon.TimeoutSec)
	}

	if cfg.Daemon.URLsFile != "urls.txt" {
		t.Errorf("URLsFile = %q, want default urls.txt", cfg.Daemon.URLsFile)
	}
	if cfg.Daemon.CycleSleepSec != 300 {
		t.Errorf("CycleSleepSec = %d, want default 300", cfg.Daemon.CycleSleepSec)
	}
	if cfg.Daemon.Depth != "standard" {
		t.Errorf("Depth = %q, want default standard", cfg.Daemon.Depth)
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	_, err := LoadConfig("does_not_exist.toml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoadConfigEmptyFile(t *testing.T) {
	path := writeTempFile(t, "")
	defer os.Remove(path)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.Version != 1 {
		t.Errorf("Version = %d, want 1", cfg.Version)
	}
	if cfg.Daemon.Parallel != 10 {
		t.Errorf("Parallel = %d, want default 10", cfg.Daemon.Parallel)
	}
}

func TestParallelClampLow(t *testing.T) {
	content := `[daemon]
parallel = -5
timeout = 10
`
	path := writeTempFile(t, content)
	defer os.Remove(path)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.Daemon.Parallel != 1 {
		t.Errorf("Parallel = %d, want 1 (clamped)", cfg.Daemon.Parallel)
	}
}

func TestParallelClampHigh(t *testing.T) {
	content := `[daemon]
parallel = 100
timeout = 10
`
	path := writeTempFile(t, content)
	defer os.Remove(path)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.Daemon.Parallel != 20 {
		t.Errorf("Parallel = %d, want 20 (clamped)", cfg.Daemon.Parallel)
	}
}

func TestTimeoutClampLow(t *testing.T) {
	content := `[daemon]
timeout = 0
`
	path := writeTempFile(t, content)
	defer os.Remove(path)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.Daemon.TimeoutSec < 1 {
		t.Errorf("TimeoutSec = %d, want at least 1", cfg.Daemon.TimeoutSec)
	}
}

func TestCycleSleepClampLow(t *testing.T) {
	content := `[daemon]
cycle_sleep = 1
`
	path := writeTempFile(t, content)
	defer os.Remove(path)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.Daemon.CycleSleepSec < 10 {
		t.Errorf("CycleSleepSec = %d, want at least 10", cfg.Daemon.CycleSleepSec)
	}
}

func TestInvalidDepth(t *testing.T) {
	content := `[daemon]
depth = "ultra"
`
	path := writeTempFile(t, content)
	defer os.Remove(path)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.Daemon.Depth != "standard" {
		t.Errorf("Depth = %q, want standard (fallback)", cfg.Daemon.Depth)
	}
}

func TestConfigTimeoutDuration(t *testing.T) {
	d := DaemonConfig{TimeoutSec: 15}
	if d.Timeout() != 15_000_000_000 {
		t.Errorf("Timeout() = %v, want 15s", d.Timeout())
	}
}

func TestConfigCycleSleepDuration(t *testing.T) {
	d := DaemonConfig{CycleSleepSec: 120}
	if d.CycleSleep() != 120_000_000_000 {
		t.Errorf("CycleSleep() = %v, want 120s", d.CycleSleep())
	}
}

func TestConfigRetestIntervalDuration(t *testing.T) {
	d := DaemonConfig{RetestIntervalSec: 3600}
	if d.RetestInterval() != 3600_000_000_000 {
		t.Errorf("RetestInterval() = %v, want 3600s", d.RetestInterval())
	}
}

func TestConfigTestDepth(t *testing.T) {
	d := DaemonConfig{Depth: "comprehensive"}
	if d.TestDepth() != "comprehensive" {
		t.Errorf("TestDepth() = %q, want comprehensive", d.TestDepth())
	}
}

func TestConfigString(t *testing.T) {
	cfg := DefaultConfig()
	s := cfg.String()
	if len(s) == 0 {
		t.Fatal("String() returned empty")
	}
}

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}
