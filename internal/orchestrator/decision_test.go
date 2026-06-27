package orchestrator

import (
	"testing"
	"time"

	"github.com/amiralavi/viberay/internal/models"
)

func TestDecide_Depth(t *testing.T) {
	tests := []struct {
		name      string
		total     int
		reality   int
		override  models.TestDepth
		wantDepth models.TestDepth
	}{
		{"empty", 0, 0, "", models.DepthQuick},
		{"small no reality", 5, 0, "", models.DepthComprehensive},
		{"small with reality", 5, 1, "", models.DepthComprehensive},
		{"medium no reality", 50, 0, "", models.DepthFull},
		{"medium with reality", 50, 2, "", models.DepthComprehensive},
		{"large no reality", 200, 0, "", models.DepthStandard},
		{"large with reality", 200, 1, "", models.DepthFull},
		{"huge", 1000, 0, "", models.DepthQuick},
		{"override", 1000, 0, models.DepthFull, models.DepthFull},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := models.TestContext{
				Input: models.InputStats{
					Total:               tt.total,
					ProtocolDistribution: map[string]int{string(models.ProtocolReality): tt.reality},
				},
			}
			prefs := UserPreferences{Depth: tt.override}
			d, err := Decide(ctx, prefs)
			if err != nil {
				t.Fatalf("Decide error: %v", err)
			}
			if d.Depth != tt.wantDepth {
				t.Errorf("depth = %q, want %q", d.Depth, tt.wantDepth)
			}
		})
	}
}

func TestDecide_Style(t *testing.T) {
	tests := []struct {
		name     string
		total    int
		override models.OutputStyle
		want     models.OutputStyle
	}{
		{"tiny", 3, models.StyleAuto, models.StyleTable},
		{"small", 50, models.StyleAuto, models.StyleTable},
		{"large", 200, models.StyleAuto, models.StyleCSV},
		{"override", 200, models.StyleMarkdown, models.StyleMarkdown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := models.TestContext{Input: models.InputStats{Total: tt.total}}
			prefs := UserPreferences{Style: tt.override}
			d, _ := Decide(ctx, prefs)
			if d.Style != tt.want {
				t.Errorf("style = %q, want %q", d.Style, tt.want)
			}
		})
	}
}

func TestDecide_Concurrency(t *testing.T) {
	ctx := models.TestContext{System: models.SystemInfo{CPUCount: 4}}
	prefs := UserPreferences{Workers: 0}
	d, _ := Decide(ctx, prefs)
	if d.Concurrency.Workers != 8 {
		t.Errorf("workers = %d, want 8", d.Concurrency.Workers)
	}

	prefs.Workers = 16
	d, _ = Decide(ctx, prefs)
	if d.Concurrency.Workers != 16 {
		t.Errorf("override workers = %d, want 16", d.Concurrency.Workers)
	}
}

func TestDecide_Cache(t *testing.T) {
	// Explicit false
	f := false
	ctx := models.TestContext{Input: models.InputStats{Total: 100, Duplicates: 10}}
	prefs := UserPreferences{Cache: &f}
	d, _ := Decide(ctx, prefs)
	if d.CacheEnabled {
		t.Error("expected cache disabled by override")
	}

	// Auto: duplicates 10% > 5% threshold
	prefs.Cache = nil
	d, _ = Decide(ctx, prefs)
	if !d.CacheEnabled {
		t.Error("expected cache enabled (10% duplicates)")
	}

	// Auto: duplicates 1% < 5% threshold
	ctx.Input.Duplicates = 1
	d, _ = Decide(ctx, prefs)
	if d.CacheEnabled {
		t.Error("expected cache disabled (1% duplicates)")
	}
}

func TestDecide_Timeout(t *testing.T) {
	ctx := models.TestContext{Input: models.InputStats{
		Total: 10,
		ProtocolDistribution: map[string]int{string(models.ProtocolReality): 1},
	}}
	prefs := UserPreferences{Timeout: 0}
	d, _ := Decide(ctx, prefs)
	if d.TimeoutPerTest != 7*time.Second {
		t.Errorf("timeout = %v, want 7s", d.TimeoutPerTest)
	}

	prefs.Timeout = 3 * time.Second
	d, _ = Decide(ctx, prefs)
	if d.TimeoutPerTest != 3*time.Second {
		t.Errorf("override timeout = %v, want 3s", d.TimeoutPerTest)
	}
}

func TestDecide_Retry(t *testing.T) {
	ctx := models.TestContext{Input: models.InputStats{Total: 50}}
	prefs := UserPreferences{MaxRetries: -1}
	d, _ := Decide(ctx, prefs)
	if d.Retry.MaxRetries != 2 {
		t.Errorf("retries = %d, want 2 (small batch)", d.Retry.MaxRetries)
	}

	ctx.Input.Total = 200
	d, _ = Decide(ctx, prefs)
	if d.Retry.MaxRetries != 1 {
		t.Errorf("retries = %d, want 1 (large batch)", d.Retry.MaxRetries)
	}

	prefs.MaxRetries = 5
	d, _ = Decide(ctx, prefs)
	if d.Retry.MaxRetries != 5 {
		t.Errorf("override retries = %d, want 5", d.Retry.MaxRetries)
	}
}

func TestBuildContext(t *testing.T) {
	configs := []models.ProxyConfig{
		{SS: &models.SSConfig{BaseConfig: models.BaseConfig{Server: "1.1.1.1", Port: 443}}},
		{VMess: &models.VMessConfig{BaseConfig: models.BaseConfig{Server: "2.2.2.2", Port: 443}}},
		{SS: &models.SSConfig{BaseConfig: models.BaseConfig{Server: "1.1.1.1", Port: 443}}}, // duplicate
	}

	ctx := BuildContext(configs, 1)
	if ctx.Input.Total != 3 {
		t.Errorf("total = %d, want 3", ctx.Input.Total)
	}
	if ctx.Input.Duplicates != 1 {
		t.Errorf("duplicates = %d, want 1", ctx.Input.Duplicates)
	}
	if ctx.Input.ParseErrors != 1 {
		t.Errorf("parseErrors = %d, want 1", ctx.Input.ParseErrors)
	}
	if ctx.System.CPUCount <= 0 {
		t.Error("expected CPUCount > 0")
	}
	if ctx.Input.ProtocolDistribution[string(models.ProtocolSS)] != 2 {
		t.Errorf("SS count = %d, want 2", ctx.Input.ProtocolDistribution[string(models.ProtocolSS)])
	}
}

func TestBumpDepth(t *testing.T) {
	tests := []struct {
		in, want models.TestDepth
	}{
		{models.DepthQuick, models.DepthStandard},
		{models.DepthStandard, models.DepthFull},
		{models.DepthFull, models.DepthComprehensive},
		{models.DepthComprehensive, models.DepthComprehensive},
	}
	for _, tt := range tests {
		t.Run(string(tt.in), func(t *testing.T) {
			if got := bumpDepth(tt.in); got != tt.want {
				t.Errorf("bumpDepth(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
