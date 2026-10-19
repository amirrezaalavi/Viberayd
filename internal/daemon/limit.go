package daemon

import (
	"log/slog"
	"sort"
)

// limitSurvivors caps how many TCP-reachable configs go on to the expensive
// stages (real proxy test, domain/content check, stability probes) in one
// cycle, keeping the fastest TCP handshakes.
//
// Why a cap: the TCP pre-filter is cheap (parallel dials, no subprocess) and
// rejects most of a large corpus, but every survivor costs a spawned xray, a
// listener wait and one or more proxied requests. A single dead-but-reachable
// path can burn the whole per-config budget (tens of seconds), so an uncapped
// survivor set makes cycle duration unbounded — measured on a ~650k-candidate
// corpus: the cycle was still running after 30 minutes and nothing new was
// published. A cap makes the cycle predictable and spends the expensive work
// on the configs most likely to be usable (lowest handshake latency first).
//
// A limit <= 0 disables the cap. The input slice is sorted in place; the
// returned slice shares its backing array.
func limitSurvivors(survivors []NamedConfig, tcp map[string]TCPResult, limit int) []NamedConfig {
	if limit <= 0 || len(survivors) <= limit {
		return survivors
	}

	sort.SliceStable(survivors, func(i, j int) bool {
		li, iok := tcp[survivors[i].Hash]
		lj, jok := tcp[survivors[j].Hash]
		// A survivor without a TCP result is kept last: it cannot be ranked,
		// and it should not displace a config we measured.
		if iok != jok {
			return iok
		}
		if li.Latency != lj.Latency {
			return li.Latency < lj.Latency
		}
		return survivors[i].Hash < survivors[j].Hash
	})

	kept := survivors[:limit]
	slowest := kept[len(kept)-1]
	if tr, ok := tcp[slowest.Hash]; ok {
		slog.Info("xray test capped",
			"survivors", len(survivors),
			"cap", limit,
			"dropped", len(survivors)-limit,
			"slowest_kept_ms", tr.Latency,
		)
	} else {
		slog.Info("xray test capped",
			"survivors", len(survivors),
			"cap", limit,
			"dropped", len(survivors)-limit,
		)
	}
	return kept
}
