//go:build windows

package daemon

import (
	"testing"
)

// Unix signal delivery to self is not reproducible on Windows: os.Process.Signal
// only supports os.Kill, and signal.Notify does not translate SIGINT/SIGTERM into
// a channel event there. The equivalent Unix coverage lives in
// signals_delivery_unix_test.go; these stubs keep the test names visible in
// `go test` output on Windows without silently claiming coverage.
//
// The portable behaviour of SignalContext (initial state, double cancel,
// pre-cancelled parent) is covered by signals_test.go on every platform.

func TestSignalContextCancelsOnSIGINT(t *testing.T) {
	t.Skip("SIGINT delivery to self is not supported on Windows")
}

func TestSignalContextCancelsOnSIGTERM(t *testing.T) {
	t.Skip("SIGTERM delivery to self is not supported on Windows")
}
