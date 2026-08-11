package tester

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amirrezaalavi/Viberayd/internal/models"
)

func TestShouldRetry(t *testing.T) {
	if shouldRetry(models.TestResult{Status: models.StatusSuccess}) {
		t.Error("shouldRetry should be false for success")
	}
	if !shouldRetry(models.TestResult{Status: models.StatusFailed}) {
		t.Error("shouldRetry should be true for failed")
	}
	if !shouldRetry(models.TestResult{Status: models.StatusError}) {
		t.Error("shouldRetry should be true for error")
	}
}

func TestResilientRunner_RetriesOnFailure(t *testing.T) {
	pipe := NewPipeline(models.DepthQuick)
	rr := NewResilientRunner(pipe, models.OrchestratorDecision{
		Retry: models.RetryPolicy{MaxRetries: 2, BackoffBase: 50 * time.Millisecond},
	})

	cfg := models.ProxyConfig{SS: &models.SSConfig{BaseConfig: models.BaseConfig{Server: "127.0.0.1", Port: 1}}}
	var par atomic.Int32
	par.Store(4)

	res := rr.Run(context.Background(), cfg, 0, &par)
	// 127.0.0.1:1 should fail quickly (connection refused) and retry
	if res.Status != models.StatusFailed && res.Status != models.StatusError {
		t.Fatalf("expected failure, got %q", res.Status)
	}
	if res.Retries == 0 {
		t.Error("expected at least 1 retry on failure")
	}
}

func TestResilientRunner_MaxRetries(t *testing.T) {
	pipe := NewPipeline(models.DepthQuick)
	rr := NewResilientRunner(pipe, models.OrchestratorDecision{
		Retry: models.RetryPolicy{MaxRetries: 1, BackoffBase: 10 * time.Millisecond},
	})

	cfg := models.ProxyConfig{SS: &models.SSConfig{BaseConfig: models.BaseConfig{Server: "127.0.0.1", Port: 1}}}
	var par atomic.Int32
	par.Store(4)

	res := rr.Run(context.Background(), cfg, 0, &par)
	if res.Retries > 1 {
		t.Errorf("expected at most 1 retry, got %d", res.Retries)
	}
}

func TestResilientRunner_ContextCancelDuringBackoff(t *testing.T) {
	pipe := NewPipeline(models.DepthQuick)
	rr := NewResilientRunner(pipe, models.OrchestratorDecision{
		Retry: models.RetryPolicy{MaxRetries: 3, BackoffBase: 5 * time.Second},
	})

	cfg := models.ProxyConfig{SS: &models.SSConfig{BaseConfig: models.BaseConfig{Server: "127.0.0.1", Port: 1}}}
	ctx, cancel := context.WithCancel(context.Background())
	var par atomic.Int32
	par.Store(4)

	// Cancel immediately so backoff is interrupted
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	res := rr.Run(ctx, cfg, 0, &par)
	if res.Status != models.StatusError && res.Status != models.StatusFailed {
		t.Fatalf("expected error/failed after cancel, got %q", res.Status)
	}
	if len(res.Errors) == 0 {
		t.Error("expected error message about cancellation")
	}
}
