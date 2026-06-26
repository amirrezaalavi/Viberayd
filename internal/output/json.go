package output

import (
	"encoding/json"
	"io"

	"github.com/amiralavi/viberay/internal/models"
)

// JSONFormatter renders full machine-readable output.
type JSONFormatter struct{}

// jsonOutput is the top-level JSON structure.
type jsonOutput struct {
	Summary models.Summary     `json:"summary"`
	Results []models.TestResult `json:"results"`
}

func (f *JSONFormatter) Format(w io.Writer, results []models.TestResult, summary models.Summary) error {
	out := jsonOutput{
		Summary: summary,
		Results: results,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
