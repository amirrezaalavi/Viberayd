package output

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/amirrezaalavi/Viberay/internal/models"
)

// Export writes categorized config files and error logs to baseDir.
type Export struct {
	BaseDir   string
	Timestamp time.Time
}

// NewExport creates an exporter with a timestamped subdirectory.
func NewExport(baseDir string) *Export {
	return &Export{
		BaseDir:   baseDir,
		Timestamp: time.Now(),
	}
}

// Run writes all categorized outputs.
func (e *Export) Run(results []models.TestResult) error {
	dir := filepath.Join(e.BaseDir, e.Timestamp.Format("20060102_150405"))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}

	var valid, failed, reality, legacy []string
	var errors []string

	for _, r := range results {
		line := fmt.Sprintf("# %s\n%s\n", r.Config.String(), configToURI(r.Config))

		switch r.Status {
		case models.StatusSuccess:
			valid = append(valid, line)
		case models.StatusFailed, models.StatusError:
			failed = append(failed, line)
			errors = append(errors, fmt.Sprintf("%s: %s", r.Config.String(), strings.Join(r.Errors, "; ")))
		}

		if r.Config.Protocol() == models.ProtocolReality {
			reality = append(reality, line)
		}
		if isLegacy(r.Config) {
			legacy = append(legacy, line)
		}
	}

	_ = writeFile(filepath.Join(dir, "valid", "configs.txt"), strings.Join(valid, "\n"))
	_ = writeFile(filepath.Join(dir, "failed", "configs.txt"), strings.Join(failed, "\n"))
	_ = writeFile(filepath.Join(dir, "failed", "errors.log"), strings.Join(errors, "\n"))
	_ = writeFile(filepath.Join(dir, "reality", "configs.txt"), strings.Join(reality, "\n"))
	_ = writeFile(filepath.Join(dir, "legacy", "warnings.txt"), strings.Join(legacy, "\n"))

	return nil
}

func writeFile(path, content string) error {
	if content == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// configToURI returns a best-effort URI representation. Not guaranteed to be
// a perfectly round-trippable URI (some fields may be lost).
func configToURI(cfg models.ProxyConfig) string {
	b := cfg.Base()
	switch cfg.Protocol() {
	case models.ProtocolSS:
		return fmt.Sprintf("ss://%s:%s@%s:%d#%s", cfg.SS.Method, cfg.SS.Password, b.Server, b.Port, b.Name)
	case models.ProtocolVMess:
		return fmt.Sprintf("vmess://%s@%s:%d#%s", cfg.VMess.UUID, b.Server, b.Port, b.Name)
	case models.ProtocolVLess:
		return fmt.Sprintf("vless://%s@%s:%d?security=%s#%s", cfg.VLess.UUID, b.Server, b.Port, tlsSec(cfg.VLess.TLSConfig), b.Name)
	case models.ProtocolTrojan:
		return fmt.Sprintf("trojan://%s@%s:%d#%s", cfg.Trojan.Password, b.Server, b.Port, b.Name)
	case models.ProtocolReality:
		return fmt.Sprintf("vless://%s@%s:%d?security=reality&pbk=%s#%s", cfg.Reality.UUID, b.Server, b.Port, cfg.Reality.PublicKey, b.Name)
	}
	return ""
}

func tlsSec(t models.TLSConfig) string {
	if t.Enabled {
		return "tls"
	}
	return "none"
}

func isLegacy(cfg models.ProxyConfig) bool {
	if cfg.VMess != nil && cfg.VMess.AlterID > 0 {
		return true // VMess with alterID is considered legacy
	}
	return false
}
