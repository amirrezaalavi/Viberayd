package daemon

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/amirrezaalavi/Viberayd/internal/models"
)

type Config struct {
	Version int
	Daemon  DaemonConfig
	HTTP    HTTPConfig
}

type DaemonConfig struct {
	URLsFile          string
	OutputFile        string
	StateFile         string
	CycleSleepSec     int
	Parallel          int
	TimeoutSec        int
	Depth             string
	KeepSuccessful    bool
	RetestIntervalSec int
	MaxLatencyMs      int
	// PortBase is the first local SOCKS port the xray test pool draws from.
	// It must differ between daemon instances sharing a host so two pools can
	// never race for the same port.
	PortBase int
	// Full-depth domain content check (see tester.CheckDomains).
	Domains          string // comma-separated URLs
	DomainTimeoutSec int
	DomainQuorum     int
	DomainMinBytes   int
	// StabilityProbes is how many independent end-to-end verifications a
	// config must pass before it is reported as working. 1 = the old
	// single-sample behaviour; 2 is the default so a node that fails most
	// attempts never reaches working.txt.
	StabilityProbes int
	// MaxXrayTests caps how many TCP-reachable configs go on to the expensive
	// stages (real proxy test, domain check, stability probes) per cycle,
	// keeping the fastest TCP handshakes. The cheap TCP pre-filter kills most
	// of a large corpus, but the survivors each need a subprocess and can burn
	// tens of seconds on a path that is reachable yet dead. Without a cap a
	// ~650k-candidate corpus never finished a cycle once probes were enabled
	// (measured: still running after 30 min, nothing published). 0 = no cap.
	MaxXrayTests int
}

// DefaultDomains is the full-depth content-check set: large, independent,
// geographically distributed endpoints.
const DefaultDomains = "https://youtube.com,https://www.google.com,https://www.cloudflare.com"

type HTTPConfig struct {
	Enabled bool
	Port    int
	SubPath string
	APIPort int
}

func DefaultConfig() Config {
	return Config{
		Version: 1,
		Daemon: DaemonConfig{
			URLsFile:          "urls.txt",
			OutputFile:        "working.txt",
			StateFile:         "state.json",
			CycleSleepSec:     300,
			Parallel:          10,
			TimeoutSec:        10,
			Depth:             "standard",
			KeepSuccessful:    true,
			RetestIntervalSec: 1800,
			PortBase:          10820,
			Domains:           DefaultDomains,
			DomainTimeoutSec:  5,
			DomainQuorum:      2,
			DomainMinBytes:    4096,
			StabilityProbes:   2,
			MaxXrayTests:      1200,
		},
		HTTP: HTTPConfig{
			Enabled: false,
			Port:    8080,
			SubPath: "/sub",
			APIPort: 8081,
		},
	}
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

func LoadConfigFromEnv() Config {
	cfg := DefaultConfig()

	cfg.Daemon.URLsFile = env("DAEMON_URLS_FILE", cfg.Daemon.URLsFile)
	cfg.Daemon.OutputFile = env("DAEMON_OUTPUT_FILE", cfg.Daemon.OutputFile)
	cfg.Daemon.StateFile = env("DAEMON_STATE_FILE", cfg.Daemon.StateFile)

	if v, ok := os.LookupEnv("DAEMON_CYCLE_SLEEP"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Daemon.CycleSleepSec = n
		}
	}

	if v, ok := os.LookupEnv("DAEMON_PARALLEL"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Daemon.Parallel = n
		}
	}

	if v, ok := os.LookupEnv("DAEMON_TIMEOUT"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Daemon.TimeoutSec = n
		}
	}

	cfg.Daemon.Depth = env("DAEMON_DEPTH", cfg.Daemon.Depth)
	cfg.Daemon.Domains = env("DAEMON_DOMAINS", cfg.Daemon.Domains)

	if v, ok := os.LookupEnv("DAEMON_DOMAIN_TIMEOUT"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Daemon.DomainTimeoutSec = n
		}
	}

	if v, ok := os.LookupEnv("DAEMON_DOMAIN_QUORUM"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Daemon.DomainQuorum = n
		}
	}

	if v, ok := os.LookupEnv("DAEMON_DOMAIN_MIN_BYTES"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Daemon.DomainMinBytes = n
		}
	}

	if v, ok := os.LookupEnv("DAEMON_STABILITY_PROBES"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Daemon.StabilityProbes = n
		}
	}

	if v, ok := os.LookupEnv("DAEMON_MAX_XRAY_TESTS"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Daemon.MaxXrayTests = n
		}
	}

	if v, ok := os.LookupEnv("DAEMON_KEEP_SUCCESSFUL"); ok {
		cfg.Daemon.KeepSuccessful = v == "true" || v == "1" || v == "yes"
	}

	if v, ok := os.LookupEnv("DAEMON_RETEST_INTERVAL"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Daemon.RetestIntervalSec = n
		}
	}

	if v, ok := os.LookupEnv("DAEMON_MAX_LATENCY_MS"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Daemon.MaxLatencyMs = n
		}
	}

	if v, ok := os.LookupEnv("DAEMON_PORT_BASE"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Daemon.PortBase = n
		}
	}

	if v, ok := os.LookupEnv("HTTP_ENABLED"); ok {
		cfg.HTTP.Enabled = v == "true" || v == "1" || v == "yes"
	}

	if v, ok := os.LookupEnv("HTTP_PORT"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.HTTP.Port = n
		}
	}

	cfg.HTTP.SubPath = env("HTTP_SUB_PATH", cfg.HTTP.SubPath)

	if v, ok := os.LookupEnv("HTTP_API_PORT"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.HTTP.APIPort = n
		}
	}

	validate(&cfg)

	return cfg
}

