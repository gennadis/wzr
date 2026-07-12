package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"wzr/internal/notify"
	"wzr/internal/pipeline"
	"wzr/internal/executor"
	"wzr/internal/skills"
)

const defaultApprovalTimeout = 30 * time.Minute

// QwenClient is the interface for running Qwen prompts — satisfied by *executor.CLIExecutor.
type QwenClient interface {
	Run(ctx context.Context, prompt string, outputCh chan<- string) error
	// RunText runs the model in plain text mode (no shell tools) — used for
	// structured evaluations like verify steps and success-criteria checks.
	RunText(ctx context.Context, prompt string, outputCh chan<- string) error
}

// Runner executes pipelines step-by-step using a Qwen subprocess.
type Runner struct {
	skills          *skills.Registry
	qwenClient      QwenClient
	notifier        notify.Notifier
	approvalHub     *ApprovalHub
	roiTracker      *ROITracker
	runStore        *RunStore
	ApprovalTimeout time.Duration // overrides step TimeoutMinutes when non-zero (useful in tests)
}

// NewRunner creates a Runner wired with all dependencies.
func NewRunner(
	skillReg *skills.Registry,
	qwenClient QwenClient,
	notifier notify.Notifier,
	hub *ApprovalHub,
	roi *ROITracker,
	store *RunStore,
) *Runner {
	return &Runner{
		skills:      skillReg,
		qwenClient:  qwenClient,
		notifier:    notifier,
		approvalHub: hub,
		roiTracker:  roi,
		runStore:    store,
	}
}

// StartAsync begins pipeline execution in a background goroutine and returns
// the run ID immediately. Use the run ID to subscribe to SSE events.
func (r *Runner) StartAsync(runID string, p *pipeline.Pipeline, params map[string]string) {
	go func() {
		_, _ = r.executeWithID(context.Background(), runID, p, params)
	}()
}

// Execute runs a pipeline and returns the final Run record.
func (r *Runner) Execute(ctx context.Context, p *pipeline.Pipeline, params map[string]string) (*Run, error) {
	return r.executeWithID(ctx, fmt.Sprintf("run-%d", time.Now().UnixNano()), p, params)
}

func (r *Runner) executeWithID(ctx context.Context, runID string, p *pipeline.Pipeline, params map[string]string) (*Run, error) {
	sub := pipeline.SubstituteParams(p, params)
	run := &Run{
		ID:           runID,
		PipelineName: p.Name,
		Status:       RunStatusRunning,
		StartedAt:    time.Now(),
		Steps:        make([]StepResult, 0, len(sub.Steps)),
	}
	record := &RunRecord{Status: RunStatusRunning}
	r.runStore.Store(run.ID, record)
	started := time.Now()

	var prevOutput string
	for i := range sub.Steps {
		step := &sub.Steps[i]
		r.fireEvent(ctx, run, step, notify.StatusRunning, "")

		var output string
		var stepErr error
		switch step.Type {
		case pipeline.StepTypeApproval:
			output, stepErr = r.runApprovalStep(ctx, run, step)
		case pipeline.StepTypeVerify:
			output, stepErr = r.runVerifyStep(ctx, run, step, prevOutput)
		default:
			output, stepErr = r.executeStep(ctx, run, sub, step, prevOutput)
		}

		if stepErr != nil {
			healed := r.runRepairFlow(ctx, run, sub, step, output)
			if !healed {
				run.Steps = append(run.Steps, StepResult{StepID: step.ID, Status: RunStatusFailed, Error: stepErr})
				run.Status = RunStatusFailed
				record.Status = RunStatusFailed
				return run, nil
			}
			output = "repaired and retried successfully"
		}

		run.Steps = append(run.Steps, StepResult{StepID: step.ID, Status: RunStatusSuccess, Output: output})
		r.runStore.AppendLog(run.ID, output)
		prevOutput = output
		r.fireEvent(ctx, run, step, notify.StatusSuccess, output)

		if i < len(sub.Steps)-1 && r.checkSuccessCriteria(ctx, run, sub) {
			run.Status = RunStatusSuccess
			record.Status = RunStatusSuccess
			remaining := len(sub.Steps) - 1 - i
			if err := r.roiTracker.Log(run.ID, p.Name, time.Since(started), p.ManualMinutes, true); err != nil {
				log.Printf("roi log error for run %s: %v", run.ID, err)
			}
			r.notify(ctx, notify.StepEvent{
				RunID:    run.ID,
				Pipeline: p.Name,
				Status:   notify.StatusEarlySuccess,
				Output:   fmt.Sprintf("Goal achieved after step %q — skipping %d remaining step(s)", step.ID, remaining),
				TS:       time.Now().Unix(),
			})
			return run, nil
		}
	}

	run.Status = RunStatusSuccess
	record.Status = RunStatusSuccess
	duration := time.Since(started)

	if err := r.roiTracker.Log(run.ID, p.Name, duration, p.ManualMinutes, true); err != nil {
		log.Printf("roi log error for run %s: %v", run.ID, err)
	}
	r.notify(ctx, notify.StepEvent{
		RunID:    run.ID,
		Pipeline: p.Name,
		Status:   notify.StatusSuccess,
		Output:   fmt.Sprintf("Pipeline completed in %s", duration.Round(time.Second)),
		TS:       time.Now().Unix(),
	})
	return run, nil
}

