package tester

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/amirrezaalavi/Viberayd/internal/models"
)

func TestRunStabilityProbes(t *testing.T) {
	// probes<=1 must not run any extra verification (single-sample behaviour).
	for _, probes := range []int{0, 1} {
		calls := 0
		if err := runStabilityProbes(probes, func() error { calls++; return nil }); err != nil {
			t.Errorf("probes=%d: unexpected error %v", probes, err)
		}
		if calls != 0 {
			t.Errorf("probes=%d: verify called %d times, want 0", probes, calls)
		}
	}

	// probes=2 runs exactly one extra verification; probes=3 runs two.
	for probes, want := range map[int]int{2: 1, 3: 2} {
		calls := 0
		if err := runStabilityProbes(probes, func() error { calls++; return nil }); err != nil {
			t.Errorf("probes=%d: unexpected error %v", probes, err)
		}
		if calls != want {
			t.Errorf("probes=%d: verify called %d times, want %d", probes, calls, want)
		}
	}
}

func TestRunStabilityProbesFailsOnAnyProbe(t *testing.T) {
	boom := errors.New("tunnel dropped")
	calls := 0
	err := runStabilityProbes(3, func() error {
		calls++
		if calls == 2 {
			return boom
		}
		return nil
	})
	if err == nil {
		t.Fatal("expected an error when a probe fails")
	}
	if !errors.Is(err, boom) {
		t.Errorf("error should wrap the probe failure, got %v", err)
	}
	if !strings.Contains(err.Error(), "stability probe 3/3") {
		t.Errorf("error should name the failing probe, got %v", err)
	}
	if calls != 2 {
		t.Errorf("verify called %d times, want 2 (stop at first failure)", calls)
	}
}

func TestPipelineBudgetCoversAllStages(t *testing.T) {
	p := NewPipeline(models.DepthFull)
	p.TCPTimeout = 10 * time.Second
	p.TLSTimeout = 10 * time.Second
	p.ProtoTimeout = 10 * time.Second
	p.XrayTimeout = 10 * time.Second
	p.StabilityProbes = 2
	p.DomainURLs = []string{"https://example.com"}

	// TCP + TLS + protocol + 2 proxy passes + 2 domain passes = 7 units.
	if got, want := p.Budget(), 70*time.Second; got != want {
		t.Errorf("full budget = %v, want %v", got, want)
	}

	// A single sample and no domains needs only 4 units.
	p.StabilityProbes = 1
	p.XrayTimeout = 10 * time.Second
	p.DomainURLs = nil
	if got, want := p.Budget(), 40*time.Second; got != want {
		t.Errorf("standard budget = %v, want %v", got, want)
	}

	// A zero-value pipeline must still return a usable bound.
	if got := (&Pipeline{}).Budget(); got <= 0 {
		t.Errorf("zero pipeline budget = %v, want > 0", got)
	}
}
