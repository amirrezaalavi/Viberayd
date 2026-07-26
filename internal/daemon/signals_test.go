package daemon

import (
	"context"
	"syscall"
	"testing"
	"time"
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

func TestSignalContextCancelsOnSIGINT(t *testing.T) {
	ctx, cancel := SignalContext(context.Background())
	defer cancel()

	go func() {
		time.Sleep(20 * time.Millisecond)
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	}()

	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("context was not cancelled after SIGINT")
	}
}

func TestSignalContextCancelsOnSIGTERM(t *testing.T) {
	ctx, cancel := SignalContext(context.Background())
	defer cancel()

	go func() {
		time.Sleep(20 * time.Millisecond)
		syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	}()

	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("context was not cancelled after SIGTERM")
	}
}

func TestSignalContextPreCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	sigCtx, sigCancel := SignalContext(ctx)
	defer sigCancel()

	<-sigCtx.Done()
}
