package notify

import "context"

// EventStatus describes the state of a step or run event.
type EventStatus string

const (
	StatusRunning          EventStatus = "running"
	StatusSuccess          EventStatus = "success"
	StatusFailed           EventStatus = "failed"
	StatusAwaitingApproval EventStatus = "awaiting_approval"
	StatusRepairSuggested  EventStatus = "repair_suggested"
	StatusNarration        EventStatus = "narration"
	StatusPostmortem       EventStatus = "postmortem"
)

// StepEvent is the unified event shape shared by SSE and SberChat notifiers.
type StepEvent struct {
	RunID      string      `json:"run_id"`
	Pipeline   string      `json:"pipeline"`
	StepID     string      `json:"step_id"`
	StepName   string      `json:"step_name"`
	Status     EventStatus `json:"status"`
	Output     string      `json:"output,omitempty"`
	Error      string      `json:"error,omitempty"`
	Diagnosis  string      `json:"diagnosis,omitempty"`
	FixCommand string      `json:"fix_command,omitempty"`
	TS         int64       `json:"ts"`
}

// Notifier receives step events from the runner.
type Notifier interface {
	Notify(ctx context.Context, event StepEvent) error
}
