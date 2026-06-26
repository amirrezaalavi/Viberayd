package models

import (
	"fmt"
	"time"
)

// Stage names for the testing pipeline.
const (
	StageTCP       = "tcp"
	StageTLS       = "tls"
	StageProtocol  = "protocol"
	StageProxy     = "proxy"
	StageCompleted = "completed"
)

// TestStatus indicates the outcome of a test.
type TestStatus string

const (
	StatusSuccess TestStatus = "success"
	StatusFailed  TestStatus = "failed"
	StatusError   TestStatus = "error"
	StatusSkipped TestStatus = "skipped"
)

// LatencyBreakdown holds timing for each test stage.
type LatencyBreakdown struct {
	Connect   time.Duration `json:"connect,omitempty" yaml:"connect,omitempty"`
	TLS       time.Duration `json:"tls,omitempty" yaml:"tls,omitempty"`
	Handshake time.Duration `json:"handshake,omitempty" yaml:"handshake,omitempty"`
	Response  time.Duration `json:"response,omitempty" yaml:"response,omitempty"`
	Total     time.Duration `json:"total,omitempty" yaml:"total,omitempty"`
}

// TestMetrics holds optional performance measurements.
type TestMetrics struct {
	BandwidthMbps float64       `json:"bandwidth_mbps,omitempty" yaml:"bandwidth_mbps,omitempty"`
	PacketLossPct float64       `json:"packet_loss_percent,omitempty" yaml:"packet_loss_percent,omitempty"`
	Jitter        time.Duration `json:"jitter,omitempty" yaml:"jitter,omitempty"`
}

// TestResult captures the outcome of testing a single configuration.
type TestResult struct {
	ID        string             `json:"id" yaml:"id"`                              // unique per-config identifier
	Config    ProxyConfig        `json:"config" yaml:"config"`
	Status    TestStatus         `json:"status" yaml:"status"`
	Stage     string             `json:"stage" yaml:"stage"`                        // last stage attempted
	Latencies LatencyBreakdown   `json:"latencies" yaml:"latencies"`
	Errors    []string           `json:"errors,omitempty" yaml:"errors,omitempty"`
	Metrics   TestMetrics        `json:"metrics,omitempty" yaml:"metrics,omitempty"`
	Timestamp time.Time          `json:"timestamp" yaml:"timestamp"`
	TestID    string             `json:"test_id" yaml:"test_id"`                    // run-level identifier
	PortUsed  int                `json:"port_used,omitempty" yaml:"port_used,omitempty"`
	Retries   int                `json:"retries,omitempty" yaml:"retries,omitempty"`
}

// Summary aggregates results across an entire test run.
type Summary struct {
	Total            int            `json:"total" yaml:"total"`
	Passed           int            `json:"passed" yaml:"passed"`
	Failed           int            `json:"failed" yaml:"failed"`
	Errors           int            `json:"errors" yaml:"errors"`
	Skipped          int            `json:"skipped" yaml:"skipped"`
	ByProtocol       map[string]int `json:"by_protocol" yaml:"by_protocol"`
	AvgLatencyMs     float64        `json:"avg_latency_ms" yaml:"avg_latency_ms"`
	SuccessRatePct   float64        `json:"success_rate_percent" yaml:"success_rate_percent"`
	Duration         time.Duration  `json:"duration" yaml:"duration"`
	ConfigsPerSecond float64        `json:"configs_per_second" yaml:"configs_per_second"`
}

// String returns a human-readable summary line.
func (s Summary) String() string {
	return fmt.Sprintf(
		"Summary: %d total, %d passed, %d failed, %d errors, %.1f%% success rate, %.2f configs/s",
		s.Total, s.Passed, s.Failed, s.Errors, s.SuccessRatePct, s.ConfigsPerSecond,
	)
}

// ValidationResult holds parse-time validation state.
type ValidationResult struct {
	Valid    bool     `json:"valid" yaml:"valid"`
	Errors   []string `json:"errors,omitempty" yaml:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty" yaml:"warnings,omitempty"`
}

// AddError records a validation error and marks the result invalid.
func (vr *ValidationResult) AddError(msg string) {
	vr.Valid = false
	vr.Errors = append(vr.Errors, msg)
}

// AddWarning records a non-fatal warning.
func (vr *ValidationResult) AddWarning(msg string) {
	vr.Warnings = append(vr.Warnings, msg)
}
