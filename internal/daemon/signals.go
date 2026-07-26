package daemon

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func SignalContext(ctx context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-ctx.Done():
			return
		case sig := <-sigCh:
			slog.Info("received signal, shutting down", "signal", sig)
			cancel()
		}
	}()

	return ctx, func() {
		signal.Stop(sigCh)
		cancel()
	}
}
