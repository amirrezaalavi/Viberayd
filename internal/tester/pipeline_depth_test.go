package tester

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/amirrezaalavi/Viberayd/internal/models"
)

// localAcceptor opens a 127.0.0.1 TCP listener that accepts and immediately
// closes connections: enough for TCP-dial and Shadowsocks-protocol stages,
// which treat connect success as a pass. Hermetic — no external network.
func localAcceptor(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port
}

func localSSConfig(port int) models.ProxyConfig {
	return models.ProxyConfig{
		SS: &models.SSConfig{
			BaseConfig: models.BaseConfig{Server: "127.0.0.1", Port: port, Protocol: models.ProtocolSS},
			Method:     "aes-256-gcm",
			Password:   "test",
		},
	}
}

func quickPipeline() *Pipeline {
	p := NewPipeline(models.DepthQuick)
	p.TCPTimeout = 2 * time.Second
	p.ProtoTimeout = 2 * time.Second
	p.XrayBin = "/nonexistent-probe-binary"
	return p
}

// Quick must complete TCP+TLS+protocol without spawning xray: the bogus
// XrayBin would fail the run if quick ever attempted a spawn.
func TestPipeline_QuickCompletesWithoutXray(t *testing.T) {
	port := localAcceptor(t)
	p := quickPipeline()
	res := p.Run(context.Background(), localSSConfig(port), 18741)
	if res.Status != models.StatusSuccess {
		t.Fatalf("quick: status = %q errors %v, want success", res.Status, res.Errors)
	}
	if res.Stage != models.StageCompleted {
		t.Fatalf("quick: stage = %q, want completed", res.Stage)
	}
}

// Quick stops at TCP when the port refuses: no TLS/protocol/xray attempted.
func TestPipeline_QuickTCPRefused(t *testing.T) {
	p := quickPipeline()
	res := p.Run(context.Background(), localSSConfig(1), 18742) // nothing on port 1
	if res.Status != models.StatusFailed {
		t.Fatalf("quick refused: status = %q, want failed", res.Status)
	}
	if res.Stage != models.StageTCP {
		t.Fatalf("quick refused: stage = %q, want tcp", res.Stage)
	}
}

// Standard must attempt the xray stage: the bogus binary fails it fast at
// StageProxy instead of completing.
func TestPipeline_StandardAttemptsXray(t *testing.T) {
	port := localAcceptor(t)
	p := NewPipeline(models.DepthStandard)
	p.TCPTimeout = 2 * time.Second
	p.ProtoTimeout = 2 * time.Second
	p.XrayTimeout = 5 * time.Second
	p.XrayBin = "/nonexistent-probe-binary"
	res := p.Run(context.Background(), localSSConfig(port), 18743)
	if res.Status != models.StatusFailed {
		t.Fatalf("standard: status = %q, want failed (bad xray bin)", res.Status)
	}
	if res.Stage != models.StageProxy {
		t.Fatalf("standard: stage = %q, want proxy", res.Stage)
	}
	if len(res.Errors) == 0 || !strings.Contains(res.Errors[0], "start xray") {
		t.Fatalf("standard: errors = %v, want start-xray failure", res.Errors)
	}
}

// Full attempts xray before domains: with a bogus binary it must fail at
// StageProxy, never reaching the domain stage.
func TestPipeline_FullAttemptsXrayFirst(t *testing.T) {
	port := localAcceptor(t)
	p := NewPipeline(models.DepthFull)
	p.TCPTimeout = 2 * time.Second
	p.ProtoTimeout = 2 * time.Second
	p.XrayTimeout = 5 * time.Second
	p.XrayBin = "/nonexistent-probe-binary"
	res := p.Run(context.Background(), localSSConfig(port), 18744)
	if res.Status != models.StatusFailed {
		t.Fatalf("full: status = %q, want failed (bad xray bin)", res.Status)
	}
	if res.Stage != models.StageProxy {
		t.Fatalf("full: stage = %q, want proxy (domains come after xray)", res.Stage)
	}
}

// Comprehensive is a deprecated alias of full: identical gating behavior.
func TestPipeline_ComprehensiveAliasFull(t *testing.T) {
	port := localAcceptor(t)
	p := NewPipeline(models.DepthComprehensive)
	p.TCPTimeout = 2 * time.Second
	p.ProtoTimeout = 2 * time.Second
	p.XrayTimeout = 5 * time.Second
	p.XrayBin = "/nonexistent-probe-binary"
	res := p.Run(context.Background(), localSSConfig(port), 18745)
	if res.Status != models.StatusFailed || res.Stage != models.StageProxy {
		t.Fatalf("comprehensive: status = %q stage = %q, want failed/proxy like full",
			res.Status, res.Stage)
	}
}
