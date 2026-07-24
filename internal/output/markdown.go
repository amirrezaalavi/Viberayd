package output

import (
	"fmt"
	"io"
	"time"

	"github.com/amirrezaalavi/Viberay/internal/models"
)

// MarkdownFormatter renders working configs with their original share link and latency.
type MarkdownFormatter struct{}

func (f *MarkdownFormatter) Format(w io.Writer, results []models.TestResult, summary models.Summary) error {
	fmt.Fprintln(w, "# VibeRay Working Configs")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Share Link | Latency |")
	fmt.Fprintln(w, "|------------|---------|")

	for _, r := range results {
		if r.Status != models.StatusSuccess {
			continue
		}
		latency := r.Latencies.Total
		if latency == 0 {
			latency = r.Latencies.Connect
		}
		fmt.Fprintf(w, "| `%s` | %s |\n", r.Config.Raw, latency.Round(time.Millisecond))
	}

	return nil
}
