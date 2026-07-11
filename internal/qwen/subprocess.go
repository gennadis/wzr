package qwen

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"wzr/internal/pipeline"
)

// Client wraps the Qwen Code CLI binary.
type Client struct {
	BinaryPath string
}

// NewClient creates a Client using the given binary path.
func NewClient(binaryPath string) *Client {
	return &Client{BinaryPath: binaryPath}
}

// Run spawns qwen in agentic mode (auto-approve, run_shell_command allowed).
// Use this for pipeline step execution where Qwen needs to run commands.
func (c *Client) Run(ctx context.Context, prompt string, outputCh chan<- string) error {
	return c.spawn(ctx, prompt, outputCh,
		"--approval-mode", "auto-edit",
		"--allowed-tools", "run_shell_command",
	)
}

// RunText spawns qwen in plain text-generation mode with no tools allowed.
// Use this for creator, chat, and any call that must return structured text only.
func (c *Client) RunText(ctx context.Context, prompt string, outputCh chan<- string) error {
	return c.spawn(ctx, prompt, outputCh, "--allowed-tools", "")
}

func (c *Client) spawn(ctx context.Context, prompt string, outputCh chan<- string, extraArgs ...string) error {
	args := append([]string{"-p", prompt, "--output-format", "text"}, extraArgs...)
	cmd := exec.CommandContext(ctx, c.BinaryPath, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start qwen: %w", err)
	}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		outputCh <- scanner.Text()
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read qwen stdout: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("qwen: %w", err)
	}
	return nil
}

// BuildStepPrompt assembles the prompt string sent to Qwen for a pipeline step.
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
