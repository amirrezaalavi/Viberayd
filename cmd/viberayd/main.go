package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/amirrezaalavi/Viberay/internal/daemon"
)

func main() {
	configPath := flag.String("config", "config.toml", "path to configuration file")
	flag.Parse()

	logLevel := &slog.LevelVar{}
	logLevel.Set(slog.LevelInfo)
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))

	cfg, err := daemon.LoadConfig(*configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	fmt.Println("viberayd starting...")
	fmt.Printf("  config: %s\n", *configPath)
	fmt.Printf("  daemon: urls=%s output=%s state=%s sleep=%ds parallel=%d timeout=%ds depth=%s keep_ok=%v retest=%ds\n",
		cfg.Daemon.URLsFile,
		cfg.Daemon.OutputFile,
		cfg.Daemon.StateFile,
		cfg.Daemon.CycleSleepSec,
		cfg.Daemon.Parallel,
		cfg.Daemon.TimeoutSec,
		cfg.Daemon.Depth,
		cfg.Daemon.KeepSuccessful,
		cfg.Daemon.RetestIntervalSec,
	)
	fmt.Printf("  http: enabled=%v port=%d sub=%s api_port=%d\n",
		cfg.HTTP.Enabled,
		cfg.HTTP.Port,
		cfg.HTTP.SubPath,
		cfg.HTTP.APIPort,
	)
	fmt.Println("viberayd stopped.")
}
