package executor

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"wzr/internal/pipeline"
)

// commandContext is the exec.CommandContext function, replaced in tests.
var commandContext = exec.CommandContext

// CLIExecutor runs a Qwen-compatible CLI binary as a subprocess.
type CLIExecutor struct {
	// Command is the binary to run (resolved from PATH).
	Command string
	// Args are additional arguments passed to the binary before the prompt.
	Args []string
}

// NewQwen returns a CLIExecutor configured for the qwen binary.
func NewQwen() *CLIExecutor {
	return &CLIExecutor{
		Command: "qwen",
		Args:    []string{"--approval-mode", "auto-edit", "--allowed-tools", "run_shell_command", "--model", "deepseek-v4-flash"},
	}
}

// NewGigacode returns a CLIExecutor configured for the gigacode binary.
func NewGigacode() *CLIExecutor {
	return &CLIExecutor{
		Command: "gigacode",
		Args:    []string{"--approval-mode", "auto-edit", "--allowed-tools", "run_shell_command", "--model", "vllm/Qwen3-Coder-Next-262k"},
	}
}

// Run spawns the binary in agentic mode using the executor's configured Args.
// Use this for pipeline step execution where the binary needs to run commands.
func (c *CLIExecutor) Run(ctx context.Context, prompt string, outputCh chan<- string) error {
	return c.spawn(ctx, prompt, outputCh, c.Args)
}

// RunText spawns the binary in plain text-generation mode with no tools allowed.
// Use this for creator, chat, and any call that must return structured text only.
func (c *CLIExecutor) RunText(ctx context.Context, prompt string, outputCh chan<- string) error {
	return c.spawn(ctx, prompt, outputCh, []string{"--allowed-tools", ""})
}

func (c *CLIExecutor) spawn(ctx context.Context, prompt string, outputCh chan<- string, args []string) error {
	fullArgs := make([]string, 0, 4+len(args))
	fullArgs = append(fullArgs, "-p", prompt, "--output-format", "text")
	fullArgs = append(fullArgs, args...)
	cmd := commandContext(ctx, c.Command, fullArgs...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", c.Command, err)
	}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		outputCh <- scanner.Text()
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read %s stdout: %w", c.Command, err)
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("%s: %w", c.Command, err)
	}
	return nil
}

// BuildStepPrompt assembles the prompt string sent to the executor for a pipeline step.
func BuildStepPrompt(p *pipeline.Pipeline, step *pipeline.Step, skillContent, prevOutput string) (string, error) {
	paramsJSON, err := json.Marshal(step.Params)
	if err != nil {
		return "", fmt.Errorf("marshal step params: %w", err)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "You are executing step %q (%s) in pipeline %q.\n\n", step.Name, step.ID, p.Name)

	if prevOutput != "" {
		fmt.Fprintf(&b, "Context from previous step:\n%s\n\n", prevOutput)
	}

	b.WriteString("Your task:\n")
	if skillContent != "" {
		fmt.Fprintf(&b, "Use the following skill and execute it autonomously:\n%s\nWith parameters: %s\n", skillContent, paramsJSON)
	} else {
		fmt.Fprintf(&b, "Call MCP server %q, tool %q with these parameters:\n%s\nReport what you did and the result.\n",
			step.Server, step.Tool, paramsJSON)
	}

	return b.String(), nil
}
