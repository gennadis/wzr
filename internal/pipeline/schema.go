package pipeline

// StepType enumerates the valid step kinds.
type StepType string

const (
	StepTypeSkill    StepType = "skill"
	StepTypeMCP      StepType = "mcp"
	StepTypeApproval StepType = "approval"
)

// Step represents a single pipeline step.
type Step struct {
	ID             string            `yaml:"id"`
	Name           string            `yaml:"name"`
	Type           StepType          `yaml:"type"`
	Skill          string            `yaml:"skill,omitempty"`
	Server         string            `yaml:"server,omitempty"`
	Tool           string            `yaml:"tool,omitempty"`
	Params         map[string]string `yaml:"params,omitempty"`
	DependsOn      string            `yaml:"depends_on,omitempty"`
	TimeoutMinutes int               `yaml:"timeout_minutes,omitempty"`
}

// OnFailure holds pipeline-level failure notification config.
type OnFailure struct {
	Notify string `yaml:"notify,omitempty"`
}

// Pipeline is the top-level YAML structure.
type Pipeline struct {
	Name          string            `yaml:"name"`
	Version       string            `yaml:"version,omitempty"`
	Description   string            `yaml:"description,omitempty"`
	Params        map[string]string `yaml:"params,omitempty"`
	Steps         []Step            `yaml:"steps"`
	OnFailure     OnFailure         `yaml:"on_failure,omitempty"`
	ManualMinutes int               `yaml:"manual_minutes,omitempty"`
}
