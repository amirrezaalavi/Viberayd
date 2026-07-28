package daemon

import (
	"os"
	"testing"
)

func setenv(t *testing.T, key, value string) {
	t.Helper()
	orig, ok := os.LookupEnv(key)
	os.Setenv(key, value)
	t.Cleanup(func() {
		if ok {
			os.Setenv(key, orig)
		} else {
			os.Unsetenv(key)
		}
	})
}

func unsetenv(t *testing.T, key string) {
	t.Helper()
	orig, ok := os.LookupEnv(key)
	os.Unsetenv(key)
	t.Cleanup(func() {
		if ok {
			os.Setenv(key, orig)
		}
	})
}

func TestDefaultConfig(t *testing.T) {
	for _, key := range []string{
		"DAEMON_URLS_FILE", "DAEMON_OUTPUT_FILE", "DAEMON_STATE_FILE",
		"DAEMON_CYCLE_SLEEP", "DAEMON_PARALLEL", "DAEMON_TIMEOUT",
		"DAEMON_DEPTH", "DAEMON_KEEP_SUCCESSFUL", "DAEMON_RETEST_INTERVAL",
		"HTTP_ENABLED", "HTTP_PORT", "HTTP_SUB_PATH", "HTTP_API_PORT",
	} {
		unsetenv(t, key)
	}

	cfg := LoadConfigFromEnv()

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
	setenv(t, "DAEMON_URLS_FILE", "my_urls.txt")
	setenv(t, "DAEMON_OUTPUT_FILE", "my_working.txt")
	setenv(t, "DAEMON_STATE_FILE", "my_state.json")
	setenv(t, "DAEMON_CYCLE_SLEEP", "60")
	setenv(t, "DAEMON_PARALLEL", "5")
	setenv(t, "DAEMON_TIMEOUT", "15")
	setenv(t, "DAEMON_DEPTH", "comprehensive")
	setenv(t, "DAEMON_KEEP_SUCCESSFUL", "false")
	setenv(t, "DAEMON_RETEST_INTERVAL", "3600")
	setenv(t, "HTTP_ENABLED", "true")
	setenv(t, "HTTP_PORT", "9090")
	setenv(t, "HTTP_SUB_PATH", "/my_sub")
	setenv(t, "HTTP_API_PORT", "9091")

	cfg := LoadConfigFromEnv()

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
	setenv(t, "DAEMON_PARALLEL", "3")
	setenv(t, "DAEMON_TIMEOUT", "5")

	cfg := LoadConfigFromEnv()

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

func TestParallelClampLow(t *testing.T) {
	setenv(t, "DAEMON_PARALLEL", "-5")

	cfg := LoadConfigFromEnv()

	if cfg.Daemon.Parallel != 1 {
		t.Errorf("Parallel = %d, want 1 (clamped)", cfg.Daemon.Parallel)
	}
}

func TestParallelClampHigh(t *testing.T) {
	setenv(t, "DAEMON_PARALLEL", "100")

	cfg := LoadConfigFromEnv()

	if cfg.Daemon.Parallel != 20 {
		t.Errorf("Parallel = %d, want 20 (clamped)", cfg.Daemon.Parallel)
	}
}

func TestTimeoutClampLow(t *testing.T) {
	setenv(t, "DAEMON_TIMEOUT", "0")

	cfg := LoadConfigFromEnv()

	if cfg.Daemon.TimeoutSec < 1 {
		t.Errorf("TimeoutSec = %d, want at least 1", cfg.Daemon.TimeoutSec)
	}
}

func TestCycleSleepClampLow(t *testing.T) {
	setenv(t, "DAEMON_CYCLE_SLEEP", "1")

	cfg := LoadConfigFromEnv()

	if cfg.Daemon.CycleSleepSec < 10 {
		t.Errorf("CycleSleepSec = %d, want at least 10", cfg.Daemon.CycleSleepSec)
	}
}

func TestInvalidDepth(t *testing.T) {
	setenv(t, "DAEMON_DEPTH", "ultra")

	cfg := LoadConfigFromEnv()

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


