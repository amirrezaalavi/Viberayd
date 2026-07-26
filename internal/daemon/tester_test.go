package daemon

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/amirrezaalavi/Viberay/internal/models"
)

func TestTCPPingReachable(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	addr := ln.Addr().String()
	host, portStr, _ := net.SplitHostPort(addr)

	go func() {
		conn, _ := ln.Accept()
		if conn != nil {
			conn.Close()
		}
	}()

	candidates := map[string]models.ProxyConfig{
		"test1": {
			VMess: &models.VMessConfig{
				BaseConfig: models.BaseConfig{
					Server: host,
					Port:   atoi(portStr),
				},
			},
		},
	}

	results := TCPPing(context.Background(), candidates, 2*time.Second)

	res, ok := results["test1"]
	if !ok {
		t.Fatal("missing result for test1")
	}
	if !res.Success {
		t.Errorf("expected success, got failure")
	}
}

func TestTCPPingUnreachable(t *testing.T) {
	candidates := map[string]models.ProxyConfig{
		"dead": {
			VMess: &models.VMessConfig{
				BaseConfig: models.BaseConfig{
					Server: "127.0.0.1",
					Port:   1,
				},
			},
		},
	}

	results := TCPPing(context.Background(), candidates, 500*time.Millisecond)

	res, ok := results["dead"]
	if !ok {
		t.Fatal("missing result for dead")
	}
	if res.Success {
		t.Errorf("expected failure for port 1")
	}
}

func TestTCPPingEmpty(t *testing.T) {
	results := TCPPing(context.Background(), map[string]models.ProxyConfig{}, 1*time.Second)
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}

func TestTCPPingCanceledContext(t *testing.T) {
	candidates := map[string]models.ProxyConfig{
		"a": {
			SS: &models.SSConfig{
				BaseConfig: models.BaseConfig{
					Server: "127.0.0.1",
					Port:   1,
				},
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	results := TCPPing(ctx, candidates, 5*time.Second)
	if len(results) != 0 {
		t.Errorf("got %d results, want 0 (cancelled context does no work)", len(results))
	}
}

func TestXrayTestEmpty(t *testing.T) {
	results := XrayTest(context.Background(), nil, XrayTestConfig{
		Parallel: 5,
		Timeout:  1 * time.Second,
		Depth:    models.DepthQuick,
	})
	if results != nil {
		t.Errorf("expected nil, got %d results", len(results))
	}
}

func TestXrayTestNoConfigs(t *testing.T) {
	results := XrayTest(context.Background(), []NamedConfig{}, XrayTestConfig{
		Parallel: 5,
		Timeout:  1 * time.Second,
		Depth:    models.DepthQuick,
	})
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}

func TestXrayTestQuickDepth(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	results := XrayTest(ctx, []NamedConfig{
		{
			Hash: "dead1",
			Cfg: models.ProxyConfig{
				Trojan: &models.TrojanConfig{
					BaseConfig: models.BaseConfig{
						Server: "127.0.0.1",
						Port:   1,
					},
					Password: "test",
				},
			},
		},
	}, XrayTestConfig{
		Parallel: 5,
		Timeout:  1 * time.Second,
		Depth:    models.DepthQuick,
		PortBase: 11820,
	})

	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Success {
		t.Errorf("expected failure for dead config")
	}
	if results[0].Hash != "dead1" {
		t.Errorf("Hash = %q, want dead1", results[0].Hash)
	}
}

func TestXrayTestMultipleConfigs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	configs := []NamedConfig{
		{
			Hash: "a",
			Cfg: models.ProxyConfig{
				Trojan: &models.TrojanConfig{
					BaseConfig: models.BaseConfig{Server: "127.0.0.1", Port: 1},
					Password:   "p1",
				},
			},
		},
		{
			Hash: "b",
			Cfg: models.ProxyConfig{
				VMess: &models.VMessConfig{
					BaseConfig: models.BaseConfig{Server: "127.0.0.1", Port: 2},
					UUID:       "00000000-0000-0000-0000-000000000000",
				},
			},
		},
	}

	results := XrayTest(ctx, configs, XrayTestConfig{
		Parallel: 2,
		Timeout:  1 * time.Second,
		Depth:    models.DepthQuick,
		PortBase: 11920,
	})

	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}

	byHash := make(map[string]TestResult)
	for _, r := range results {
		byHash[r.Hash] = r
	}

	for _, h := range []string{"a", "b"} {
		r, ok := byHash[h]
		if !ok {
			t.Errorf("missing result for %s", h)
			continue
		}
		if r.Success {
			t.Errorf("%s: expected failure", h)
		}
	}
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}
