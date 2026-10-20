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
		lat, err := runStabilityProbes(probes, func() (time.Duration, error) {
			calls++
			return 10 * time.Millisecond, nil
		})
		if err != nil {
			t.Errorf("probes=%d: unexpected error %v", probes, err)
		}
		if calls != 0 {
			t.Errorf("probes=%d: verify called %d times, want 0", probes, calls)
		}
		if len(lat) != 0 {
			t.Errorf("probes=%d: got %d latencies, want 0", probes, len(lat))
		}
	}

	// probes=2 runs exactly one extra verification; probes=3 runs two.
	for probes, want := range map[int]int{2: 1, 3: 2} {
		calls := 0
		lat, err := runStabilityProbes(probes, func() (time.Duration, error) {
			calls++
			return time.Duration(calls) * time.Millisecond, nil
		})
		if err != nil {
			t.Errorf("probes=%d: unexpected error %v", probes, err)
		}
		if calls != want {
			t.Errorf("probes=%d: verify called %d times, want %d", probes, calls, want)
		}
		if len(lat) != want {
			t.Errorf("probes=%d: got %d latencies, want %d", probes, len(lat), want)
		}
	}
}

func TestRunStabilityProbesFailsOnAnyProbe(t *testing.T) {
	boom := errors.New("tunnel dropped")
	calls := 0
	_, err := runStabilityProbes(3, func() (time.Duration, error) {
		calls++
		if calls == 2 {
			return 0, boom
		}
		return time.Millisecond, nil
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

func TestMedianDuration(t *testing.T) {
	cases := []struct {
		name    string
		samples []time.Duration
		want    time.Duration
	}{
		{"none", nil, 0},
		{"single", []time.Duration{700 * time.Millisecond}, 700 * time.Millisecond},
		{"odd count takes the middle", []time.Duration{900 * time.Millisecond, 300 * time.Millisecond, 600 * time.Millisecond}, 600 * time.Millisecond},
		{"even count averages the middle two", []time.Duration{400 * time.Millisecond, 200 * time.Millisecond}, 300 * time.Millisecond},
		{"one slow outlier does not win", []time.Duration{120 * time.Millisecond, 130 * time.Millisecond, 9 * time.Second}, 130 * time.Millisecond},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := medianDuration(tc.samples); got != tc.want {
				t.Errorf("medianDuration(%v) = %v, want %v", tc.samples, got, tc.want)
			}
		})
	}
}

func TestMedianDurationDoesNotMutateInput(t *testing.T) {
	samples := []time.Duration{300 * time.Millisecond, 100 * time.Millisecond, 200 * time.Millisecond}
	medianDuration(samples)
	if samples[0] != 300*time.Millisecond || samples[1] != 100*time.Millisecond {
		t.Errorf("input was reordered: %v", samples)
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
