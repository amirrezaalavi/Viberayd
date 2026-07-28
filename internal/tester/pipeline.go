package tester

import (
	"context"
	"fmt"
	"time"

	"github.com/amirrezaalavi/Viberay/internal/models"
)

// Pipeline runs the testing stages for a single config according to the chosen depth.
type Pipeline struct {
	Depth       models.TestDepth
	TCPTimeout  time.Duration
	TLSTimeout  time.Duration
	ProtoTimeout time.Duration
	XrayTimeout time.Duration
	PortBase    int
	XrayBin     string
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
	}
}

// Run executes the test stages for cfg and returns a TestResult.
// Stages are gated by Depth:
//   - Quick:       TCP only
//   - Standard:    TCP + TLS
//   - Full:        TCP + TLS + Protocol
//   - Comprehensive: TCP + TLS + Protocol + Xray proxy test
func (p *Pipeline) Run(ctx context.Context, cfg models.ProxyConfig, port int) models.TestResult {
	res := models.TestResult{
		ID:        fmt.Sprintf("%s-%s", cfg.Protocol(), cfg.Addr()),
		Config:    cfg,
		Status:    models.StatusSuccess,
		Stage:     models.StageTCP,
		Timestamp: time.Now(),
		PortUsed:  port,
	}

	// Stage 1: TCP
	tcpRes := TestTCPForConfig(ctx, cfg, p.TCPTimeout)
	res.Latencies.Connect = tcpRes.Latency
	if !tcpRes.Success {
		res.Status = models.StatusFailed
		res.Errors = append(res.Errors, tcpRes.Error)
		return res
	}

	if p.Depth == models.DepthQuick {
		res.Stage = models.StageCompleted
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

	if p.Depth == models.DepthStandard {
		res.Stage = models.StageCompleted
		return res
	}

	// Stage 3: Protocol handshake
	res.Stage = models.StageProtocol
	protoRes := TestProtocol(ctx, cfg, p.ProtoTimeout)
	res.Latencies.Handshake = protoRes.Latency
	if !protoRes.Success {
		res.Status = models.StatusFailed
		res.Errors = append(res.Errors, protoRes.Error)
		return res
	}

	if p.Depth == models.DepthFull {
		res.Stage = models.StageCompleted
		return res
	}

	// Stage 4: Xray proxy test (comprehensive)
	res.Stage = models.StageProxy
	xr := NewXrayRunner(p.XrayBin)
	xrayRes := xr.TestXrayProxy(ctx, cfg, port, p.XrayTimeout)
	res.Latencies.Response = xrayRes.Latency
	if !xrayRes.Success {
		res.Status = models.StatusFailed
		res.Errors = append(res.Errors, xrayRes.Error)
		return res
	}

	res.Stage = models.StageCompleted
	res.Latencies.Total = time.Since(res.Timestamp)
	return res
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
