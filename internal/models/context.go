package models

import "time"

// TestDepth controls how thoroughly each configuration is tested.
// Ladder (each level includes everything above it):
// quick    = TCP dial + TLS handshake + protocol probe (no subprocess)
// standard = quick + xray real-delay test (TTFB through the proxy, 2xx required)
// full     = standard + multi-domain content check (quorum over real websites)
type TestDepth string

const (
	DepthQuick    TestDepth = "quick"    // TCP + TLS + Protocol
	DepthStandard TestDepth = "standard" // Quick + xray real-delay
	DepthFull     TestDepth = "full"     // Standard + domain content check
	// DepthComprehensive is a deprecated alias of DepthFull, kept so existing
	// DAEMON_DEPTH=comprehensive deployments keep working. New code must use
	// DepthFull; the pipeline normalizes this before gating.
	DepthComprehensive TestDepth = "comprehensive" // alias of full (deprecated)
)

// IsValid reports whether d is a known test depth.
func (d TestDepth) IsValid() bool {
	switch d {
	case DepthQuick, DepthStandard, DepthFull, DepthComprehensive:
		return true
	}
	return false
}

// OutputStyle controls the formatting of test results.
type OutputStyle string

const (
	StyleAuto     OutputStyle = "auto"
	StyleJSON     OutputStyle = "json"
	StyleCSV      OutputStyle = "csv"
	StyleTable    OutputStyle = "table"
	StyleMarkdown OutputStyle = "markdown"
	StyleHTML     OutputStyle = "html"
)

// IsValid reports whether s is a known output style.
func (s OutputStyle) IsValid() bool {
	switch s {
	case StyleAuto, StyleJSON, StyleCSV, StyleTable, StyleMarkdown, StyleHTML:
		return true
	}
	return false
}

// CacheStats tracks cache hit/miss performance.
type CacheStats struct {
	DNSEntries   int `json:"dns_entries" yaml:"dns_entries"`
	TLSSessions  int `json:"tls_sessions" yaml:"tls_sessions"`
	Results      int `json:"results" yaml:"results"`
	Hits         int `json:"hits" yaml:"hits"`
	Misses       int `json:"misses" yaml:"misses"`
	Evictions    int `json:"evictions" yaml:"evictions"`
}

// SystemInfo describes the runtime environment.
type SystemInfo struct {
	CPUCount    int    `json:"cpu_cores" yaml:"cpu_cores"`
	MemoryMB    int64  `json:"memory_mb" yaml:"memory_mb"`
	OS          string `json:"os" yaml:"os"`
	Arch        string `json:"arch" yaml:"arch"`
	XrayVersion string `json:"xray_version,omitempty" yaml:"xray_version,omitempty"`
	GoVersion   string `json:"go_version" yaml:"go_version"`
}

// InputStats describes the parsed input.
type InputStats struct {
	Total               int            `json:"total" yaml:"total"`
	ProtocolDistribution map[string]int `json:"protocol_distribution" yaml:"protocol_distribution"`
	Duplicates          int            `json:"duplicates" yaml:"duplicates"`
	ParseErrors         int            `json:"parse_errors" yaml:"parse_errors"`
}

// RuntimeState tracks the execution progress.
type RuntimeState struct {
	StartTime   time.Time     `json:"start_time" yaml:"start_time"`
	EndTime     *time.Time    `json:"end_time,omitempty" yaml:"end_time,omitempty"`
	Duration    time.Duration `json:"duration" yaml:"duration"`
	ActiveTests int           `json:"active_tests" yaml:"active_tests"`
	Completed   int           `json:"completed" yaml:"completed"`
}

// History tracks prior runs and cache status.
type History struct {
	PreviousRuns int        `json:"previous_runs" yaml:"previous_runs"`
	CacheStats   CacheStats `json:"cache_stats" yaml:"cache_stats"`
}

// TestContext is the central state container for the AI orchestrator.
type TestContext struct {
	System  SystemInfo   `json:"system" yaml:"system"`
	Input   InputStats   `json:"input" yaml:"input"`
	Runtime RuntimeState `json:"runtime" yaml:"runtime"`
	History History      `json:"history,omitempty" yaml:"history,omitempty"`
}

// ConcurrencySettings holds the resolved worker-pool parameters.
type ConcurrencySettings struct {
	Workers     int `json:"workers" yaml:"workers"`
	MaxParallel int `json:"max_parallel" yaml:"max_parallel"`
}

// RetryPolicy holds the resolved retry behaviour.
type RetryPolicy struct {
	MaxRetries  int           `json:"max_retries" yaml:"max_retries"`
	BackoffBase time.Duration `json:"backoff_base" yaml:"backoff_base"`
}

// OrchestratorDecision captures the AI-decided parameters for a run.
type OrchestratorDecision struct {
	Depth       TestDepth           `json:"depth" yaml:"depth"`
	Style       OutputStyle         `json:"style" yaml:"style"`
	Concurrency ConcurrencySettings `json:"concurrency" yaml:"concurrency"`
	Retry       RetryPolicy         `json:"retry" yaml:"retry"`
	CacheEnabled bool               `json:"cache_enabled" yaml:"cache_enabled"`
	TimeoutPerTest time.Duration    `json:"timeout_per_test" yaml:"timeout_per_test"`
}
