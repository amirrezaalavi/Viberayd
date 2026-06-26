package output

import (
	"os"
	"testing"

	"github.com/amiralavi/viberay/internal/models"
)

func TestCheckpoint_SaveLoad(t *testing.T) {
	results := []models.TestResult{
		{ID: "t1", Status: models.StatusSuccess, Stage: models.StageCompleted},
	}
	remaining := []models.ProxyConfig{
		{VMess: &models.VMessConfig{BaseConfig: models.BaseConfig{Server: "1.1.1.1", Port: 443}}},
	}
	summary := models.Summary{Total: 1, Passed: 1}

	tmp := t.TempDir()
	path, err := SaveCheckpoint(tmp+"/cp.json", results, summary, remaining)
	if err != nil {
		t.Fatalf("SaveCheckpoint: %v", err)
	}

	cp, err := LoadCheckpoint(path)
	if err != nil {
		t.Fatalf("LoadCheckpoint: %v", err)
	}
	if cp.Version != checkpointVersion {
		t.Errorf("version = %q, want %q", cp.Version, checkpointVersion)
	}
	if len(cp.Results) != 1 {
		t.Errorf("results len = %d, want 1", len(cp.Results))
	}
	if len(cp.Remaining) != 1 {
		t.Errorf("remaining len = %d, want 1", len(cp.Remaining))
	}
	if cp.Summary.Total != 1 {
		t.Errorf("summary total = %d, want 1", cp.Summary.Total)
	}
}

func TestCheckpoint_DefaultPath(t *testing.T) {
	path, err := SaveCheckpoint("", nil, models.Summary{}, nil)
	if err != nil {
		t.Fatalf("SaveCheckpoint: %v", err)
	}
	defer os.Remove(path)

	if _, err := os.Stat(path); err != nil {
		t.Errorf("checkpoint file should exist: %v", err)
	}
}

func TestCheckpoint_LoadMissing(t *testing.T) {
	_, err := LoadCheckpoint("/nonexistent/path/cp.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestRemoveCheckpoint(t *testing.T) {
	tmp := t.TempDir()
	path := tmp + "/cp.json"
	if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := RemoveCheckpoint(path); err != nil {
		t.Fatalf("RemoveCheckpoint: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("checkpoint should be deleted")
	}
}