func (r *Runner) runApprovalStep(ctx context.Context, run *Run, step *pipeline.Step) (string, error) {
	r.fireEvent(ctx, run, step, notify.StatusAwaitingApproval, "")
	timeout := r.resolveTimeout(step)
	if !r.approvalHub.WaitForApproval(run.ID, step.ID, timeout) {
		return "", fmt.Errorf("step %q: approval rejected or timed out", step.ID)
	}
	return "approved", nil
}

func (r *Runner) executeStep(ctx context.Context, run *Run, p *pipeline.Pipeline, step *pipeline.Step, prevOutput string) (string, error) {
	skillContent := ""
	if step.Type == pipeline.StepTypeSkill {
		if s, err := r.skills.Get(step.Skill); err == nil {
			skillContent = s.Content
		}
	}

	prompt, err := executor.BuildStepPrompt(p, step, skillContent, prevOutput)
	if err != nil {
		return "", fmt.Errorf("build step prompt: %w", err)
	}

	outputCh := make(chan string, 128)
	errCh := make(chan error, 1)
	go func() {
		errCh <- r.qwenClient.Run(ctx, prompt, outputCh)
		close(outputCh)
	}()

	var lines []string
	for line := range outputCh {
		lines = append(lines, line)
		r.runStore.AppendLog(run.ID, line)
		r.fireEvent(ctx, run, step, notify.StatusNarration, line)
	}

	if err := <-errCh; err != nil {
		return strings.Join(lines, "\n"), fmt.Errorf("step %q: %w", step.ID, err)
	}
	return strings.Join(lines, "\n"), nil
}

func (r *Runner) runRepairFlow(ctx context.Context, run *Run, p *pipeline.Pipeline, step *pipeline.Step, errOutput string) bool {
	repairPrompt := buildRepairPrompt(p, step, errOutput)
	repairOut := r.runQwenCollect(ctx, run, repairPrompt)
	result := parseRepairResult(repairOut)

	r.notify(ctx, notify.StepEvent{
		RunID: run.ID, Pipeline: run.PipelineName,
		StepID: step.ID, StepName: step.Name,
		Status:     notify.StatusRepairSuggested,
		Diagnosis:  result.Diagnosis,
		FixCommand: result.FixCommand,
		TS:         time.Now().Unix(),
	})

	repairTimeout := defaultApprovalTimeout
	if r.ApprovalTimeout > 0 {
		repairTimeout = r.ApprovalTimeout
	}
	repairKey := step.ID + ":repair"
	if !r.approvalHub.WaitForApproval(run.ID, repairKey, repairTimeout) {
		r.triggerPostmortem(ctx, run, step, errOutput)
		return false
	}

	fixPrompt := "Execute this fix and report the result:\n" + result.FixCommand
	r.runQwenCollect(ctx, run, fixPrompt)

	var retryErr error
	if step.Type == pipeline.StepTypeVerify {
		_, retryErr = r.runVerifyStep(ctx, run, step, errOutput)
	} else {
		_, retryErr = r.executeStep(ctx, run, p, step, errOutput)
	}
	if retryErr != nil {
		r.triggerPostmortem(ctx, run, step, errOutput)
		return false
	}
	return true
}

func (r *Runner) triggerPostmortem(ctx context.Context, run *Run, step *pipeline.Step, errOutput string) {
	prompt := fmt.Sprintf(
		"Pipeline %q failed at step %q.\n\nError:\n%s\n\nRun log:\n%s\n\n"+
			"Provide a post-mortem analysis: root cause, impact, and prevention.",
		run.PipelineName, step.ID, errOutput, r.runStore.GetLog(run.ID),
	)
	analysis := r.runQwenCollect(ctx, run, prompt)
	r.notify(ctx, notify.StepEvent{
		RunID: run.ID, Pipeline: run.PipelineName,
		StepID: step.ID, StepName: step.Name,
		Status: notify.StatusPostmortem, Output: analysis, TS: time.Now().Unix(),
	})
}

// runQwenCollect runs a Qwen prompt and returns all output lines joined.
func (r *Runner) runQwenCollect(ctx context.Context, run *Run, prompt string) string {
	return r.collectWith(ctx, run, prompt, r.qwenClient.Run)
}

func (r *Runner) fireEvent(ctx context.Context, run *Run, step *pipeline.Step, status notify.EventStatus, output string) {
	r.notify(ctx, notify.StepEvent{
		RunID: run.ID, Pipeline: run.PipelineName,
		StepID: step.ID, StepName: step.Name,
		Status: status, Output: output, TS: time.Now().Unix(),
	})
}

