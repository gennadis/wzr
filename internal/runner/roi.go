package runner

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

type roiEntry struct {
	RunID         string  `json:"run_id"`
	Pipeline      string  `json:"pipeline"`
	DurationSecs  float64 `json:"duration_secs"`
	ManualMinutes int     `json:"manual_minutes"`
	Success       bool    `json:"success"`
}

// Stats summarizes aggregated ROI data from run history.
type Stats struct {
	Runs              int     `json:"runs"`
	TotalSavedMinutes int     `json:"total_saved_minutes"`
	SuccessRate       float64 `json:"success_rate"`
}

// ROITracker persists run records to a JSON file and computes aggregate stats.
type ROITracker struct {
	mu       sync.Mutex
	filePath string
}

// NewROITracker creates a ROITracker writing to filePath.
func NewROITracker(filePath string) *ROITracker {
	return &ROITracker{filePath: filePath}
}

// Log appends a run record to the history file.
func (t *ROITracker) Log(runID, pipelineName string, duration time.Duration, manualMinutes int, success bool) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	entries, err := t.load()
	if err != nil {
		return err
	}
	entries = append(entries, roiEntry{
		RunID:         runID,
		Pipeline:      pipelineName,
		DurationSecs:  duration.Seconds(),
		ManualMinutes: manualMinutes,
		Success:       success,
	})
	return t.save(entries)
}

// Stats reads the history file and returns aggregate metrics.
func (t *ROITracker) Stats() (Stats, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	entries, err := t.load()
	if err != nil {
		return Stats{}, err
	}
	var s Stats
	s.Runs = len(entries)
	successes := 0
	for _, e := range entries {
		if e.Success {
			successes++
			s.TotalSavedMinutes += e.ManualMinutes
		}
	}
	if s.Runs > 0 {
		s.SuccessRate = float64(successes) / float64(s.Runs)
	}
	return s, nil
}

func (t *ROITracker) load() ([]roiEntry, error) {
	data, err := os.ReadFile(t.filePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read history: %w", err)
	}
	var entries []roiEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parse history: %w", err)
	}
	return entries, nil
}

func (t *ROITracker) save(entries []roiEntry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal history: %w", err)
	}
	if err := os.WriteFile(t.filePath, data, 0o600); err != nil {
		return fmt.Errorf("write history: %w", err)
	}
	return nil
}
