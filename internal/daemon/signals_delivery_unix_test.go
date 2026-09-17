//go:build !windows

package daemon

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"
)

// These tests deliver a real signal to the current process, which requires a
// Unix syscall.Kill. Windows has no equivalent that raises SIGINT/SIGTERM via
// signal.Notify, so the sibling file signals_delivery_windows_test.go skips them.

func TestSignalContextCancelsOnSIGINT(t *testing.T) {
	ctx, cancel := SignalContext(context.Background())
	defer cancel()

	go func() {
		time.Sleep(20 * time.Millisecond)
		if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
			t.Errorf("failed to send SIGINT to self: %v", err)
		}
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
		if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
			t.Errorf("failed to send SIGTERM to self: %v", err)
		}
	}()

	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("context was not cancelled after SIGTERM")
	}
}