func (r *Runner) notify(ctx context.Context, event notify.StepEvent) {
	if err := r.notifier.Notify(ctx, event); err != nil {
		log.Printf("notify error for run %s: %v", event.RunID, err)
	}
}

func (r *Runner) resolveTimeout(step *pipeline.Step) time.Duration {
	if r.ApprovalTimeout > 0 {
		return r.ApprovalTimeout
	}
	if step.TimeoutMinutes > 0 {
		return time.Duration(step.TimeoutMinutes) * time.Minute
	}
	return defaultApprovalTimeout
}

type repairResult struct {
	Diagnosis  string `json:"diagnosis"`
	FixCommand string `json:"fix_command"`
}

func buildRepairPrompt(p *pipeline.Pipeline, step *pipeline.Step, errOutput string) string {
	return fmt.Sprintf(
		"Step %q (id: %s) in pipeline %q failed:\n\n%s\n\n"+
			"Respond with ONLY JSON: {\"diagnosis\": \"...\", \"fix_command\": \"...\"}",
		step.Name, step.ID, p.Name, errOutput,
	)
}

func parseRepairResult(output string) repairResult {
	start := strings.Index(output, "{")
	end := strings.LastIndex(output, "}")
	if start >= 0 && end > start {
		var res repairResult
		if err := json.Unmarshal([]byte(output[start:end+1]), &res); err == nil {
			return res
		}
	}
	return repairResult{Diagnosis: output}
}

// runVerifyStep evaluates the previous step's output against a declared criteria.
// Returns an error (causing the repair flow) when Qwen responds with FAIL.
func (r *Runner) runVerifyStep(ctx context.Context, run *Run, step *pipeline.Step, prevOutput string) (string, error) {
	criteria := step.Params["criteria"]
	prompt := buildVerifyPrompt(criteria, prevOutput)
	output := r.runQwenTextCollect(ctx, run, prompt)
	if isVerifyFail(output) {
		return output, fmt.Errorf("step %q: verify FAIL — %s", step.ID, output)
	}
	return output, nil
}

// checkSuccessCriteria asks Qwen whether the pipeline's overall goal is already
// satisfied given everything logged so far. Returns true only when criteria is
// set and Qwen answers YES.
func (r *Runner) checkSuccessCriteria(ctx context.Context, run *Run, p *pipeline.Pipeline) bool {
	if p.SuccessCriteria == "" {
		return false
	}
	prompt := buildSuccessCriteriaPrompt(p.SuccessCriteria, r.runStore.GetLog(run.ID))
	output := r.runQwenTextCollect(ctx, run, prompt)
	return isSuccessYes(output)
}

// runQwenTextCollect runs a text-only Qwen prompt (no shell tools) and returns
// all output lines joined. Used for structured evaluations that must not execute
// arbitrary commands.
func (r *Runner) runQwenTextCollect(ctx context.Context, run *Run, prompt string) string {
	return r.collectWith(ctx, run, prompt, r.qwenClient.RunText)
}

type qwenRunFn func(context.Context, string, chan<- string) error

// collectWith is the shared pump behind runQwenCollect and runQwenTextCollect.
func (r *Runner) collectWith(ctx context.Context, run *Run, prompt string, fn qwenRunFn) string {
	outputCh := make(chan string, 128)
	errCh := make(chan error, 1)
	go func() {
		errCh <- fn(ctx, prompt, outputCh)
		close(outputCh)
	}()
	var lines []string
	for line := range outputCh {
		lines = append(lines, line)
		r.runStore.AppendLog(run.ID, line)
	}
	if err := <-errCh; err != nil {
		log.Printf("qwen collect error for run %s: %v", run.ID, err)
	}
	return strings.Join(lines, "\n")
}

func buildVerifyPrompt(criteria, prevOutput string) string {
	return fmt.Sprintf(
		"You are a quality gate for a pipeline step.\n\n"+
			"Previous step output:\n%s\n\n"+
			"Criteria to satisfy:\n%s\n\n"+
			"Does the output satisfy the criteria? "+
			"Respond with PASS or FAIL on the first line, then your reasoning.",
		prevOutput, criteria,
	)
}

func buildSuccessCriteriaPrompt(criteria, runLog string) string {
	return fmt.Sprintf(
		"You are evaluating whether a pipeline goal has been fully achieved.\n\n"+
			"Goal:\n%s\n\n"+
			"Pipeline output so far:\n%s\n\n"+
			"Has the goal been fully achieved? "+
			"Answer YES or NO on the first line, then your reasoning.",
		criteria, runLog,
	)
}

// isVerifyFail returns true when Qwen's first line starts with FAIL.
func isVerifyFail(output string) bool {
	first := strings.ToUpper(strings.TrimSpace(strings.SplitN(output, "\n", 2)[0]))
	return strings.HasPrefix(first, "FAIL")
}

// isSuccessYes returns true when Qwen's first line starts with YES.
func isSuccessYes(output string) bool {
	first := strings.ToUpper(strings.TrimSpace(strings.SplitN(output, "\n", 2)[0]))
	return strings.HasPrefix(first, "YES")
}