func validate(cfg *Config) {
	if cfg.Daemon.Parallel < 1 {
		slog.Warn("parallel too low, clamping to 1", "provided", cfg.Daemon.Parallel)
		cfg.Daemon.Parallel = 1
	}
	if cfg.Daemon.Parallel > 20 {
		slog.Warn("parallel too high, clamping to 20", "provided", cfg.Daemon.Parallel)
		cfg.Daemon.Parallel = 20
	}
	if cfg.Daemon.TimeoutSec < 1 {
		slog.Warn("timeout too low, setting to 1", "provided", cfg.Daemon.TimeoutSec)
		cfg.Daemon.TimeoutSec = 1
	}
	if cfg.Daemon.CycleSleepSec < 10 {
		slog.Warn("cycle_sleep too low, setting to 10", "provided", cfg.Daemon.CycleSleepSec)
		cfg.Daemon.CycleSleepSec = 10
	}
	if cfg.Daemon.MaxLatencyMs < 0 {
		slog.Warn("max_latency_ms too low, disabling threshold", "provided", cfg.Daemon.MaxLatencyMs)
		cfg.Daemon.MaxLatencyMs = 0
	}
	// A port base below 1024 needs privileges and a base near the top of the
	// range cannot fit the test pool; fall back loudly rather than failing.
	if cfg.Daemon.PortBase < 1024 || cfg.Daemon.PortBase > 64000 {
		slog.Warn("port_base out of range, using default 10820", "provided", cfg.Daemon.PortBase)
		cfg.Daemon.PortBase = 10820
	}

	depth := models.TestDepth(cfg.Daemon.Depth)
	if depth != "" && !depth.IsValid() {
		slog.Warn("unknown depth, using standard", "provided", cfg.Daemon.Depth)
		cfg.Daemon.Depth = "standard"
	}
	if depth == models.DepthComprehensive {
		slog.Warn("depth comprehensive is deprecated, behaving as full", "provided", cfg.Daemon.Depth)
	}
	if cfg.Daemon.DomainTimeoutSec < 1 {
		slog.Warn("domain timeout too low, setting to 1", "provided", cfg.Daemon.DomainTimeoutSec)
		cfg.Daemon.DomainTimeoutSec = 1
	}
	if cfg.Daemon.DomainQuorum < 1 {
		slog.Warn("domain quorum too low, setting to 1", "provided", cfg.Daemon.DomainQuorum)
		cfg.Daemon.DomainQuorum = 1
	}
	if cfg.Daemon.DomainMinBytes < 1 {
		slog.Warn("domain min bytes too low, setting to 1", "provided", cfg.Daemon.DomainMinBytes)
		cfg.Daemon.DomainMinBytes = 1
	}
	if cfg.Daemon.Domains == "" {
		slog.Warn("empty domain list, restoring defaults")
		cfg.Daemon.Domains = DefaultDomains
	}
	if cfg.Daemon.StabilityProbes < 1 {
		slog.Warn("stability probes too low, setting to 1 (single sample)", "provided", cfg.Daemon.StabilityProbes)
		cfg.Daemon.StabilityProbes = 1
	}
	if cfg.Daemon.StabilityProbes > 10 {
		slog.Warn("stability probes too high, clamping to 10", "provided", cfg.Daemon.StabilityProbes)
		cfg.Daemon.StabilityProbes = 10
	}
	if cfg.Daemon.MaxXrayTests < 0 {
		slog.Warn("max xray tests negative, disabling cap", "provided", cfg.Daemon.MaxXrayTests)
		cfg.Daemon.MaxXrayTests = 0
	}
	if cfg.Daemon.MaxXrayTests > 100000 {
		slog.Warn("max xray tests too high, clamping to 100000", "provided", cfg.Daemon.MaxXrayTests)
		cfg.Daemon.MaxXrayTests = 100000
	}
}

