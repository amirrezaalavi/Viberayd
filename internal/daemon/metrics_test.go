package daemon

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleMetrics(t *testing.T) {
	s := NewState()
	s.Configs["a"] = &ConfigEntry{State: StateWorking}
	s.Configs["b"] = &ConfigEntry{State: StateWorking}
	s.Configs["c"] = &ConfigEntry{State: StateFailed}
	s.Configs["d"] = &ConfigEntry{State: StateFailed}
	s.Configs["e"] = &ConfigEntry{State: StateFailed}
	s.Configs["f"] = &ConfigEntry{State: StateUnreachable}
	s.Configs["g"] = &ConfigEntry{State: StateUnknown}

	d := &Daemon{State: s}
	hs := &httpServer{daemon: d}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	hs.handleMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/plain; version=0.0.4") {
		t.Errorf("Content-Type = %q, want text/plain; version=0.0.4", ct)
	}

	body := w.Body.String()

	// Metric names and HELP/TYPE lines present.
	for _, want := range []string{
		"# HELP viberayd_configs_total",
		"# TYPE viberayd_configs_total gauge",
		"# HELP viberayd_build_info",
		"# TYPE viberayd_build_info gauge",
		`viberayd_build_info{version="dev"} 1`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q in:\n%s", want, body)
		}
	}

	// Per-state counts match the fixture.
	for _, want := range []string{
		`viberayd_configs_total{state="working"} 2`,
		`viberayd_configs_total{state="failed"} 3`,
		`viberayd_configs_total{state="unreachable"} 1`,
		`viberayd_configs_total{state="unknown"} 1`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q in:\n%s", want, body)
		}
	}
}

func TestHandleMetricsEmptyState(t *testing.T) {
	d := &Daemon{State: NewState()}
	hs := &httpServer{daemon: d}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	hs.handleMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	body := w.Body.String()
	for _, want := range []string{
		`viberayd_configs_total{state="working"} 0`,
		`viberayd_configs_total{state="failed"} 0`,
		`viberayd_configs_total{state="unreachable"} 0`,
		`viberayd_configs_total{state="unknown"} 0`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q in:\n%s", want, body)
		}
	}
}

func TestHandleMetricsMethodNotAllowed(t *testing.T) {
	hs := &httpServer{daemon: &Daemon{State: NewState()}}
	req := httptest.NewRequest(http.MethodPost, "/metrics", nil)
	w := httptest.NewRecorder()
	hs.handleMetrics(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", w.Code)
	}
}
