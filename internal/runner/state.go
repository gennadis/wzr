package runner

import (
	"sync"
	"time"
)

// RunStatus represents the lifecycle state of a run or step.
type RunStatus string

const (
	RunStatusPending RunStatus = "pending"
	RunStatusRunning RunStatus = "running"
	RunStatusSuccess RunStatus = "success"
	RunStatusFailed  RunStatus = "failed"
)

// StepResult holds the outcome of a single pipeline step.
type StepResult struct {
	StepID string
	Status RunStatus
	Output string
	Error  error
}

// Run is the record of one pipeline execution.
type Run struct {
	ID           string
	PipelineName string
	Steps        []StepResult
	Status       RunStatus
	StartedAt    time.Time
}

// ApprovalHub manages per-step approval channels used by both approval gates
// and repair proposals.
type ApprovalHub struct {
	mu    sync.Mutex
	chans map[string]chan bool
}

// NewApprovalHub creates an empty ApprovalHub.
func NewApprovalHub() *ApprovalHub {
	return &ApprovalHub{chans: make(map[string]chan bool)}
}

// WaitForApproval blocks until an approval is received or timeout elapses.
// Returns true if approved, false if rejected or timed out.
func (h *ApprovalHub) WaitForApproval(runID, stepID string, timeout time.Duration) bool {
	key := runID + ":" + stepID
	ch := make(chan bool, 1)
	h.mu.Lock()
	h.chans[key] = ch
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		delete(h.chans, key)
		h.mu.Unlock()
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case approved := <-ch:
		return approved
	case <-timer.C:
		return false
	}
}

// Respond delivers an approval decision to the waiting goroutine.
func (h *ApprovalHub) Respond(runID, stepID string, approved bool) {
	key := runID + ":" + stepID
	h.mu.Lock()
	ch, ok := h.chans[key]
	h.mu.Unlock()
	if !ok {
		return
	}
	select {
	case ch <- approved:
	default:
	}
}
