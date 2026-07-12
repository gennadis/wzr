package runner

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"wzr/internal/notify"
	"wzr/internal/pipeline"
	"wzr/internal/skills"
)

// stubQwen returns pre-configured output per call index.
type stubQwen struct {
	mu      sync.Mutex
	outputs [][]string
	errs    []error
	call    int
}

func (s *stubQwen) Run(_ context.Context, _ string, outputCh chan<- string) error {
	return s.emit(outputCh)
}

func (s *stubQwen) RunText(_ context.Context, _ string, outputCh chan<- string) error {
	return s.emit(outputCh)
}

func (s *stubQwen) emit(outputCh chan<- string) error {
	s.mu.Lock()
	idx := s.call
	s.call++
	s.mu.Unlock()

	if idx < len(s.outputs) {
		for _, l := range s.outputs[idx] {
			outputCh <- l
		}
	}
	if idx < len(s.errs) {
		return s.errs[idx]
	}
	return nil
}

// captureNotifier records all received events in a thread-safe slice and a channel.
type captureNotifier struct {
	mu     sync.Mutex
	events []notify.StepEvent
	ch     chan notify.StepEvent
}

func newCapture() *captureNotifier {
	return &captureNotifier{ch: make(chan notify.StepEvent, 256)}
}

func (c *captureNotifier) Notify(_ context.Context, e notify.StepEvent) error {
	c.mu.Lock()
	c.events = append(c.events, e)
	c.mu.Unlock()
	c.ch <- e
	return nil
}

func (c *captureNotifier) byStatus(s notify.EventStatus) []notify.StepEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []notify.StepEvent
	for _, e := range c.events {
		if e.Status == s {
			out = append(out, e)
		}
	}
	return out
}

// waitForStatus blocks until an event with the given status is captured or 5 s.
func (c *captureNotifier) waitForStatus(s notify.EventStatus) (notify.StepEvent, bool) {
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for {
		select {
		case e := <-c.ch:
			if e.Status == s {
				return e, true
			}
		case <-deadline.C:
			return notify.StepEvent{}, false
		}
	}
}

func newTestRunner(t *testing.T, q QwenClient) (*Runner, *captureNotifier, *ApprovalHub) {
	t.Helper()
	skillFS := fstest.MapFS{
		"my-skill.md": {Data: []byte("# skill\nDo the thing.")},
	}
	cn := newCapture()
	hub := NewApprovalHub()
	roi := NewROITracker(filepath.Join(t.TempDir(), "history.json"))
	store := NewRunStore()
	r := NewRunner(skills.NewRegistry(skillFS), q, cn, hub, roi, store)
	r.ApprovalTimeout = 50 * time.Millisecond
	return r, cn, hub
}

func simplePipeline(steps ...pipeline.Step) *pipeline.Pipeline {
	return &pipeline.Pipeline{Name: "test-pipe", Steps: steps, ManualMinutes: 10}
}

// --- happy path ---

