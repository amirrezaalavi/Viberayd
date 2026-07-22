package errors

import (
	"errors"
	"fmt"
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

func TestRecommend_InvalidCategory(t *testing.T) {
	// Undefined category should fall through to the default branch
	if got := Recommend(Category("not-a-real-category")); got != ActionContinue {
		t.Errorf("Recommend(garbage) = %q, want continue (default)", got)
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

func TestStrategyFor_NoConfigID(t *testing.T) {
	s := StrategyFor(ErrXrayCrash, "")
	if s.Message == "" {
		t.Error("message should not be empty")
	}
	if s.Category != CategoryRuntime {
		t.Errorf("category = %q, want runtime", s.Category)
	}
}

// --- Categorize ---

func TestCategorize_ParseErrors(t *testing.T) {
	parseErrs := []error{
		ErrInvalidProtocol,
		ErrInvalidEncoding,
		ErrInvalidFormat,
		ErrMissingField,
		ErrInvalidUUID,
		ErrInvalidPort,
		ErrInvalidNetwork,
		ErrInvalidSecurity,
		ErrInvalidFlow,
		ErrInvalidPublicKey,
		ErrInvalidFingerprint,
	}
	for _, e := range parseErrs {
		t.Run(e.Error(), func(t *testing.T) {
			ce := Categorize(e, "test")
			if ce.Category != CategoryParse {
				t.Errorf("Categorize(%v) = %q, want parse", e, ce.Category)
			}
		})
	}
}

func TestCategorize_NetworkErrors(t *testing.T) {
	netErrs := []error{ErrTCPConnect, ErrPortExhausted, ErrPortConflict}
	for _, e := range netErrs {
		t.Run(e.Error(), func(t *testing.T) {
			ce := Categorize(e, "test")
			if ce.Category != CategoryNetwork {
				t.Errorf("Categorize(%v) = %q, want network", e, ce.Category)
			}
		})
	}
}

func TestCategorize_ProtocolErrors(t *testing.T) {
	protoErrs := []error{ErrTLSHandshake, ErrProtocolTest}
	for _, e := range protoErrs {
		t.Run(e.Error(), func(t *testing.T) {
			ce := Categorize(e, "test")
			if ce.Category != CategoryProtocol {
				t.Errorf("Categorize(%v) = %q, want protocol", e, ce.Category)
			}
		})
	}
}

func TestCategorize_ResourceErrors(t *testing.T) {
	ce := Categorize(ErrResourceExhaust, "test")
	if ce.Category != CategoryResource {
		t.Errorf("got %q, want resource", ce.Category)
	}
}

func TestCategorize_RuntimeErrors(t *testing.T) {
	rtErrs := []error{ErrXrayCrash, ErrProxyTest, ErrXrayNotFound}
	for _, e := range rtErrs {
		t.Run(e.Error(), func(t *testing.T) {
			ce := Categorize(e, "test")
			if ce.Category != CategoryRuntime {
				t.Errorf("Categorize(%v) = %q, want runtime", e, ce.Category)
			}
		})
	}
}

func TestCategorize_Unknown(t *testing.T) {
	ce := Categorize(errors.New("completely novel error"), "test")
	if ce.Category != CategoryUnknown {
		t.Errorf("got %q, want unknown", ce.Category)
	}
}

func TestCategorize_WrappedError(t *testing.T) {
	// errors.Is must walk the %w chain for Categorize to find the sentinel
	wrapped := fmt.Errorf("layer 1: %w", fmt.Errorf("layer 2: %w", ErrInvalidPort))
	ce := Categorize(wrapped, "wrapped-test")
	if ce.Category != CategoryParse {
		t.Errorf("Categorize(wrapped) = %q, want parse (port)", ce.Category)
	}
	if ce.ConfigID != "wrapped-test" {
		t.Errorf("config ID = %q, want wrapped-test", ce.ConfigID)
	}
}

func TestCategorizedError_Error(t *testing.T) {
	ce := CategorizedError{
		Category: CategoryNetwork,
		Err:      ErrTCPConnect,
		ConfigID: "",
	}
	got := ce.Error()
	want := "network: TCP connection failed"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestCategorizedError_ErrorWithConfigID(t *testing.T) {
	ce := CategorizedError{
		Category: CategoryProtocol,
		Err:      ErrTLSHandshake,
		ConfigID: "1.2.3.4:443",
	}
	got := ce.Error()
	want := "protocol[1.2.3.4:443]: TLS handshake failed"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestCategorizedError_Unwrap(t *testing.T) {
	ce := CategorizedError{
		Category: CategoryNetwork,
		Err:      ErrTCPConnect,
	}
	if got := errors.Unwrap(ce); got != ErrTCPConnect {
		t.Errorf("Unwrap = %v, want ErrTCPConnect", got)
	}
}
