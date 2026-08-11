package daemon

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

const (
	StateUnknown     = "unknown"
	StateUnreachable = "unreachable"
	StateFailed      = "failed"
	StateWorking     = "working"
)

type State struct {
	Version   int                    `json:"version"`
	UpdatedAt time.Time              `json:"updated_at"`
	Configs   map[string]*ConfigEntry `json:"configs"` // key: sha256(raw)
}

type ConfigEntry struct {
	Raw          string    `json:"raw"`
	Host         string    `json:"host"`
	Port         int       `json:"port"`
	Protocol     string    `json:"protocol"`
	SourceURL    string    `json:"source_url"`
	FirstSeen    time.Time `json:"first_seen"`
	LastTested   time.Time `json:"last_tested"`
	LastSuccess  time.Time `json:"last_success"`
	SuccessCount int       `json:"success_count"`
	FailCount    int       `json:"fail_count"`
	State        string    `json:"state"`
	LatencyMs    int       `json:"latency_ms"`
}

func NewState() *State {
	return &State{
		Version:   1,
		UpdatedAt: time.Now(),
		Configs:   make(map[string]*ConfigEntry),
	}
}

func LoadState(path string) (*State, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return NewState(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read state: %w", err)
	}

	s := NewState()
	if err := json.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("decode state: %w", err)
	}

	if s.Configs == nil {
		s.Configs = make(map[string]*ConfigEntry)
	}

	s.UpdatedAt = time.Now()

	return s, nil
}

func SaveState(path string, s *State) error {
	s.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}

	tmp := path + ".tmp." + randHex(8)
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("write state tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename state: %w", err)
	}

	return nil
}

func SelectCandidates(s *State, retestInterval time.Duration, keepSuccessful bool, now time.Time) []string {
	var candidates []string
	for hash, entry := range s.Configs {
		switch entry.State {
		case StateUnknown, StateFailed, StateUnreachable:
			candidates = append(candidates, hash)
		case StateWorking:
			if keepSuccessful && now.Sub(entry.LastTested) >= retestInterval {
				candidates = append(candidates, hash)
			}
		}
	}
	return candidates
}

type TestResult struct {
	Hash      string
	Success   bool
	LatencyMs int
}

// ApplyResults updates config states from a round of test results. When
// maxLatencyMs is provided and > 0, a successful test whose latency exceeds
// the threshold is treated as a failure (config excluded from working output).
// The measured latency is still recorded for visibility.
func ApplyResults(s *State, results []TestResult, now time.Time, maxLatencyMs ...int) {
	threshold := 0
	if len(maxLatencyMs) > 0 {
		threshold = maxLatencyMs[0]
	}

	for _, r := range results {
		entry, ok := s.Configs[r.Hash]
		if !ok {
			continue
		}

		entry.LastTested = now

		success := r.Success
		if success && threshold > 0 && r.LatencyMs > threshold {
			// Passed the test but latency over the operator's ceiling:
			// treat as not working so it drops out of the output.
			success = false
		}

		if success {
			entry.State = StateWorking
			entry.LastSuccess = now
			entry.SuccessCount++
			entry.LatencyMs = r.LatencyMs
		} else {
			entry.FailCount++
			if entry.State == StateWorking {
				entry.State = StateFailed
			} else if entry.State != StateUnreachable {
				entry.State = StateFailed
			}
			if r.LatencyMs > 0 {
				// Keep the measured latency even on failure (e.g. a
				// threshold rejection) for visibility in state.json.
				entry.LatencyMs = r.LatencyMs
			}
		}
	}
}

func randHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
