package errors

import (
	"errors"
	"testing"
)

func TestRecommend(t *testing.T) {
	tests := []struct {
		cat  Category
		want RecoveryAction
	}{
		{CategoryParse, ActionSkip},
		{CategoryNetwork, ActionRetry},
		{CategoryProtocol, ActionContinue},
		{CategoryResource, ActionReduceLoad},
		{CategoryRuntime, ActionRestart},
		{CategoryUnknown, ActionContinue},
	}
	for _, tt := range tests {
		t.Run(string(tt.cat), func(t *testing.T) {
			if got := Recommend(tt.cat); got != tt.want {
				t.Errorf("Recommend(%q) = %q, want %q", tt.cat, got, tt.want)
			}
		})
	}
}

func TestStrategyFor(t *testing.T) {
	s := StrategyFor(ErrTCPConnect, "cfg-1")
	if s.Category != CategoryNetwork {
		t.Errorf("category = %q, want network", s.Category)
	}
	if s.Action != ActionRetry {
		t.Errorf("action = %q, want retry", s.Action)
	}
	if s.Message == "" {
		t.Error("message should not be empty")
	}
}

func TestStrategyFor_Unknown(t *testing.T) {
	s := StrategyFor(errors.New("random error"), "cfg-2")
	if s.Category != CategoryUnknown {
		t.Errorf("category = %q, want unknown", s.Category)
	}
	if s.Action != ActionContinue {
		t.Errorf("action = %q, want continue", s.Action)
	}
}
