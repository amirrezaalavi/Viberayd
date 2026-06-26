package errors

import "fmt"

// RecoveryAction describes what the system should do after a categorized failure.
type RecoveryAction string

const (
	ActionSkip       RecoveryAction = "skip"       // drop the item (parse errors)
	ActionRetry      RecoveryAction = "retry"      // retry with backoff (network errors)
	ActionContinue   RecoveryAction = "continue"   // log and keep going (protocol errors)
	ActionReduceLoad RecoveryAction = "reduce-load" // decrease parallelism (resource errors)
	ActionRestart    RecoveryAction = "restart"    // restart xray process (runtime errors)
	ActionAbort      RecoveryAction = "abort"      // stop everything (unrecoverable)
)

// Recommend returns the suggested recovery action for a given category.
func Recommend(cat Category) RecoveryAction {
	switch cat {
	case CategoryParse:
		return ActionSkip
	case CategoryNetwork:
		return ActionRetry
	case CategoryProtocol:
		return ActionContinue
	case CategoryResource:
		return ActionReduceLoad
	case CategoryRuntime:
		return ActionRestart
	default:
		return ActionContinue
	}
}

// RecoveryStrategy bundles the category, action, and a human-readable suggestion.
type RecoveryStrategy struct {
	Category Category       `json:"category"`
	Action   RecoveryAction `json:"action"`
	Message  string         `json:"message"`
}

// StrategyFor returns a full strategy for an error.
func StrategyFor(err error, configID string) RecoveryStrategy {
	ce := Categorize(err, configID)
	action := Recommend(ce.Category)
	msg := fmt.Sprintf("%s: %s (action: %s)", ce.Category, ce.Err.Error(), action)
	return RecoveryStrategy{
		Category: ce.Category,
		Action:   action,
		Message:  msg,
	}
}
