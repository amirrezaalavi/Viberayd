package tester

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/amiralavi/viberay/internal/errors"
	"github.com/amiralavi/viberay/internal/models"
)

// ResilientRunner wraps Pipeline.Run with retry, backoff, categorization,
// and optional parallelism reduction on resource exhaustion.
type ResilientRunner struct {
	Pipe         *Pipeline
	MaxRetries   int
	BackoffBase  time.Duration
	ReduceLoadOn bool // if true, halve parallelism after resource/runtime errors
}

// NewResilientRunner creates a runner with defaults from an OrchestratorDecision.
func NewResilientRunner(pipe *Pipeline, decision models.OrchestratorDecision) *ResilientRunner {
	return &ResilientRunner{
		Pipe:         pipe,
		MaxRetries:   decision.Retry.MaxRetries,
		BackoffBase:  decision.Retry.BackoffBase,
		ReduceLoadOn: true,
	}
}

// Run executes the pipeline for cfg with retry and recovery logic.
// It returns the final TestResult and a flag indicating whether load was reduced.
func (rr *ResilientRunner) Run(ctx context.Context, cfg models.ProxyConfig, port int, parallelism *atomic.Int32) models.TestResult {
	res := rr.Pipe.Run(ctx, cfg, port)

	for attempt := 0; attempt < rr.MaxRetries && shouldRetry(res); attempt++ {
		wait := rr.BackoffBase * time.Duration(1<<attempt)
		if wait > 30*time.Second {
			wait = 30 * time.Second
		}

		select {
		case <-time.After(wait):
		case <-ctx.Done():
			res.Status = models.StatusError
			res.Errors = append(res.Errors, fmt.Sprintf("cancelled during retry backoff: %v", ctx.Err()))
			return res
		}

		slog.Debug("retrying config",
			"config", cfg.String(),
			"attempt", attempt+1,
			"backoff", wait,
		)
		res = rr.Pipe.Run(ctx, cfg, port)
		res.Retries = attempt + 1

		// If a resource/runtime error occurred, optionally reduce parallelism.
		if rr.ReduceLoadOn && (res.Status == models.StatusFailed || res.Status == models.StatusError) {
			for _, e := range res.Errors {
				strategy := errors.StrategyFor(fmt.Errorf("%s", e), cfg.String())
				if strategy.Action == errors.ActionReduceLoad || strategy.Action == errors.ActionRestart {
					newVal := parallelism.Load() / 2
					if newVal < 1 {
						newVal = 1
					}
					parallelism.Store(newVal)
					slog.Warn("reduced parallelism due to resource/runtime error",
						"config", cfg.String(),
						"new_parallelism", newVal,
						"category", strategy.Category,
					)
				}
			}
		}
	}

	return res
}

func shouldRetry(res models.TestResult) bool {
	return res.Status == models.StatusFailed || res.Status == models.StatusError
}
