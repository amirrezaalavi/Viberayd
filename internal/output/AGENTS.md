# Output Package — Agent Guide

This package is responsible for formatting, exporting, and checkpointing test results. It outputs only verified, working configurations.

## Purpose

Once the pipeline completes testing configurations, they are formatted into user-friendly reports or exported into categorized disk files. If the process is canceled by a user (via SIGINT / Ctrl+C), this package writes checkpoints to disk so that execution can be resumed later without losing progress.

---

## Files

- `formatter.go`: Defines the central `Formatter` interface and factory (`New()`).
- `table.go`: Implements `TableFormatter`, outputting configs as raw URIs followed by latency (space-separated, printed per line). This is the CLI's default format.
- `json.go`: Implements `JSONFormatter`, printing successful entries as JSON structures.
- `csv.go`: Implements `CSVFormatter`, printing entries as comma-separated `raw-uri,latency` columns.
- `markdown.go`: Implements `MarkdownFormatter`, formatting successful configs into a Markdown table with latency headers.
- `exporter.go`: Writes structured files under timestamped sub-directories (valid configs, failed configs, error logs, reality, legacy/deprecated configs).
- `checkpoint.go`: Handles saving/loading checkpoint JSON payloads containing current progress and outstanding configs.
- `checkpoint_test.go` and `output_test.go`: Unit tests ensuring accurate output representation, schema verification, and resume/interrupt simulations.

---

## Key Interfaces & Implementations

### `Formatter` Interface

```go
type Formatter interface {
    Format(w io.Writer, results []models.TestResult, summary models.Summary) error
}
```

The formatters strictly output **only working configurations** using the exact, preserved `ProxyConfig.Raw` URI strings.

### Export Categorization (`exporter.go`)

The `Export.Run(results)` method creates a timestamped subdirectory (e.g. `20260717_120000`) and exports:
- `valid/configs.txt`: List of working configs (with names and raw share links).
- `failed/configs.txt`: List of failed configs.
- `failed/errors.log`: Detailed, semicolon-separated list of errors matched to config names.
- `reality/configs.txt`: Reality configs exclusively.
- `legacy/warnings.txt`: Warns on deprecated nodes (e.g., VMess configs with an `AlterID > 0`).

### Checkpointing Engine (`checkpoint.go`)

Allows a user to interrupt heavy tests via Ctrl+C, storing intermediate state:
```go
type Checkpoint struct {
    Version   string             `json:"version"`
    SavedAt   time.Time          `json:"saved_at"`
    Results   []models.TestResult `json:"results"`
    Summary   models.Summary     `json:"summary"`
    Remaining []models.ProxyConfig `json:"remaining"`
}
```
- **`SaveCheckpoint(path, results, summary, remaining)`**: Serializes the state. If no path is specified, saves to system temp space.
- **`LoadCheckpoint(path)`**: Deserializes the checkpoint file.

---

## Conventions & Gotchas

- **Do Not Reconstruct URIs:** Always print the original share URI using `ProxyConfig.Raw` inside formatters to prevent information loss (e.g., custom transport parameters, sni fields, obfs strings).
- **Checkpoints on Exit:** Checkpoints are triggered during signal catches in `cmd/viberay/main.go` only if there are outstanding unprocessed configs.
- **HTML Stub:** HTML reports are planned but not yet implemented. Creating a formatter for HTML style returns an explicit error.
