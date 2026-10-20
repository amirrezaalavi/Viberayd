package tester

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestWaitForListenerDetectsBoundPort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	defer ln.Close()

	if !waitForListener(context.Background(), addr, time.Second) {
		t.Errorf("waitForListener(%s) = false, want true for a bound listener", addr)
	}
}

func TestWaitForListenerTimesOutWhenClosed(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	start := time.Now()
	if waitForListener(context.Background(), addr, 300*time.Millisecond) {
		t.Fatalf("waitForListener(%s) = true, want false for a closed listener", addr)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("waitForListener took %s, want a bounded wait", elapsed)
	}
}

func TestWaitForListenerHonoursContext(t *testing.T) {
	// A port that is very unlikely to be bound.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if waitForListener(ctx, "127.0.0.1:1", 5*time.Second) {
		t.Error("waitForListener returned true with a cancelled context")
	}
}

func TestTailText(t *testing.T) {
	if got := tailText("  hello  ", 100); got != "hello" {
		t.Errorf("tailText trim = %q, want %q", got, "hello")
	}
	if got := tailText("abcdefghij", 4); got != "...ghij" {
		t.Errorf("tailText truncate = %q, want %q", got, "...ghij")
	}
}
