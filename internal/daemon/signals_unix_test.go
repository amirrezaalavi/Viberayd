//go:build !windows

// The signal-delivery tests live here because syscall.Kill does not exist on
// Windows: keeping them in the shared file made `go vet` fail the Windows CI
// job ("undefined: syscall.Kill"), which failed every run of this branch
// before the tests were split.
package daemon

import (
	"context"
	"syscall"
	"testing"
	"time"
)

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
