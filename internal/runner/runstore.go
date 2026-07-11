package runner

import "sync"

// RunRecord holds the accumulated log and final status of a run.
type RunRecord struct {
	Log    string
	Status RunStatus
}

// RunStore is a thread-safe in-memory store for active and recent runs.
type RunStore struct {
	mu      sync.RWMutex
	records map[string]*RunRecord
}

// NewRunStore creates an empty RunStore.
func NewRunStore() *RunStore {
	return &RunStore{records: make(map[string]*RunRecord)}
}

// Store inserts or replaces the record for the given run ID.
func (s *RunStore) Store(id string, record *RunRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[id] = record
}

// Get retrieves the record for the given run ID.
func (s *RunStore) Get(id string) (*RunRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.records[id]
	return r, ok
}

// AppendLog appends a line to the accumulated log for the given run.
func (s *RunStore) AppendLog(id, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, ok := s.records[id]; ok {
		r.Log += text + "\n"
	}
}

// GetLog returns the accumulated log for the given run.
func (s *RunStore) GetLog(id string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if r, ok := s.records[id]; ok {
		return r.Log
	}
	return ""
}
