# Logging Package — Agent Guide

This package provides unified structured logging configurations using Go's standard library `log/slog` package.

## Purpose

To initialize and customize log outputs across the CLI tool. By routing all system messages (info, debug, warning, error) to `os.Stderr`, we leave `os.Stdout` clean and dedicated entirely to the final validated proxy configurations.

---

## Files

- `logger.go`: Contains logger initialization (`Init()`) and level-parsing utilities.

---

## Public API

### `Init(level string) *slog.Logger`

Returns a standard structured text handler configured to write to `os.Stderr`. 

- **Level Parsing:** Accepts case-insensitive values (`"debug"`, `"info"`, `"warn"`, `"error"`). If invalid, defaults to `"info"`.
- **Source Context:** When configured with `"debug"` level, it automatically includes file and line source locations (`AddSource: true`) inside the structured output.

---

## Conventions & Gotchas

- **Do Not Use Standard `log`:** Always use `log/slog` to record system activity. Avoid standard `log.Printf` or `fmt.Println` for logging, as they lack key-value structure and stream management.
- **Write Only to Stderr:** The logger must write exclusively to `os.Stderr` to prevent debug outputs from polluting the CLI's standard output pipeline (which outputs the working proxy URIs).
