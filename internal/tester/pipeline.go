package tester

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/amirrezaalavi/Viberayd/internal/models"
)

// xrayRunner is the part of the pipeline that needs a real tunnel. It is an
// interface so the depth and stability logic can be driven by tests without
// spawning xray.
type xrayRunner interface {
	TestXrayProxy(ctx context.Context, cfg models.ProxyConfig, socksPort int, timeout time.Duration) XrayResult
	withTempXray(ctx context.Context, cfg models.ProxyConfig, port int, timeout time.Duration, fn func(proxyAddr string) error) error
}

// newXrayRunner builds the runner for one stage. It is a variable so tests
// can inject a fake that counts the end-to-end verifications.
var newXrayRunner = func(binPath string) xrayRunner { return NewXrayRunner(binPath) }

// Pipeline runs the testing stages for a single config according to the chosen depth.
type Pipeline struct {
	Depth        models.TestDepth
	TCPTimeout   time.Duration
	TLSTimeout   time.Duration
	ProtoTimeout time.Duration
	XrayTimeout  time.Duration
	PortBase     int
	XrayBin      string
	// Domain check (full depth only): fetch these through the proxy and
	// require 2xx + DomainMinBytes; pass when >= DomainQuorum succeed.
	DomainURLs     []string
	DomainTimeout  time.Duration
	DomainMinBytes int64
	DomainQuorum   int
	// StabilityProbes is how many independent end-to-end verifications a
	// config must pass before it is reported as working. A single sample
	// admits nodes that fail most attempts (measured: 36% of a served set
	// failed at least one of three re-checks), which is the false-positive
	// problem. Values < 1 are treated as 1.
	StabilityProbes int
}

// NewPipeline creates a pipeline with defaults.
func NewPipeline(depth models.TestDepth) *Pipeline {
	return &Pipeline{
		Depth:        depth,
		TCPTimeout:   5 * time.Second,
		TLSTimeout:   5 * time.Second,
		ProtoTimeout: 5 * time.Second,
		XrayTimeout:  30 * time.Second,
		PortBase:     10820,
		XrayBin:      "xray",
		DomainURLs: []string{
			"https://youtube.com",
			"https://www.google.com",
			"https://www.cloudflare.com",
		},
		DomainTimeout:  5 * time.Second,
		DomainMinBytes: 4096,
		DomainQuorum:   2,
		// Two independent end-to-end verifications by default: one flaky
		// sample is not evidence of a usable tunnel.
		StabilityProbes: 2,
	}
}

// Budget returns a generous upper bound for one config's whole run. Every
// stage has its own timeout, so the caller's context must not be a single
// stage timeout: with the stability probes a full-depth config can legitimately
// need several proxy and domain passes, and a one-timeout wrapper silently
// cancelled the extra probes (they all failed with "context deadline
// exceeded", rejecting configs that had already passed once).
//
// Count: TCP + TLS + protocol + one end-to-end verification, plus one more
// end-to-end verification per extra stability probe, where an end-to-end
// verification costs a proxy attempt plus (at full depth) a domain pass.
func (p *Pipeline) Budget() time.Duration {
	probes := p.StabilityProbes
	if probes < 1 {
		probes = 1
	}
	units := 3 + probes // TCP, TLS, protocol + probes end-to-end verifications
	if p.normalizedDepth() == models.DepthFull && len(p.DomainURLs) > 0 {
		units += probes // a domain pass per verification
	}
	stage := p.XrayTimeout
	for _, d := range []time.Duration{p.TCPTimeout, p.TLSTimeout, p.ProtoTimeout} {
		if d > stage {
			stage = d
		}
	}
	if stage <= 0 {
		stage = 30 * time.Second
	}
	return time.Duration(units) * stage
}

// normalizedDepth collapses the deprecated comprehensive alias onto full.
func (p *Pipeline) normalizedDepth() models.TestDepth {
	if p.Depth == models.DepthComprehensive {
		return models.DepthFull
	}
	return p.Depth
}

