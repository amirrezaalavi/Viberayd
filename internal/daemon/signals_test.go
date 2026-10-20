package daemon

import (
	"context"
	"testing"
)

func TestSignalContextCancel(t *testing.T) {
	ctx, cancel := SignalContext(context.Background())
	defer cancel()

	if ctx.Err() != nil {
		t.Fatal("context should not be cancelled initially")
	}
}

func TestSignalContextDoubleCancel(t *testing.T) {
	ctx, cancel := SignalContext(context.Background())

	cancel()
	cancel()

	<-ctx.Done()
}

func TestSignalContextPreCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	sigCtx, sigCancel := SignalContext(ctx)
	defer sigCancel()

	<-sigCtx.Done()
}