func TestExecute_SuccessPath(t *testing.T) {
	q := &stubQwen{outputs: [][]string{{"narration 1", "narration 2"}}}
	r, cn, _ := newTestRunner(t, q)

	p := simplePipeline(pipeline.Step{ID: "s1", Name: "Step one", Type: pipeline.StepTypeSkill, Skill: "my-skill"})
	run, err := r.Execute(context.Background(), p, nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if run.Status != RunStatusSuccess {
		t.Errorf("run status: got %q, want success", run.Status)
	}
	if len(cn.byStatus(notify.StatusNarration)) != 2 {
		t.Errorf("narration events: got %d, want 2", len(cn.byStatus(notify.StatusNarration)))
	}
	if len(cn.byStatus(notify.StatusSuccess)) == 0 {
		t.Error("no success event fired")
	}
}

func TestExecute_MCPStep(t *testing.T) {
	q := &stubQwen{outputs: [][]string{{"mcp result"}}}
	r, cn, _ := newTestRunner(t, q)

	p := simplePipeline(pipeline.Step{ID: "s1", Name: "MCP", Type: pipeline.StepTypeMCP, Server: "jira", Tool: "search_issues"})
	run, err := r.Execute(context.Background(), p, nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if run.Status != RunStatusSuccess {
		t.Errorf("run status: got %q", run.Status)
	}
	if len(cn.byStatus(notify.StatusNarration)) == 0 {
		t.Error("expected narration events for MCP step")
	}
}

// --- approval gate ---

func TestExecute_ApprovalGate_Approved(t *testing.T) {
	q := &stubQwen{outputs: [][]string{{"before"}, {"after"}}}
	r, cn, hub := newTestRunner(t, q)

	p := simplePipeline(
		pipeline.Step{ID: "s1", Name: "Before", Type: pipeline.StepTypeSkill, Skill: "my-skill"},
		pipeline.Step{ID: "gate", Name: "Approval gate", Type: pipeline.StepTypeApproval},
		pipeline.Step{ID: "s2", Name: "After", Type: pipeline.StepTypeSkill, Skill: "my-skill"},
	)

	runCh := make(chan *Run, 1)
	go func() {
		e, ok := cn.waitForStatus(notify.StatusAwaitingApproval)
		if ok {
			hub.Respond(e.RunID, e.StepID, true)
		}
	}()
	go func() {
		run, _ := r.Execute(context.Background(), p, nil)
		runCh <- run
	}()

	select {
	case run := <-runCh:
		if run.Status != RunStatusSuccess {
			t.Errorf("run status: got %q, want success", run.Status)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("test timed out")
	}
}

func TestExecute_ApprovalGate_Timeout(t *testing.T) {
	q := &stubQwen{} // no outputs — repair and postmortem calls return empty
	r, cn, _ := newTestRunner(t, q)

	p := simplePipeline(pipeline.Step{ID: "gate", Name: "Gate", Type: pipeline.StepTypeApproval})
	run, err := r.Execute(context.Background(), p, nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if run.Status != RunStatusFailed {
		t.Errorf("run status: got %q, want failed", run.Status)
	}
	if len(cn.byStatus(notify.StatusAwaitingApproval)) == 0 {
		t.Error("awaiting_approval event not fired")
	}
}

// --- self-healing repair flow ---

func TestExecute_RepairApproved_RetrySucceeds(t *testing.T) {
	stepErr := errors.New("step failed")
	q := &stubQwen{
		outputs: [][]string{
			{"error output"},
			{`{"diagnosis":"disk full","fix_command":"rm /tmp/junk"}`},
			{"fix executed"},
			{"success output"},
		},
		errs: []error{stepErr, nil, nil, nil},
	}
	r, cn, hub := newTestRunner(t, q)

	p := simplePipeline(pipeline.Step{ID: "s1", Name: "Flaky", Type: pipeline.StepTypeMCP, Server: "jenkins", Tool: "trigger_build"})

	runCh := make(chan *Run, 1)
	go func() {
		e, ok := cn.waitForStatus(notify.StatusRepairSuggested)
		if ok {
			hub.Respond(e.RunID, e.StepID+":repair", true)
		}
	}()
	go func() {
		run, _ := r.Execute(context.Background(), p, nil)
		runCh <- run
	}()

	select {
	case run := <-runCh:
		if run.Status != RunStatusSuccess {
			t.Errorf("run status: got %q, want success", run.Status)
		}
		if len(cn.byStatus(notify.StatusRepairSuggested)) == 0 {
			t.Error("repair_suggested event not fired")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("test timed out")
	}
}

func TestExecute_RepairRejected_PostmortemFired(t *testing.T) {
	stepErr := errors.New("step failed")
	q := &stubQwen{
		outputs: [][]string{
			{"error output"},
			{`{"diagnosis":"unknown","fix_command":""}`},
			{"postmortem analysis"},
		},
		errs: []error{stepErr, nil, nil},
	}
	r, cn, hub := newTestRunner(t, q)

	p := simplePipeline(pipeline.Step{ID: "s1", Name: "Bad step", Type: pipeline.StepTypeMCP, Server: "jenkins", Tool: "trigger_build"})

	runCh := make(chan *Run, 1)
	go func() {
		e, ok := cn.waitForStatus(notify.StatusRepairSuggested)
		if ok {
			hub.Respond(e.RunID, e.StepID+":repair", false)
		}
	}()
	go func() {
		run, _ := r.Execute(context.Background(), p, nil)
		runCh <- run
	}()

	select {
	case run := <-runCh:
		if run.Status != RunStatusFailed {
			t.Errorf("run status: got %q, want failed", run.Status)
		}
		if len(cn.byStatus(notify.StatusPostmortem)) == 0 {
			t.Error("postmortem event not fired")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("test timed out")
	}
}

// --- verify step ---

func TestExecute_VerifyStep_Pass(t *testing.T) {
	q := &stubQwen{outputs: [][]string{
		{"step output"},
		{"PASS: criteria satisfied"},
	}}
	r, cn, _ := newTestRunner(t, q)

	p := simplePipeline(
		pipeline.Step{ID: "s1", Name: "Do work", Type: pipeline.StepTypeSkill, Skill: "my-skill"},
		pipeline.Step{ID: "v1", Name: "Check output", Type: pipeline.StepTypeVerify,
			Params: map[string]string{"criteria": "output must be non-empty"}},
	)
	run, err := r.Execute(context.Background(), p, nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if run.Status != RunStatusSuccess {
		t.Errorf("run status: got %q, want success", run.Status)
	}
	if len(cn.byStatus(notify.StatusSuccess)) == 0 {
		t.Error("no success event fired")
	}
}

func TestExecute_VerifyStep_Fail_TriggersRepair(t *testing.T) {
	q := &stubQwen{
		outputs: [][]string{
			{"step output"},
			{"FAIL: output is missing required field"},
			{`{"diagnosis":"missing field","fix_command":"retry"}`},
			{"postmortem"},
		},
		errs: []error{nil, nil, nil, nil},
	}
	r, cn, hub := newTestRunner(t, q)

	p := simplePipeline(
		pipeline.Step{ID: "s1", Name: "Do work", Type: pipeline.StepTypeSkill, Skill: "my-skill"},
		pipeline.Step{ID: "v1", Name: "Check output", Type: pipeline.StepTypeVerify,
			Params: map[string]string{"criteria": "output must contain required field"}},
	)

	runCh := make(chan *Run, 1)
	go func() {
		e, ok := cn.waitForStatus(notify.StatusRepairSuggested)
		if ok {
			hub.Respond(e.RunID, e.StepID+":repair", false)
		}
	}()
	go func() {
		run, _ := r.Execute(context.Background(), p, nil)
		runCh <- run
	}()

	select {
	case run := <-runCh:
		if run.Status != RunStatusFailed {
			t.Errorf("run status: got %q, want failed", run.Status)
		}
		if len(cn.byStatus(notify.StatusRepairSuggested)) == 0 {
			t.Error("repair_suggested event not fired after verify FAIL")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("test timed out")
	}
}

// --- success criteria ---

func TestExecute_SuccessCriteria_EarlyExit(t *testing.T) {
	q := &stubQwen{outputs: [][]string{
		{"step 1 output"},
		{"YES: goal is achieved"},
	}}
	r, cn, _ := newTestRunner(t, q)

	p := &pipeline.Pipeline{
		Name:            "test-pipe",
		ManualMinutes:   10,
		SuccessCriteria: "output must confirm success",
		Steps: []pipeline.Step{
			{ID: "s1", Name: "First step", Type: pipeline.StepTypeSkill, Skill: "my-skill"},
			{ID: "s2", Name: "Second step", Type: pipeline.StepTypeSkill, Skill: "my-skill"},
		},
	}
	run, err := r.Execute(context.Background(), p, nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if run.Status != RunStatusSuccess {
		t.Errorf("run status: got %q, want success", run.Status)
	}
	// s2 must not have run — only 1 step result
	if len(run.Steps) != 1 {
		t.Errorf("expected 1 completed step (early exit), got %d", len(run.Steps))
	}
	if len(cn.byStatus(notify.StatusEarlySuccess)) == 0 {
		t.Error("early_success event not fired")
	}
}

// --- unit helpers ---

func TestParseRepairResult_ValidJSON(t *testing.T) {
	out := `preamble {"diagnosis":"disk full","fix_command":"df -h"} trailing`
	res := parseRepairResult(out)
	if res.Diagnosis != "disk full" {
		t.Errorf("Diagnosis: got %q", res.Diagnosis)
	}
	if res.FixCommand != "df -h" {
		t.Errorf("FixCommand: got %q", res.FixCommand)
	}
}

func TestParseRepairResult_NoJSON(t *testing.T) {
	out := "plain text"
	res := parseRepairResult(out)
	if res.Diagnosis != out {
		t.Errorf("expected raw output as diagnosis, got %q", res.Diagnosis)
	}
}

func TestROITracker(t *testing.T) {
	roi := NewROITracker(filepath.Join(t.TempDir(), "history.json"))
	if err := roi.Log("r1", "pipe", 30*time.Second, 60, true); err != nil {
		t.Fatalf("Log: %v", err)
	}
	if err := roi.Log("r2", "pipe", 15*time.Second, 60, false); err != nil {
		t.Fatalf("Log: %v", err)
	}
	s, err := roi.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if s.Runs != 2 {
		t.Errorf("Runs: got %d, want 2", s.Runs)
	}
	if s.TotalSavedMinutes != 60 {
		t.Errorf("TotalSavedMinutes: got %d, want 60", s.TotalSavedMinutes)
	}
	if s.SuccessRate != 0.5 {
		t.Errorf("SuccessRate: got %f, want 0.5", s.SuccessRate)
	}
}

func TestRunStore(t *testing.T) {
	s := NewRunStore()
	s.Store("r1", &RunRecord{Status: RunStatusRunning})
	s.AppendLog("r1", "line1")
	s.AppendLog("r1", "line2")
	rec, ok := s.Get("r1")
	if !ok {
		t.Fatal("record not found")
	}
	if rec.Status != RunStatusRunning {
		t.Errorf("Status: got %q", rec.Status)
	}
	if s.GetLog("r1") == "" {
		t.Error("log is empty")
	}
}

func TestApprovalHub_ApproveAndReject(t *testing.T) {
	hub := NewApprovalHub()

	go func() {
		time.Sleep(5 * time.Millisecond)
		hub.Respond("run1", "step1", true)
	}()
	if !hub.WaitForApproval("run1", "step1", time.Second) {
		t.Error("expected approval")
	}

	go func() {
		time.Sleep(5 * time.Millisecond)
		hub.Respond("run1", "step2", false)
	}()
	if hub.WaitForApproval("run1", "step2", time.Second) {
		t.Error("expected rejection")
	}
}

func TestApprovalHub_Timeout(t *testing.T) {
	hub := NewApprovalHub()
	if hub.WaitForApproval("run1", "step1", 20*time.Millisecond) {
		t.Error("expected timeout (false)")
	}
}
