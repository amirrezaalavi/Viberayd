package output

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/amiralavi/viberay/internal/models"
)

// MarkdownFormatter renders a documentation-friendly report.
type MarkdownFormatter struct{}

func (f *MarkdownFormatter) Format(w io.Writer, results []models.TestResult, summary models.Summary) error {
	fmt.Fprintln(w, "# VibeRay Test Report")
	fmt.Fprintf(w, "\nGenerated: %s\n\n", time.Now().Format(time.RFC3339))

	// Summary table
	fmt.Fprintln(w, "## Summary")
	fmt.Fprintln(w, "| Metric | Value |")
	fmt.Fprintln(w, "|--------|-------|")
	fmt.Fprintf(w, "| Total Configs | %d |\n", summary.Total)
	fmt.Fprintf(w, "| Passed | %d |\n", summary.Passed)
	fmt.Fprintf(w, "| Failed | %d |\n", summary.Failed)
	fmt.Fprintf(w, "| Errors | %d |\n", summary.Errors)
	fmt.Fprintf(w, "| Skipped | %d |\n", summary.Skipped)
	fmt.Fprintf(w, "| Success Rate | %.1f%% |\n", summary.SuccessRatePct)
	fmt.Fprintf(w, "| Avg Latency | %.1f ms |\n", summary.AvgLatencyMs)
	fmt.Fprintf(w, "| Throughput | %.2f configs/s |\n", summary.ConfigsPerSecond)
	fmt.Fprintf(w, "| Duration | %s |\n", summary.Duration.Round(time.Millisecond))

	// Protocol distribution
	fmt.Fprintln(w, "\n## Protocol Distribution")
	fmt.Fprintln(w, "| Protocol | Count |")
	fmt.Fprintln(w, "|----------|-------|")
	for proto, count := range summary.ByProtocol {
		fmt.Fprintf(w, "| %s | %d |\n", proto, count)
	}

	// Detailed results
	fmt.Fprintln(w, "\n## Detailed Results")
	fmt.Fprintln(w, "| Protocol | Server | Port | Status | Stage | Latency | Errors |")
	fmt.Fprintln(w, "|----------|--------|------|--------|-------|---------|--------|")
	for _, r := range results {
		b := r.Config.Base()
		latency := "-"
		if r.Latencies.Total > 0 {
			latency = r.Latencies.Total.Round(time.Millisecond).String()
		}
		errs := strings.Join(r.Errors, "; ")
		if errs == "" {
			errs = "-"
		}
		fmt.Fprintf(w, "| %s | %s | %d | %s | %s | %s | %s |\n",
			string(r.Config.Protocol()),
			b.Server,
			b.Port,
			r.Status,
			r.Stage,
			latency,
			errs,
		)
	}

	return nil
}
