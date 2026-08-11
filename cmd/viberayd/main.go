package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/amirrezaalavi/Viberayd/internal/daemon"
)

func main() {
	singleCycle := flag.Bool("once", false, "run a single cycle and exit")
	flag.Parse()

	logLevel := &slog.LevelVar{}
	logLevel.Set(slog.LevelInfo)
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))

	cfg := daemon.LoadConfigFromEnv()

	fmt.Println("viberayd starting...")
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

	d := daemon.NewDaemon(&cfg)

	ctx, cancel := daemon.SignalContext(context.Background())
	defer cancel()

	if *singleCycle {
		if err := d.RunCycle(ctx); err != nil {
			slog.Error("cycle failed", "error", err)
			os.Exit(1)
		}
	} else {
		if err := d.Run(ctx); err != nil {
			slog.Error("daemon failed", "error", err)
			os.Exit(1)
		}
	}

	fmt.Println("viberayd stopped.")
}
