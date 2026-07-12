package executor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"wzr/internal/pipeline"
)

func makeScript(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "fake-qwen.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil { //nolint:gosec // script must be executable to run as test stub
		t.Fatalf("write script: %v", err)
	}
	return path
}

func TestRun_Streams(t *testing.T) {
	script := makeScript(t, "echo line1\necho line2\necho line3\n")
	c := &CLIExecutor{Command: script}
	ch := make(chan string, 10)

	if err := c.Run(context.Background(), "prompt", ch); err != nil {
		t.Fatalf("Run: %v", err)
	}
	close(ch)

	var lines []string
	for l := range ch {
		lines = append(lines, l)
	}
	if len(lines) != 3 {
		t.Errorf("got %d lines, want 3: %v", len(lines), lines)
	}
	if lines[0] != "line1" || lines[1] != "line2" || lines[2] != "line3" {
		t.Errorf("unexpected lines: %v", lines)
	}
}

func TestRun_NonZeroExit(t *testing.T) {
	script := makeScript(t, "echo error output\nexit 1\n")
	c := &CLIExecutor{Command: script}
	ch := make(chan string, 10)

	err := c.Run(context.Background(), "prompt", ch)
	if err == nil {
		t.Error("expected error on non-zero exit")
	}
}

func TestRun_ContextCancel(t *testing.T) {
	script := makeScript(t, "sleep 30\n")
	c := &CLIExecutor{Command: script}
	ch := make(chan string, 10)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := c.Run(ctx, "prompt", ch)
	if err == nil {
		t.Error("expected error on canceled context")
	}
}

func TestBuildStepPrompt_Skill(t *testing.T) {
	p := &pipeline.Pipeline{Name: "release-manager"}
	step := &pipeline.Step{
		ID:    "check-readiness",
		Name:  "Check Jira readiness",
		Type:  pipeline.StepTypeSkill,
		Skill: "check-release-readiness",
		Params: map[string]string{
			"project": "MYAPP",
			"version": "1.2.3",
		},
	}

	prompt, err := BuildStepPrompt(p, step, "# skill content here", "prev step output")
	if err != nil {
		t.Fatalf("BuildStepPrompt: %v", err)
	}
	for _, want := range []string{"release-manager", "check-readiness", "skill content here", "prev step output", "MYAPP"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}

func TestBuildStepPrompt_MCP(t *testing.T) {
	p := &pipeline.Pipeline{Name: "release-manager"}
	step := &pipeline.Step{
		ID:     "verify-prs",
		Name:   "Verify PRs",
		Type:   pipeline.StepTypeMCP,
		Server: "bitbucket",
		Tool:   "list_pull_requests",
		Params: map[string]string{"state": "open"},
	}

	prompt, err := BuildStepPrompt(p, step, "", "")
	if err != nil {
		t.Fatalf("BuildStepPrompt: %v", err)
	}
	for _, want := range []string{"bitbucket", "list_pull_requests", "open"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
	if strings.Contains(prompt, "Context from previous step") {
		t.Error("should not include prev output section when empty")
	}
}

func TestBuildStepPrompt_NoPrevOutput(t *testing.T) {
	p := &pipeline.Pipeline{Name: "p"}
	step := &pipeline.Step{ID: "s1", Name: "Step", Type: pipeline.StepTypeSkill}

	prompt, err := BuildStepPrompt(p, step, "skill", "")
	if err != nil {
		t.Fatalf("BuildStepPrompt: %v", err)
	}
	if strings.Contains(prompt, "Context from previous step") {
		t.Error("should not include prev output section when empty")
	}
}
