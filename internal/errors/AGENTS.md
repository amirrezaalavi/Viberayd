# Errors Package — Agent Guide

This package provides project-wide sentinel errors, error categorization, and recovery recommendations for handling failures during the parsing, testing, and execution phases.

## Purpose

To avoid scattered and un-categorized error strings, VibeRay groups all errors into a structured taxonomy. Every error can be mapped to a core category, which in turn maps to a recommended system action (e.g., whether to skip a config, retry, or reduce concurrency load).

---

## Files

- `errors.go`: Defines sentinel errors, the `Category` type, and the `CategorizedError` struct. It contains the logic to inspect and map standard errors to a classification category.
- `recovery.go`: Defines `RecoveryAction` and `RecoveryStrategy`. Suggests what action the orchestrator or runner should take based on the error classification.
- `recovery_test.go`: Table-driven tests validating taxonomy categorizations and suggested recovery recommendations.

---

## Error Categories & Recommendations

| Category | Sentinel Errors Included | Recovery Action | Description / Orchestrator Action |
|----------|--------------------------|-----------------|-----------------------------------|
| `parse` | `ErrInvalidProtocol`, `ErrInvalidEncoding`, `ErrInvalidFormat`, `ErrMissingField`, `ErrInvalidUUID`, `ErrInvalidPort`, `ErrInvalidNetwork`, `ErrInvalidSecurity`, `ErrInvalidFlow`, `ErrInvalidPublicKey`, `ErrInvalidFingerprint` | `skip` | Instantly drops the configuration without testing. |
| `network` | `ErrTCPConnect`, `ErrPortExhausted`, `ErrPortConflict` | `retry` | Retries with backoff (typically 1s, 2s, 4s). |
| `protocol` | `ErrTLSHandshake`, `ErrProtocolTest` | `continue` | Logs the specific handshake failure and moves to the next config. |
| `resource` | `ErrResourceExhaust` | `reduce-load` | Decreases dynamic worker concurrency to reduce system load. |
| `runtime` | `ErrXrayCrash`, `ErrProxyTest`, `ErrXrayNotFound` | `restart` | Re-initializes/restarts the underlying service engine or proxy binary. |
| `unknown` | All other unclassified errors | `continue` | Logs and continues to prevent a single bad state from stopping the run. |

---

## Key Types & APIs

### `CategorizedError`

A struct wrapping standard errors with a category and an optional configuration identifier.
```go
type CategorizedError struct {
    Category Category
    Err      error
    ConfigID string // Config address/name
}
```

- **`Categorize(err error, configID string) CategorizedError`**: Inspects standard Go errors (and their wrapped chains via `errors.Is`) and maps them to a `CategorizedError`.

### `RecoveryStrategy`

A bundle containing the categorized information and recommended system action.
```go
type RecoveryStrategy struct {
    Category Category       `json:"category"`
    Action   RecoveryAction `json:"action"`
    Message  string         `json:"message"`
}
```

- **`Recommend(cat Category) RecoveryAction`**: Direct mapping function returning the predefined action for a category.
- **`StrategyFor(err error, configID string) RecoveryStrategy`**: Combined convenience function that categorizes an error and provides its strategy.

---

## Conventions & Gotchas

- **Wrapping Sentinels:** Always use `%w` when wrapping errors in the codebase so that `errors.Is(err, Err...)` matching in `Categorize()` remains functional.
- **Panic Recovery:** Avoid using `panic()`. Recoveries are handled gracefully at the task level by recommending actions.
