package tester

import (
	"context"
	"testing"
	"time"

	"github.com/amirrezaalavi/Viberayd/internal/models"
)

// fakeXrayRunner counts the end-to-end verifications and can be told to
// fail one of them. It replaces the real xray runner so the depth and
// stability logic can be exercised without spawning a tunnel.
type fakeXrayRunner struct {
	calls     int
	failAt    int // 1-based call index that fails; 0 = never
	latency   time.Duration
	tempCalls int
}

func (f *fakeXrayRunner) TestXrayProxy(ctx context.Context, cfg models.ProxyConfig, socksPort int, timeout time.Duration) XrayResult {
	f.calls++
	lat := f.latency
	if lat == 0 {
		lat = 100 * time.Millisecond
	}
	if f.failAt == f.calls {
		return XrayResult{Success: false, Latency: lat, Error: "fake failure"}
	}
	return XrayResult{Success: true, Latency: lat, HTTPCode: 204}
}

func (f *fakeXrayRunner) withTempXray(ctx context.Context, cfg models.ProxyConfig, port int, timeout time.Duration, fn func(proxyAddr string) error) error {
	f.tempCalls++
	return fn("127.0.0.1:1")
}

// withFakeXray installs a fake runner for the duration of a test.
func withFakeXray(t *testing.T, f *fakeXrayRunner) {
	t.Helper()
	old := newXrayRunner
	newXrayRunner = func(string) xrayRunner { return f }
	t.Cleanup(func() { newXrayRunner = old })
}

// standardPipeline is a standard-depth pipeline whose cheap stages are
// skipped: the test drives the proxy/stability stages directly.
func standardPipeline() *Pipeline {
	p := NewPipeline(models.DepthStandard)
	p.TCPTimeout = time.Second
	p.TLSTimeout = time.Second
	p.ProtoTimeout = time.Second
	p.XrayTimeout = time.Second
	return p
}

// TestStabilityGateRunsAtStandardDepth pins the fix: the repeat
// verification used to be tied to the FULL-depth domain stage, so the
// DEFAULT depth (standard) had no gate at all — DAEMON_STABILITY_PROBES
// silently did nothing there.
func TestStabilityGateRunsAtStandardDepth(t *testing.T) {
	fake := &fakeXrayRunner{}
	withFakeXray(t, fake)

	p := standardPipeline()
	p.StabilityProbes = 3

	// The TCP/TLS/protocol stages need a live endpoint; the test drives
	// the proxy stage through the fake runner, so the pre-proxy stages
	// are satisfied by a local acceptor.
	port := localAcceptor(t)
	res := p.Run(context.Background(), localSSConfig(port), 18751)
	if res.Status != models.StatusSuccess {
		t.Fatalf("status = %q errors %v, want success", res.Status, res.Errors)
	}
	// 1 initial verification + (probes-1) stability probes.
	if fake.calls != 3 {
		t.Errorf("end-to-end verifications = %d, want 3 (1 + 2 stability probes)", fake.calls)
	}
}

// TestStabilityGateFailsConfigAtStandardDepth: a repeat verification that
// fails rejects the config, at standard depth too.
func TestStabilityGateFailsConfigAtStandardDepth(t *testing.T) {
	fake := &fakeXrayRunner{failAt: 2}
	withFakeXray(t, fake)

	p := standardPipeline()
	p.StabilityProbes = 2

	port := localAcceptor(t)
	res := p.Run(context.Background(), localSSConfig(port), 18752)
	if res.Status != models.StatusFailed {
		t.Fatalf("status = %q, want failed", res.Status)
	}
	if res.Stage != models.StageStability {
		t.Errorf("stage = %q, want %q", res.Stage, models.StageStability)
	}
}

// TestStabilityProbesOneKeepsSingleSample: with one probe there is no
// repeat — the historical single-sample behavior.
func TestStabilityProbesOneKeepsSingleSample(t *testing.T) {
	fake := &fakeXrayRunner{}
	withFakeXray(t, fake)

	p := standardPipeline()
	p.StabilityProbes = 1

	port := localAcceptor(t)
	res := p.Run(context.Background(), localSSConfig(port), 18753)
	if res.Status != models.StatusSuccess {
		t.Fatalf("status = %q errors %v, want success", res.Status, res.Errors)
	}
	if fake.calls != 1 {
		t.Errorf("end-to-end verifications = %d, want 1 (no repeat)", fake.calls)
	}
}

// TestResponseLatencyIsMedianOfThePasses: the recorded latency is the
// median of the end-to-end passes, not the total run duration.
func TestResponseLatencyIsMedianOfThePasses(t *testing.T) {
	fake := &fakeXrayRunner{latency: 250 * time.Millisecond}
	withFakeXray(t, fake)

	p := standardPipeline()
	p.StabilityProbes = 2

	port := localAcceptor(t)
	res := p.Run(context.Background(), localSSConfig(port), 18754)
	if res.Status != models.StatusSuccess {
		t.Fatalf("status = %q errors %v, want success", res.Status, res.Errors)
	}
	if res.Latencies.Response != 250*time.Millisecond {
		t.Errorf("Response latency = %v, want 250ms", res.Latencies.Response)
	}
	if res.Latencies.Total <= 0 {
		t.Error("Total run duration is not set")
	}
}

// TestDomainStageOnlyAtFullDepth: the domain content check is a full-depth
// stage; standard depth must not run it (and therefore must not need a
// domain-capable tunnel).
func TestDomainStageOnlyAtFullDepth(t *testing.T) {
	fake := &fakeXrayRunner{}
	withFakeXray(t, fake)

	p := standardPipeline()
	p.StabilityProbes = 1
	p.DomainURLs = []string{"https://example.com"}

	port := localAcceptor(t)
	res := p.Run(context.Background(), localSSConfig(port), 18755)
	if res.Status != models.StatusSuccess {
		t.Fatalf("status = %q errors %v, want success", res.Status, res.Errors)
	}
	if fake.tempCalls != 0 {
		t.Errorf("temp xray runs = %d, want 0 (no domain stage at standard depth)", fake.tempCalls)
	}
}

// TestVerifyEndToEndReturnsTheMeasuredLatency keeps the seam honest: the
// verification reports the latency it measured.
func TestVerifyEndToEndReturnsTheMeasuredLatency(t *testing.T) {
	fake := &fakeXrayRunner{latency: 175 * time.Millisecond}
	withFakeXray(t, fake)

	p := standardPipeline()
	lat, err := p.verifyEndToEnd(context.Background(), localSSConfig(1), 18756, models.DepthStandard)
	if err != nil {
		t.Fatalf("verifyEndToEnd: %v", err)
	}
	if lat != 175*time.Millisecond {
		t.Errorf("latency = %v, want 175ms", lat)
	}

	failing := &fakeXrayRunner{failAt: 1}
	withFakeXray(t, failing)
	if _, err := p.verifyEndToEnd(context.Background(), localSSConfig(1), 18757, models.DepthStandard); err == nil {
		t.Error("verifyEndToEnd returned no error for a failed verification")
	}
}