// Run executes the test stages for cfg and returns a TestResult.
// Stages are gated by Depth (comprehensive is normalized to full):
//   - Quick:    TCP + TLS + Protocol (no subprocess spawned)
//   - Standard: Quick + Xray real-delay test (2xx required)
//   - Full:     Standard + multi-domain content check (quorum)
//
// When StabilityProbes > 1, every depth that reached StageProxy repeats the
// end-to-end verification at StageStability before reporting success.
//
// Latencies.Response is the MEDIAN of the end-to-end (proxied request)
// passes: that is the number a user feels, and it is what the served list is
// ordered by. Latencies.Total is the duration of the whole test run (every
// stage, every probe) and says nothing about the tunnel's speed.
func (p *Pipeline) Run(ctx context.Context, cfg models.ProxyConfig, port int) (res models.TestResult) {
	depth := p.normalizedDepth()
	res = models.TestResult{
		ID:        fmt.Sprintf("%s-%s", cfg.Protocol(), cfg.Addr()),
		Config:    cfg,
		Status:    models.StatusSuccess,
		Stage:     models.StageTCP,
		Timestamp: time.Now(),
		PortUsed:  port,
	}
	defer func() {
		// Total is the wall time of the whole run, on every exit path
		// (the early returns below used to leave it unset).
		res.Latencies.Total = time.Since(res.Timestamp)
	}()

	// Stage 1: TCP
	tcpRes := TestTCPForConfig(ctx, cfg, p.TCPTimeout)
	res.Latencies.Connect = tcpRes.Latency
	if !tcpRes.Success {
		res.Status = models.StatusFailed
		res.Errors = append(res.Errors, tcpRes.Error)
		return res
	}

	// Stage 2: TLS (if applicable)
	res.Stage = models.StageTLS
	hasTLS := p.needsTLS(cfg)
	if hasTLS {
		tlsRes := TestTLSForConfig(ctx, cfg, p.TLSTimeout)
		res.Latencies.TLS = tlsRes.Latency
		if !tlsRes.Success {
			res.Status = models.StatusFailed
			res.Errors = append(res.Errors, tlsRes.Error)
			return res
		}
	}

	// Stage 3: Protocol handshake (quick includes everything up to here)
	res.Stage = models.StageProtocol
	protoRes := TestProtocol(ctx, cfg, p.ProtoTimeout)
	res.Latencies.Handshake = protoRes.Latency
	if !protoRes.Success {
		res.Status = models.StatusFailed
		res.Errors = append(res.Errors, protoRes.Error)
		return res
	}

	if depth == models.DepthQuick {
		res.Stage = models.StageCompleted
		return res
	}

	// Stage 4: Xray real-delay test (standard and full)
	res.Stage = models.StageProxy
	xr := newXrayRunner(p.XrayBin)
	xrayRes := xr.TestXrayProxy(ctx, cfg, port, p.XrayTimeout)
	res.Latencies.Response = xrayRes.Latency
	if !xrayRes.Success {
		res.Status = models.StatusFailed
		res.Errors = append(res.Errors, xrayRes.Error)
		return res
	}
	// Every end-to-end pass measures the proxied request; the median of
	// those passes is the config's latency.
	endToEnd := []time.Duration{xrayRes.Latency}

	// Stage 5: multi-domain content check (FULL depth only, skipped when no
	// domains are configured). The xray instance from stage 4 is already
	// gone, so check inside a fresh temp xray.
	//
	// The stability stage below is NOT part of this depth gate: it used to
	// be, which left the DEFAULT depth (standard) with no repeat
	// verification at all — DAEMON_STABILITY_PROBES silently did nothing
	// there, and one sample is exactly the false-positive generator the
	// gate exists to remove.
	if depth == models.DepthFull && len(p.DomainURLs) > 0 {
		res.Stage = models.StageDomain
		var domRes DomainResult
		domErr := xr.withTempXray(ctx, cfg, port, p.XrayTimeout, func(proxyAddr string) error {
			domRes = CheckDomains(ctx, proxyAddr, p.DomainURLs, p.DomainTimeout, p.DomainMinBytes, p.DomainQuorum)
			if !domRes.Passed {
				return fmt.Errorf("domain check: %d/%d passed (quorum %d)",
					domRes.PassedCount, domRes.Checked, p.DomainQuorum)
			}
			return nil
		})
		if domErr != nil {
			res.Status = models.StatusFailed
			res.Errors = append(res.Errors, domErr.Error())
			return res
		}
	}

	// Stage 6: stability. Repeat the full end-to-end verification; a node that
	// passes once and fails on a repeat is not a usable tunnel. Measured on the
	// served set: 36% of links failed at least one of three independent
	// re-checks, so a single sample is not evidence of a working config.
	res.Stage = models.StageStability
	probeLatencies, err := runStabilityProbes(p.StabilityProbes, func() (time.Duration, error) {
		return p.verifyEndToEnd(ctx, cfg, port, depth)
	})
	if err != nil {
		res.Status = models.StatusFailed
		res.Errors = append(res.Errors, err.Error())
		return res
	}
	endToEnd = append(endToEnd, probeLatencies...)

	res.Stage = models.StageCompleted
	res.Latencies.Response = medianDuration(endToEnd)
	return res
}

