// Package output generates reports and exports for test results.
package output

import (
	"fmt"
	"io"

	"github.com/amiralavi/viberay/internal/models"
)

// Formatter renders a slice of TestResults into a specific output style.
type Formatter interface {
	// Format writes the formatted output to w.
	Format(w io.Writer, results []models.TestResult, summary models.Summary) error
}

// New returns a Formatter for the given style.
func New(style models.OutputStyle) (Formatter, error) {
	switch style {
	case models.StyleJSON:
		return &JSONFormatter{}, nil
	case models.StyleCSV:
		return &CSVFormatter{}, nil
	case models.StyleTable:
		return &TableFormatter{}, nil
	case models.StyleMarkdown:
		return &MarkdownFormatter{}, nil
	case models.StyleHTML:
		return nil, fmt.Errorf("HTML output not yet implemented")
	case models.StyleAuto:
		return nil, fmt.Errorf("auto style must be resolved before creating a formatter")
	default:
		return nil, fmt.Errorf("unsupported output style: %q", style)
	}
}
