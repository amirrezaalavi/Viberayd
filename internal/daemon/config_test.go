package daemon

import (
	"os"
	"strings"
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
		"DAEMON_STABILITY_PROBES", "DAEMON_MAX_XRAY_TESTS", "DAEMON_PORT_BASE",
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
	if cfg.Daemon.Domains != DefaultDomains {
		t.Errorf("Domains = %q, want defaults", cfg.Daemon.Domains)
	}
	if cfg.Daemon.DomainTimeoutSec != 5 {
		t.Errorf("DomainTimeoutSec = %d, want 5", cfg.Daemon.DomainTimeoutSec)
	}
	if cfg.Daemon.DomainQuorum != 2 {
		t.Errorf("DomainQuorum = %d, want 2", cfg.Daemon.DomainQuorum)
	}
	if cfg.Daemon.DomainMinBytes != 4096 {
		t.Errorf("DomainMinBytes = %d, want 4096", cfg.Daemon.DomainMinBytes)
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
	setenv(t, "DAEMON_DOMAINS", "https://a.example, https://b.example")
	setenv(t, "DAEMON_DOMAIN_TIMEOUT", "7")
	setenv(t, "DAEMON_DOMAIN_QUORUM", "1")
	setenv(t, "DAEMON_DOMAIN_MIN_BYTES", "512")
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
	if cfg.Daemon.Domains != "https://a.example, https://b.example" {
		t.Errorf("Domains = %q, want override", cfg.Daemon.Domains)
	}
	if got := cfg.Daemon.DomainList(); len(got) != 2 || got[0] != "https://a.example" || got[1] != "https://b.example" {
		t.Errorf("DomainList = %q, want trimmed pair", got)
	}
	if cfg.Daemon.DomainTimeoutSec != 7 {
		t.Errorf("DomainTimeoutSec = %d, want 7", cfg.Daemon.DomainTimeoutSec)
	}
	if cfg.Daemon.DomainQuorum != 1 {
		t.Errorf("DomainQuorum = %d, want 1", cfg.Daemon.DomainQuorum)
	}
	if cfg.Daemon.DomainMinBytes != 512 {
		t.Errorf("DomainMinBytes = %d, want 512", cfg.Daemon.DomainMinBytes)
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

func TestDomainClamps(t *testing.T) {
	setenv(t, "DAEMON_DOMAIN_TIMEOUT", "0")
	setenv(t, "DAEMON_DOMAIN_QUORUM", "0")
	setenv(t, "DAEMON_DOMAIN_MIN_BYTES", "0")
	setenv(t, "DAEMON_DOMAINS", "")

	cfg := LoadConfigFromEnv()

	if cfg.Daemon.DomainTimeoutSec < 1 {
		t.Errorf("DomainTimeoutSec = %d, want at least 1", cfg.Daemon.DomainTimeoutSec)
	}
	if cfg.Daemon.DomainQuorum < 1 {
		t.Errorf("DomainQuorum = %d, want at least 1", cfg.Daemon.DomainQuorum)
	}
	if cfg.Daemon.DomainMinBytes < 1 {
		t.Errorf("DomainMinBytes = %d, want at least 1", cfg.Daemon.DomainMinBytes)
	}
	if len(cfg.Daemon.DomainList()) == 0 {
		t.Error("DomainList empty after blank override, want defaults restored")
	}
}

func TestConfigString(t *testing.T) {
	cfg := DefaultConfig()
	s := cfg.String()
	if len(s) == 0 {
		t.Fatal("String() returned empty")
	}
	if !strings.Contains(s, "port_base=10820") {
		t.Errorf("String() should report the port base: %s", s)
	}
}

func TestStabilityProbesConfig(t *testing.T) {
	if got := DefaultConfig().Daemon.StabilityProbes; got != 2 {
		t.Errorf("default StabilityProbes = %d, want 2", got)
	}

	t.Setenv("DAEMON_STABILITY_PROBES", "3")
	if got := LoadConfigFromEnv().Daemon.StabilityProbes; got != 3 {
		t.Errorf("StabilityProbes from env = %d, want 3", got)
	}

	// 0 would disable the gate silently; clamp to a single sample instead.
	t.Setenv("DAEMON_STABILITY_PROBES", "0")
	if got := LoadConfigFromEnv().Daemon.StabilityProbes; got != 1 {
		t.Errorf("StabilityProbes=0 clamped to %d, want 1", got)
	}

	t.Setenv("DAEMON_STABILITY_PROBES", "99")
	if got := LoadConfigFromEnv().Daemon.StabilityProbes; got != 10 {
		t.Errorf("StabilityProbes=99 clamped to %d, want 10", got)
	}

	if s := DefaultConfig().String(); !strings.Contains(s, "stability_probes=2") {
		t.Errorf("String() should report the probe count: %s", s)
	}
}

func TestMaxXrayTestsConfig(t *testing.T) {
	if got := DefaultConfig().Daemon.MaxXrayTests; got != 1200 {
		t.Errorf("default MaxXrayTests = %d, want 1200", got)
	}

	t.Setenv("DAEMON_MAX_XRAY_TESTS", "250")
	if got := LoadConfigFromEnv().Daemon.MaxXrayTests; got != 250 {
		t.Errorf("MaxXrayTests from env = %d, want 250", got)
	}

	// 0 is the documented "no cap" value and must survive validation.
	t.Setenv("DAEMON_MAX_XRAY_TESTS", "0")
	if got := LoadConfigFromEnv().Daemon.MaxXrayTests; got != 0 {
		t.Errorf("MaxXrayTests=0 = %d, want 0 (unlimited)", got)
	}

	t.Setenv("DAEMON_MAX_XRAY_TESTS", "-5")
	if got := LoadConfigFromEnv().Daemon.MaxXrayTests; got != 0 {
		t.Errorf("negative MaxXrayTests = %d, want 0 (unlimited)", got)
	}

	if s := DefaultConfig().String(); !strings.Contains(s, "max_xray_tests=1200") {
		t.Errorf("String() should report the cap: %s", s)
	}
}

func TestPortBaseConfig(t *testing.T) {
	if got := DefaultConfig().Daemon.PortBase; got != 10820 {
		t.Errorf("default PortBase = %d, want 10820", got)
	}

	t.Setenv("DAEMON_PORT_BASE", "11050")
	if got := LoadConfigFromEnv().Daemon.PortBase; got != 11050 {
		t.Errorf("PortBase from env = %d, want 11050", got)
	}

	// Out-of-range values fall back to the default instead of failing boot.
	t.Setenv("DAEMON_PORT_BASE", "80")
	if got := LoadConfigFromEnv().Daemon.PortBase; got != 10820 {
		t.Errorf("out-of-range PortBase = %d, want 10820", got)
	}
}