// DomainList splits the configured comma-separated domain URLs, dropping
// blanks. Never empty: validate restores defaults for a blank value.
func (c *DaemonConfig) DomainList() []string {
	var out []string
	for _, d := range strings.Split(c.Domains, ",") {
		if d = strings.TrimSpace(d); d != "" {
			out = append(out, d)
		}
	}
	if len(out) == 0 {
		return strings.Split(DefaultDomains, ",")
	}
	return out
}

func (c *DaemonConfig) Timeout() time.Duration {
	return time.Duration(c.TimeoutSec) * time.Second
}

func (c *DaemonConfig) CycleSleep() time.Duration {
	return time.Duration(c.CycleSleepSec) * time.Second
}

func (c *DaemonConfig) RetestInterval() time.Duration {
	return time.Duration(c.RetestIntervalSec) * time.Second
}

func (c *DaemonConfig) TestDepth() models.TestDepth {
	return models.TestDepth(c.Depth)
}

func (c Config) String() string {
	return fmt.Sprintf(
		"urls=%s output=%s state=%s sleep=%ds parallel=%d timeout=%ds depth=%s keep_ok=%v retest_interval=%ds max_latency_ms=%d port_base=%d stability_probes=%d max_xray_tests=%d http_enabled=%v http_port=%d sub_path=%s api_port=%d",
		c.Daemon.URLsFile,
		c.Daemon.OutputFile,
		c.Daemon.StateFile,
		c.Daemon.CycleSleepSec,
		c.Daemon.Parallel,
		c.Daemon.TimeoutSec,
		c.Daemon.Depth,
		c.Daemon.KeepSuccessful,
		c.Daemon.RetestIntervalSec,
		c.Daemon.MaxLatencyMs,
		c.Daemon.PortBase,
		c.Daemon.StabilityProbes,
		c.Daemon.MaxXrayTests,
		c.HTTP.Enabled,
		c.HTTP.Port,
		c.HTTP.SubPath,
		c.HTTP.APIPort,
	)
}
