package daemon

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/amirrezaalavi/Viberay/internal/models"
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
}

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

	if v, ok := os.LookupEnv("DAEMON_KEEP_SUCCESSFUL"); ok {
		cfg.Daemon.KeepSuccessful = v == "true" || v == "1" || v == "yes"
	}

	if v, ok := os.LookupEnv("DAEMON_RETEST_INTERVAL"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Daemon.RetestIntervalSec = n
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

	depth := models.TestDepth(cfg.Daemon.Depth)
	if depth != "" && !depth.IsValid() {
		slog.Warn("unknown depth, using standard", "provided", cfg.Daemon.Depth)
		cfg.Daemon.Depth = "standard"
	}
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
		"urls=%s output=%s state=%s sleep=%ds parallel=%d timeout=%ds depth=%s keep_ok=%v retest_interval=%ds http_enabled=%v http_port=%d sub_path=%s api_port=%d",
		c.Daemon.URLsFile,
		c.Daemon.OutputFile,
		c.Daemon.StateFile,
		c.Daemon.CycleSleepSec,
		c.Daemon.Parallel,
		c.Daemon.TimeoutSec,
		c.Daemon.Depth,
		c.Daemon.KeepSuccessful,
		c.Daemon.RetestIntervalSec,
		c.HTTP.Enabled,
		c.HTTP.Port,
		c.HTTP.SubPath,
		c.HTTP.APIPort,
	)
}
