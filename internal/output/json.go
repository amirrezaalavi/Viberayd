package output

import (
	"encoding/json"
	"io"
	"time"

	"github.com/amirrezaalavi/Viberay/internal/models"
)

// JSONFormatter renders working configs with their original share link and latency.
type JSONFormatter struct{}

type jsonResult struct {
	URI     string `json:"uri"`
	Latency string `json:"latency"`
}

type jsonOutput struct {
	Results []jsonResult `json:"results"`
}

func (f *JSONFormatter) Format(w io.Writer, results []models.TestResult, summary models.Summary) error {
	out := jsonOutput{Results: make([]jsonResult, 0, len(results))}
	for _, r := range results {
		if r.Status != models.StatusSuccess {
			continue
		}
		latency := r.Latencies.Total
		if latency == 0 {
			latency = r.Latencies.Connect
		}
		out.Results = append(out.Results, jsonResult{
			URI:     r.Config.Raw,
			Latency: latency.Round(time.Millisecond).String(),
		})
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
