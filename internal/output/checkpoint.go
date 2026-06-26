package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/amiralavi/viberay/internal/models"
)

// Checkpoint saves partial results to disk so a long run can be resumed
// after an interrupt or crash.
type Checkpoint struct {
	Version   string             `json:"version"`
	SavedAt   time.Time          `json:"saved_at"`
	Results   []models.TestResult `json:"results"`
	Summary   models.Summary     `json:"summary"`
	Remaining []models.ProxyConfig `json:"remaining"`
}

const checkpointVersion = "1.0"

// SaveCheckpoint writes a checkpoint to the given path (or a temp file if empty).
func SaveCheckpoint(path string, results []models.TestResult, summary models.Summary, remaining []models.ProxyConfig) (string, error) {
	cp := Checkpoint{
		Version:   checkpointVersion,
		SavedAt:   time.Now(),
		Results:   results,
		Summary:   summary,
		Remaining: remaining,
	}

	b, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal checkpoint: %w", err)
	}

	if path == "" {
		path = filepath.Join(os.TempDir(), fmt.Sprintf("viberay-checkpoint-%d.json", time.Now().Unix()))
	}

	if err := os.WriteFile(path, b, 0644); err != nil {
		return "", fmt.Errorf("write checkpoint: %w", err)
	}
	return path, nil
}

// LoadCheckpoint reads a checkpoint from disk.
func LoadCheckpoint(path string) (*Checkpoint, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read checkpoint: %w", err)
	}
	var cp Checkpoint
	if err := json.Unmarshal(b, &cp); err != nil {
		return nil, fmt.Errorf("parse checkpoint: %w", err)
	}
	return &cp, nil
}

// RemoveCheckpoint deletes the checkpoint file if it exists.
func RemoveCheckpoint(path string) error {
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
