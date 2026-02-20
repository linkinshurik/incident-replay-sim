package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const runsDir = "./data/runs"

// Report represents the data to be stored for a run
// We keep it generic as map[string]interface{} to allow flexible report
// For simplicity, we store it as JSON to file runId.json

type Report map[string]interface{}

// RunStore provides methods to save and list run reports

type RunStore struct {
	mu sync.Mutex
}

func NewRunStore() *RunStore {
	return &RunStore{}
}

// Save persists the report to ./data/runs/<runId>.json atomically
func (s *RunStore) Save(runID string, report Report) error {
	if runID == "" {
		return errors.New("runID cannot be empty")
	}

	// Create directory if not exists
	if err := os.MkdirAll(runsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create data dir: %w", err)
	}

	// Marshal report to JSON
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report json: %w", err)
	}

	// Write temp file first
	tempFile := filepath.Join(runsDir, runID+".json.tmp")
	finalFile := filepath.Join(runsDir, runID+".json")

	if err := os.WriteFile(tempFile, data, 0o644); err != nil {
		return fmt.Errorf("failed to write temp report file: %w", err)
	}

	// Rename temp to final atomically
	if err := os.Rename(tempFile, finalFile); err != nil {
		return fmt.Errorf("failed to rename report file: %w", err)
	}

	return nil
}

// List returns at most 'limit' runIDs ordered by filename ascending
func (s *RunStore) List(limit int) ([]string, error) {
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read runs dir: %w", err)
	}

	var runIDs []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if filepath.Ext(name) != ".json" {
			continue
		}
		runID := name[:len(name)-len(".json")]
		runIDs = append(runIDs, runID)
	}

	// Limit the results
	if limit > 0 && len(runIDs) > limit {
		runIDs = runIDs[:limit]
	}

	return runIDs, nil
}

// Extra: Load returns loaded report for a given runID (not in spec but can be useful)
func (s *RunStore) Load(runID string) (Report, error) {
	if runID == "" {
		return nil, errors.New("runID cannot be empty")
	}

	file := filepath.Join(runsDir, runID+".json")
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read report file: %w", err)
	}

	var report Report
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to unmarshal report json: %w", err)
	}

	return report, nil
}
