package daemon

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/amirrezaalavi/Viberay/internal/models"
)

type Config struct {
	Version int          `toml:"version"`
	Daemon  DaemonConfig `toml:"daemon"`
	HTTP    HTTPConfig   `toml:"http"`
}

type DaemonConfig struct {
	URLsFile         string `toml:"urls_file"`
	OutputFile       string `toml:"output_file"`
	StateFile        string `toml:"state_file"`
	CycleSleepSec    int    `toml:"cycle_sleep"`
	Parallel         int    `toml:"parallel"`
	TimeoutSec       int    `toml:"timeout"`
	Depth            string `toml:"depth"`
	KeepSuccessful   bool   `toml:"keep_successful"`
	RetestIntervalSec int   `toml:"retest_interval"`
}

type HTTPConfig struct {
	Enabled bool   `toml:"enabled"`
	Port    int    `toml:"port"`
	SubPath string `toml:"sub_path"`
	APIPort int    `toml:"api_port"`
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

func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, fmt.Errorf("config file not found: %s", path)
	}

	md, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return cfg, fmt.Errorf("decode config: %w", err)
	}

	undecoded := md.Undecoded()
	if len(undecoded) > 0 {
		slog.Warn("unknown config keys", "keys", undecoded)
	}

	applyDefaults(&cfg)
	validate(&cfg)

	return cfg, nil
}

func applyDefaults(cfg *Config) {
	d := DefaultConfig()

	if cfg.Daemon.URLsFile == "" {
		cfg.Daemon.URLsFile = d.Daemon.URLsFile
	}
	if cfg.Daemon.OutputFile == "" {
		cfg.Daemon.OutputFile = d.Daemon.OutputFile
	}
	if cfg.Daemon.StateFile == "" {
		cfg.Daemon.StateFile = d.Daemon.StateFile
	}
	if cfg.Daemon.CycleSleepSec == 0 {
		cfg.Daemon.CycleSleepSec = d.Daemon.CycleSleepSec
	}
	if cfg.Daemon.Parallel == 0 {
		cfg.Daemon.Parallel = d.Daemon.Parallel
	}
	if cfg.Daemon.TimeoutSec == 0 {
		cfg.Daemon.TimeoutSec = d.Daemon.TimeoutSec
	}
	if cfg.Daemon.Depth == "" {
		cfg.Daemon.Depth = d.Daemon.Depth
	}
	if cfg.Daemon.RetestIntervalSec == 0 {
		cfg.Daemon.RetestIntervalSec = d.Daemon.RetestIntervalSec
	}
	if cfg.HTTP.Port == 0 {
		cfg.HTTP.Port = d.HTTP.Port
	}
	if cfg.HTTP.SubPath == "" {
		cfg.HTTP.SubPath = d.HTTP.SubPath
	}
	if cfg.HTTP.APIPort == 0 {
		cfg.HTTP.APIPort = d.HTTP.APIPort
	}
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
