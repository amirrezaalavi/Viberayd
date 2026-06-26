package output

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"

	"github.com/amiralavi/viberay/internal/models"
)

// CSVFormatter renders a minimal comma-separated summary for batch processing.
type CSVFormatter struct{}

func (f *CSVFormatter) Format(w io.Writer, results []models.TestResult, summary models.Summary) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	// Header
	if err := cw.Write([]string{
		"protocol", "server", "port", "name", "status",
		"stage", "latency_ms", "errors", "retries",
	}); err != nil {
		return err
	}

	for _, r := range results {
		b := r.Config.Base()
		errStr := ""
		if len(r.Errors) > 0 {
			errStr = r.Errors[0]
		}
		row := []string{
			string(r.Config.Protocol()),
			b.Server,
			strconv.Itoa(b.Port),
			b.Name,
			string(r.Status),
			r.Stage,
			fmt.Sprintf("%.1f", float64(r.Latencies.Total.Microseconds())/1000),
			errStr,
			strconv.Itoa(r.Retries),
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}

	// Summary row
	_ = cw.Write([]string{
		"", "", "", "SUMMARY", "",
		fmt.Sprintf("total=%d passed=%d failed=%d errors=%d", summary.Total, summary.Passed, summary.Failed, summary.Errors),
		fmt.Sprintf("%.1f", summary.AvgLatencyMs),
		fmt.Sprintf("%.1f%%", summary.SuccessRatePct),
		fmt.Sprintf("%.2f/s", summary.ConfigsPerSecond),
	})

	return nil
}
