package pipeline

// StepType enumerates the valid step kinds.
type StepType string

const (
	StepTypeSkill    StepType = "skill"
	StepTypeMCP      StepType = "mcp"
	StepTypeApproval StepType = "approval"
	StepTypeVerify   StepType = "verify"
)

// Step represents a single pipeline step.
type Step struct {
	ID             string            `yaml:"id"              json:"id"`
	Name           string            `yaml:"name"            json:"name"`
	Type           StepType          `yaml:"type"            json:"type"`
	Skill          string            `yaml:"skill,omitempty" json:"skill,omitempty"`
	Server         string            `yaml:"server,omitempty" json:"server,omitempty"`
	Tool           string            `yaml:"tool,omitempty"  json:"tool,omitempty"`
	Params         map[string]string `yaml:"params,omitempty" json:"params,omitempty"`
	DependsOn      string            `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	TimeoutMinutes int               `yaml:"timeout_minutes,omitempty" json:"timeout_minutes,omitempty"`
}

// OnFailure holds pipeline-level failure notification config.
type OnFailure struct {
	Notify string `yaml:"notify,omitempty" json:"notify,omitempty"`
}

// Pipeline is the top-level YAML structure.
type Pipeline struct {
	Name            string            `yaml:"name"                     json:"name"`
	Version         string            `yaml:"version,omitempty"        json:"version,omitempty"`
	Description     string            `yaml:"description,omitempty"    json:"description,omitempty"`
	Params          map[string]string `yaml:"params,omitempty"         json:"params,omitempty"`
	Steps           []Step            `yaml:"steps"                    json:"steps"`
	OnFailure       OnFailure         `yaml:"on_failure,omitempty"     json:"on_failure"`
	ManualMinutes   int               `yaml:"manual_minutes,omitempty" json:"manual_minutes,omitempty"`
	SuccessCriteria string            `yaml:"success_criteria,omitempty" json:"success_criteria,omitempty"`
}
