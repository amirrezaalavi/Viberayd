package daemon

import (
	"testing"

	"github.com/amirrezaalavi/Viberayd/internal/models"
)

func survivorSet(hashes ...string) []NamedConfig {
	out := make([]NamedConfig, 0, len(hashes))
	for _, h := range hashes {
		out = append(out, NamedConfig{Hash: h})
	}
	return out
}

func hashesOf(cfgs []NamedConfig) []string {
	out := make([]string, len(cfgs))
	for i, c := range cfgs {
		out[i] = c.Hash
	}
	return out
}

func TestLimitSurvivorsDisabled(t *testing.T) {
	for _, limit := range []int{0, -1} {
		in := survivorSet("a", "b", "c")
		got := limitSurvivors(in, map[string]TCPResult{}, limit)
		if len(got) != 3 {
			t.Fatalf("limit=%d: len = %d, want 3 (cap disabled)", limit, len(got))
		}
	}
}

func TestLimitSurvivorsNoopWhenUnderCap(t *testing.T) {
	in := survivorSet("a", "b")
	got := limitSurvivors(in, map[string]TCPResult{
		"a": {Hash: "a", Success: true, Latency: 30},
		"b": {Hash: "b", Success: true, Latency: 10},
	}, 5)
	if want := []string{"a", "b"}; len(got) != 2 || got[0].Hash != want[0] || got[1].Hash != want[1] {
		t.Fatalf("got %v, want unchanged order %v", hashesOf(got), want)
	}
}

func TestLimitSurvivorsKeepsFastest(t *testing.T) {
	in := survivorSet("slow", "fast", "mid", "fastest")
	tcp := map[string]TCPResult{
		"slow":    {Hash: "slow", Success: true, Latency: 900},
		"fast":    {Hash: "fast", Success: true, Latency: 40},
		"mid":     {Hash: "mid", Success: true, Latency: 200},
		"fastest": {Hash: "fastest", Success: true, Latency: 5},
	}
	got := limitSurvivors(in, tcp, 2)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	want := []string{"fastest", "fast"}
	for i, w := range want {
		if got[i].Hash != w {
			t.Fatalf("got %v, want %v (fastest first)", hashesOf(got), want)
		}
	}
}

func TestLimitSurvivorsMissingTCPResultGoesLast(t *testing.T) {
	in := survivorSet("unknown1", "known")
	tcp := map[string]TCPResult{
		"known": {Hash: "known", Success: true, Latency: 500},
	}
	got := limitSurvivors(in, tcp, 1)
	if len(got) != 1 || got[0].Hash != "known" {
		t.Fatalf("got %v, want [known]: a ranked survivor must outrank an unranked one", hashesOf(got))
	}
}

func TestLimitSurvivorsTieIsDeterministic(t *testing.T) {
	in := survivorSet("b", "a", "c")
	tcp := map[string]TCPResult{
		"a": {Hash: "a", Success: true, Latency: 7},
		"b": {Hash: "b", Success: true, Latency: 7},
		"c": {Hash: "c", Success: true, Latency: 7},
	}
	got := limitSurvivors(in, tcp, 2)
	if got[0].Hash != "a" || got[1].Hash != "b" {
		t.Fatalf("got %v, want deterministic hash order [a b]", hashesOf(got))
	}
}

func TestLimitSurvivorsKeepsConfig(t *testing.T) {
	in := []NamedConfig{{Hash: "x", Cfg: models.ProxyConfig{Raw: "vless://x@example.com:443"}}}
	got := limitSurvivors(in, map[string]TCPResult{"x": {Hash: "x", Success: true, Latency: 1}}, 1)
	if got[0].Cfg.Raw != "vless://x@example.com:443" {
		t.Fatalf("config payload lost: %+v", got[0].Cfg)
	}
}