// runStabilityProbes runs the extra end-to-end verifications and returns the
// latency each pass measured. The first verification was already performed by
// the caller, so probes=2 runs verify once more and probes<=1 runs it not at
// all. Any failure fails the config.
func runStabilityProbes(probes int, verify func() (time.Duration, error)) ([]time.Duration, error) {
	if probes < 1 {
		probes = 1
	}
	latencies := make([]time.Duration, 0, probes-1)
	for probe := 2; probe <= probes; probe++ {
		lat, err := verify()
		if err != nil {
			return latencies, fmt.Errorf("stability probe %d/%d: %w", probe, probes, err)
		}
		latencies = append(latencies, lat)
	}
	return latencies, nil
}

// medianDuration returns the median of the samples (0 when there are none).
// With an even number of samples the two middle values are averaged, so a
// single lucky fast pass does not outrank a consistently slow tunnel and a
// single slow pass does not condemn a fast one.
func medianDuration(samples []time.Duration) time.Duration {
	if len(samples) == 0 {
		return 0
	}
	sorted := make([]time.Duration, len(samples))
	copy(sorted, samples)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
}

// verifyEndToEnd performs one complete end-to-end verification: a proxied HTTP
// request that must return 2xx, plus the multi-domain content check when the
// depth enables it. It is used by the stability stage, which repeats it. The
// returned duration is the latency of the proxied request.
func (p *Pipeline) verifyEndToEnd(ctx context.Context, cfg models.ProxyConfig, port int, depth models.TestDepth) (time.Duration, error) {
	xr := newXrayRunner(p.XrayBin)
	xrayRes := xr.TestXrayProxy(ctx, cfg, port, p.XrayTimeout)
	if !xrayRes.Success {
		return xrayRes.Latency, fmt.Errorf("proxy test: %s", xrayRes.Error)
	}
	if depth != models.DepthFull || len(p.DomainURLs) == 0 {
		return xrayRes.Latency, nil
	}
	err := xr.withTempXray(ctx, cfg, port, p.XrayTimeout, func(proxyAddr string) error {
		domRes := CheckDomains(ctx, proxyAddr, p.DomainURLs, p.DomainTimeout, p.DomainMinBytes, p.DomainQuorum)
		if !domRes.Passed {
			return fmt.Errorf("domain check: %d/%d passed (quorum %d)",
				domRes.PassedCount, domRes.Checked, p.DomainQuorum)
		}
		return nil
	})
	return xrayRes.Latency, err
}

// needsTLS reports whether cfg has TLS enabled.
func (p *Pipeline) needsTLS(cfg models.ProxyConfig) bool {
	switch {
	case cfg.VMess != nil:
		return cfg.VMess.Enabled
	case cfg.VLess != nil:
		return cfg.VLess.Enabled
	case cfg.Trojan != nil:
		return cfg.Trojan.Enabled
	case cfg.Reality != nil:
		return true // Reality is always TLS-like
	case cfg.TUIC != nil:
		return cfg.TUIC.Enabled
	case cfg.Hysteria2 != nil:
		return cfg.Hysteria2.Enabled
	}
	return false
}

// ConfigPriority returns a numeric priority for sorting configs.
// Higher values = test first. Reality > VMess > VLess > Trojan > SS.
func ConfigPriority(cfg models.ProxyConfig) int {
	switch cfg.Protocol() {
	case models.ProtocolReality:
		return 5
	case models.ProtocolVMess:
		return 4
	case models.ProtocolVLess:
		return 3
	case models.ProtocolTrojan:
		return 2
	case models.ProtocolSS:
		return 1
	case models.ProtocolHysteria2:
		return 6
	case models.ProtocolTUIC:
		return 5
	case models.ProtocolWireGuard:
		return 3
	case models.ProtocolSocks5:
		return 1
	}
	return 0
}
