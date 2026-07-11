package pipeline

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Parse decodes YAML bytes into a Pipeline and validates required fields.
func Parse(data []byte) (*Pipeline, error) {
	var p Pipeline
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse pipeline YAML: %w", err)
	}
	if p.Name == "" {
		return nil, errors.New("pipeline name is required")
	}
	if len(p.Steps) == 0 {
		return nil, errors.New("pipeline must have at least one step")
	}
	for i, s := range p.Steps {
		if s.ID == "" {
			return nil, fmt.Errorf("step %d: id is required", i)
		}
		if s.Type == "" {
			return nil, fmt.Errorf("step %q: type is required", s.ID)
		}
	}
	return &p, nil
}

func substitute(s string, params map[string]string) string {
	for k, v := range params {
		s = strings.ReplaceAll(s, "{{ ."+k+" }}", v)
	}
	return s
}

// SubstituteParams returns a deep copy of the pipeline with all {{ .key }}
// placeholders replaced by the corresponding values from params.
func SubstituteParams(p *Pipeline, params map[string]string) *Pipeline {
	out := *p
	out.Steps = make([]Step, len(p.Steps))
	for i, step := range p.Steps {
		step.Skill = substitute(step.Skill, params)
		step.Server = substitute(step.Server, params)
		step.Tool = substitute(step.Tool, params)
		if step.Params != nil {
			expanded := make(map[string]string, len(step.Params))
			for k, v := range step.Params {
				expanded[k] = substitute(v, params)
			}
			step.Params = expanded
		}
		out.Steps[i] = step
	}
	out.OnFailure = OnFailure{Notify: substitute(p.OnFailure.Notify, params)}
	return &out
}
