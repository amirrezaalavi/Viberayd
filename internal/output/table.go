package output

import (
	"fmt"
	"io"
	"time"

	"github.com/amiralavi/viberay/internal/models"
)

// TableFormatter outputs working configs as their original share link plus latency.
type TableFormatter struct{}

func (f *TableFormatter) Format(w io.Writer, results []models.TestResult, summary models.Summary) error {
	for _, r := range results {
		if r.Status != models.StatusSuccess {
			continue
		}

		latency := r.Latencies.Total
		if latency == 0 {
			latency = r.Latencies.Connect
		}

		fmt.Fprintf(w, "%s %s\n", r.Config.Raw, latency.Round(time.Millisecond))
	}

	return nil
}
