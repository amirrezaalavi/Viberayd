package daemon

import (
	"fmt"
	"net/http"
)

// handleMetrics serves Prometheus-format metrics (text exposition format,
// version 0.0.4) using only the standard library.
//
// Metrics exposed:
//
//	viberayd_configs_total{state="..."} — number of configs per state
//	viberayd_build_info{version="dev"}  — build information (always 1)
func (hs *httpServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	hs.daemon.StateMu.RLock()
	working := countByState(hs.daemon.State, StateWorking)
	failed := countByState(hs.daemon.State, StateFailed)
	unreachable := countByState(hs.daemon.State, StateUnreachable)
	unknown := countByState(hs.daemon.State, StateUnknown)
	hs.daemon.StateMu.RUnlock()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	fmt.Fprintln(w, "# HELP viberayd_configs_total Number of configs by state.")
	fmt.Fprintln(w, "# TYPE viberayd_configs_total gauge")
	fmt.Fprintf(w, "viberayd_configs_total{state=\"working\"} %d\n", working)
	fmt.Fprintf(w, "viberayd_configs_total{state=\"failed\"} %d\n", failed)
	fmt.Fprintf(w, "viberayd_configs_total{state=\"unreachable\"} %d\n", unreachable)
	fmt.Fprintf(w, "viberayd_configs_total{state=\"unknown\"} %d\n", unknown)

	fmt.Fprintln(w, "# HELP viberayd_build_info Build information.")
	fmt.Fprintln(w, "# TYPE viberayd_build_info gauge")
	fmt.Fprintln(w, `viberayd_build_info{version="dev"} 1`)
}
