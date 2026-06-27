package output

import (
	"encoding/csv"
	"io"
	"time"

	"github.com/amiralavi/viberay/internal/models"
)

// CSVFormatter renders working configs as their original share link plus latency.
type CSVFormatter struct{}

func (f *CSVFormatter) Format(w io.Writer, results []models.TestResult, summary models.Summary) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	for _, r := range results {
		if r.Status != models.StatusSuccess {
			continue
		}

		latency := r.Latencies.Total
		if latency == 0 {
			latency = r.Latencies.Connect
		}

		row := []string{r.Config.Raw, latency.Round(time.Millisecond).String()}
		if err := cw.Write(row); err != nil {
			return err
		}
	}

	return nil
}
