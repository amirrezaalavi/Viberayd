package output

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/amiralavi/viberay/internal/models"
)

// TableFormatter renders human-readable aligned columns.
type TableFormatter struct{}

func (f *TableFormatter) Format(w io.Writer, results []models.TestResult, summary models.Summary) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	fmt.Fprintf(tw, "\n%-10s %-22s %-8s %-10s %-12s %-10s %s\n",
		"PROTOCOL", "SERVER", "PORT", "STATUS", "STAGE", "LATENCY", "NAME")
	fmt.Fprintln(tw, strings.Repeat("-", 90))

	for _, r := range results {
		b := r.Config.Base()
		status := string(r.Status)
		if r.Status == models.StatusSuccess {
			status = "PASS"
		} else if r.Status == models.StatusFailed {
			status = "FAIL"
		}

		latency := "-"
		if r.Latencies.Total > 0 {
			latency = r.Latencies.Total.Round(100 * time.Microsecond).String()
		}

		name := b.Name
		if len(name) > 20 {
			name = name[:17] + "..."
		}

		fmt.Fprintf(tw, "%-10s %-22s %-8d %-10s %-12s %-10s %s\n",
			string(r.Config.Protocol()),
			trunc(b.Server, 22),
			b.Port,
			status,
			r.Stage,
			latency,
			name,
		)
	}

	fmt.Fprintln(tw, strings.Repeat("-", 90))
	fmt.Fprintf(tw, "Summary: %d total | %d passed | %d failed | %d errors | %.1f%% success | %.2f configs/s\n",
		summary.Total, summary.Passed, summary.Failed, summary.Errors,
		summary.SuccessRatePct, summary.ConfigsPerSecond,
	)

	return tw.Flush()
}

func trunc(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
