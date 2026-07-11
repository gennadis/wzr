package pipeline

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
		check   func(t *testing.T, p *Pipeline)
	}{
		{
			name: "valid minimal pipeline",
			yaml: `
name: test-pipeline
steps:
  - id: step1
    name: Do something
    type: skill
    skill: my-skill
`,
			check: func(t *testing.T, p *Pipeline) {
				if p.Name != "test-pipeline" {
					t.Errorf("Name: got %q", p.Name)
				}
				if len(p.Steps) != 1 {
					t.Fatalf("Steps: got %d", len(p.Steps))
				}
				if p.Steps[0].ID != "step1" {
					t.Errorf("Step ID: got %q", p.Steps[0].ID)
				}
				if p.Steps[0].Type != StepTypeSkill {
					t.Errorf("Step Type: got %q", p.Steps[0].Type)
				}
			},
		},
		{
			name: "all step types",
			yaml: `
name: multi-type
steps:
  - id: s1
    name: Skill step
    type: skill
    skill: foo
  - id: s2
    name: MCP step
    type: mcp
    server: jira
    tool: search_issues
  - id: s3
    name: Approval gate
    type: approval
    timeout_minutes: 30
`,
			check: func(t *testing.T, p *Pipeline) {
				if len(p.Steps) != 3 {
					t.Fatalf("Steps: got %d, want 3", len(p.Steps))
				}
				if p.Steps[2].TimeoutMinutes != 30 {
					t.Errorf("TimeoutMinutes: got %d", p.Steps[2].TimeoutMinutes)
				}
			},
		},
		{
			name: "manual_minutes and description",
			yaml: `
name: with-roi
description: A pipeline with ROI tracking
manual_minutes: 60
steps:
  - id: s1
    type: skill
    skill: foo
`,
			check: func(t *testing.T, p *Pipeline) {
				if p.ManualMinutes != 60 {
					t.Errorf("ManualMinutes: got %d", p.ManualMinutes)
				}
				if p.Description != "A pipeline with ROI tracking" {
					t.Errorf("Description: got %q", p.Description)
				}
			},
		},
		{
			name:    "missing name",
			yaml:    "steps:\n  - id: s1\n    type: skill\n",
			wantErr: true,
		},
		{
			name:    "missing steps",
			yaml:    "name: empty\n",
			wantErr: true,
		},
		{
			name:    "step missing id",
			yaml:    "name: p\nsteps:\n  - type: skill\n",
			wantErr: true,
		},
		{
			name:    "step missing type",
			yaml:    "name: p\nsteps:\n  - id: s1\n",
			wantErr: true,
		},
		{
			name:    "invalid yaml",
			yaml:    "name: [unclosed",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := Parse([]byte(tc.yaml))
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.check != nil {
				tc.check(t, p)
			}
		})
	}
}

func TestSubstituteParams(t *testing.T) {
	p := &Pipeline{
		Name: "release-{{ .project }}",
		Steps: []Step{
			{
				ID:    "trigger",
				Type:  StepTypeMCP,
				Server: "jenkins",
				Tool:  "trigger_build",
				Params: map[string]string{
					"job": "release/{{ .project }}",
					"tag": "{{ .version }}",
				},
			},
			{
				ID:    "notify",
				Type:  StepTypeSkill,
				Skill: "notify-{{ .project }}",
			},
		},
		OnFailure: OnFailure{Notify: "Pipeline failed for {{ .project }}"},
	}

	out := SubstituteParams(p, map[string]string{
		"project": "myapp",
		"version": "1.2.3",
	})

	if out.Name != "release-{{ .project }}" {
		t.Errorf("Name should not be substituted, got %q", out.Name)
	}
	if out.Steps[0].Params["job"] != "release/myapp" {
		t.Errorf("job param: got %q", out.Steps[0].Params["job"])
	}
	if out.Steps[0].Params["tag"] != "1.2.3" {
		t.Errorf("tag param: got %q", out.Steps[0].Params["tag"])
	}
	if out.Steps[1].Skill != "notify-myapp" {
		t.Errorf("skill: got %q", out.Steps[1].Skill)
	}
	if out.OnFailure.Notify != "Pipeline failed for myapp" {
		t.Errorf("on_failure.notify: got %q", out.OnFailure.Notify)
	}

	// original must not be mutated
	if p.Steps[0].Params["job"] != "release/{{ .project }}" {
		t.Error("original pipeline was mutated")
	}
}

func TestSubstituteParams_noParams(t *testing.T) {
	p := &Pipeline{
		Name:  "static",
		Steps: []Step{{ID: "s1", Type: StepTypeSkill, Skill: "foo"}},
	}
	out := SubstituteParams(p, nil)
	if out.Steps[0].Skill != "foo" {
		t.Errorf("skill: got %q", out.Steps[0].Skill)
	}
}
