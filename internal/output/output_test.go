package output

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/amiralavi/viberay/internal/models"
)

func sampleResults() ([]models.TestResult, models.Summary) {
	results := []models.TestResult{
		{
			ID:     "vmess-1",
			Config: models.ProxyConfig{VMess: &models.VMessConfig{BaseConfig: models.BaseConfig{Server: "1.1.1.1", Port: 443, Protocol: models.ProtocolVMess, Name: "CF-1"}, UUID: "a0b1c2d3-e4f5-6789-abcd-ef0123456789"}},
			Status:    models.StatusSuccess,
			Stage:     models.StageCompleted,
			Latencies: models.LatencyBreakdown{Total: 45 * time.Millisecond},
		},
		{
			ID:     "ss-1",
			Config: models.ProxyConfig{SS: &models.SSConfig{BaseConfig: models.BaseConfig{Server: "2.2.2.2", Port: 8388, Protocol: models.ProtocolSS, Name: "SS-JP"}, Method: "aes-256-gcm", Password: "pass"}},
			Status:    models.StatusFailed,
			Stage:     models.StageTCP,
			Latencies: models.LatencyBreakdown{Connect: 3 * time.Second},
			Errors:    []string{"connection refused"},
		},
		{
			ID:     "reality-1",
			Config: models.ProxyConfig{Reality: &models.RealityConfig{BaseConfig: models.BaseConfig{Server: "3.3.3.3", Port: 443, Protocol: models.ProtocolReality, Name: "R-US"}, UUID: "b1c2d3e4-f5a6-7890-bcde-f01234567890", PublicKey: "abc123"}},
			Status:    models.StatusSuccess,
			Stage:     models.StageCompleted,
			Latencies: models.LatencyBreakdown{Total: 120 * time.Millisecond},
		},
	}

	summary := models.Summary{
		Total:            3,
		Passed:           2,
		Failed:           1,
		ByProtocol:       map[string]int{"vmess": 1, "ss": 1, "reality": 1},
		AvgLatencyMs:     55.0,
		SuccessRatePct:   66.7,
		ConfigsPerSecond: 1.5,
		Duration:         2 * time.Second,
	}
	return results, summary
}

func TestJSONFormatter(t *testing.T) {
	f := &JSONFormatter{}
	results, summary := sampleResults()

	var buf bytes.Buffer
	if err := f.Format(&buf, results, summary); err != nil {
		t.Fatalf("Format: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if _, ok := out["summary"]; !ok {
		t.Error("missing summary key")
	}
	if _, ok := out["results"]; !ok {
		t.Error("missing results key")
	}
}

func TestCSVFormatter(t *testing.T) {
	f := &CSVFormatter{}
	results, summary := sampleResults()

	var buf bytes.Buffer
	if err := f.Format(&buf, results, summary); err != nil {
		t.Fatalf("Format: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 4 {
		t.Fatalf("expected at least 4 CSV lines, got %d", len(lines))
	}

	if !strings.HasPrefix(lines[0], "protocol,") {
		t.Errorf("unexpected header: %s", lines[0])
	}
	if !strings.Contains(lines[len(lines)-1], "SUMMARY") {
		t.Errorf("expected summary row, got: %s", lines[len(lines)-1])
	}
}

func TestTableFormatter(t *testing.T) {
	f := &TableFormatter{}
	results, summary := sampleResults()

	var buf bytes.Buffer
	if err := f.Format(&buf, results, summary); err != nil {
		t.Fatalf("Format: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "PROTOCOL") {
		t.Error("missing PROTOCOL header")
	}
	if !strings.Contains(out, "PASS") {
		t.Error("missing PASS status")
	}
	if !strings.Contains(out, "FAIL") {
		t.Error("missing FAIL status")
	}
	if !strings.Contains(out, "Summary:") {
		t.Error("missing summary line")
	}
}

func TestMarkdownFormatter(t *testing.T) {
	f := &MarkdownFormatter{}
	results, summary := sampleResults()

	var buf bytes.Buffer
	if err := f.Format(&buf, results, summary); err != nil {
		t.Fatalf("Format: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "# VibeRay Test Report") {
		t.Error("missing title")
	}
	if !strings.Contains(out, "## Summary") {
		t.Error("missing summary section")
	}
	if !strings.Contains(out, "## Detailed Results") {
		t.Error("missing detailed results section")
	}
	if !strings.Contains(out, "| vmess |") {
		t.Error("missing vmess row")
	}
}

func TestNew(t *testing.T) {
	for _, style := range []models.OutputStyle{models.StyleJSON, models.StyleCSV, models.StyleTable, models.StyleMarkdown} {
		f, err := New(style)
		if err != nil {
			t.Errorf("New(%q) error: %v", style, err)
		}
		if f == nil {
			t.Errorf("New(%q) returned nil formatter", style)
		}
	}

	if _, err := New(models.StyleAuto); err == nil {
		t.Error("expected error for auto style")
	}
	if _, err := New(models.OutputStyle("bogus")); err == nil {
		t.Error("expected error for bogus style")
	}
}

func TestExport(t *testing.T) {
	results, _ := sampleResults()
	tmp := t.TempDir()

	ex := NewExport(tmp)
	if err := ex.Run(results); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// valid/configs.txt should contain the 2 passing configs
	data, err := os.ReadFile(filepath.Join(tmp, ex.Timestamp.Format("20060102_150405"), "valid", "configs.txt"))
	if err != nil {
		t.Fatalf("read valid configs: %v", err)
	}
	if !strings.Contains(string(data), "vmess") {
		t.Error("valid configs should contain vmess")
	}

	// failed/errors.log should contain the error
	data, err = os.ReadFile(filepath.Join(tmp, ex.Timestamp.Format("20060102_150405"), "failed", "errors.log"))
	if err != nil {
		t.Fatalf("read errors: %v", err)
	}
	if !strings.Contains(string(data), "connection refused") {
		t.Error("errors log should contain connection refused")
	}
}

func TestConfigToURI(t *testing.T) {
	tests := []struct {
		name string
		cfg  models.ProxyConfig
		want string
	}{
		{
			name: "ss",
			cfg:  models.ProxyConfig{SS: &models.SSConfig{BaseConfig: models.BaseConfig{Server: "s", Port: 80, Name: "n"}, Method: "m", Password: "p"}},
			want: "ss://m:p@s:80#n",
		},
		{
			name: "trojan",
			cfg:  models.ProxyConfig{Trojan: &models.TrojanConfig{BaseConfig: models.BaseConfig{Server: "s", Port: 443, Name: "n"}, Password: "p"}},
			want: "trojan://p@s:443#n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := configToURI(tt.cfg)
			if got != tt.want {
				t.Errorf("configToURI() = %q, want %q", got, tt.want)
			}
		})
	}
}
