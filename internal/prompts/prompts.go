package prompts

import (
	"encoding/json"
	"fmt"
	"strings"

	"wzr/internal/pipeline"
)

// BuildStepPrompt assembles the prompt sent to the AI for a regular pipeline step.
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

// BuildRepairDiagnosis asks the AI to diagnose a failed step and return a fix
// command as a JSON object.
func BuildRepairDiagnosis(p *pipeline.Pipeline, step *pipeline.Step, errOutput string) string {
	return fmt.Sprintf(
		"Step %q (id: %s) in pipeline %q failed:\n\n%s\n\n"+
			"Respond with ONLY JSON: {\"diagnosis\": \"...\", \"fix_command\": \"...\"}",
		step.Name, step.ID, p.Name, errOutput,
	)
}

// BuildRepairFix builds the prompt that executes an approved fix command.
func BuildRepairFix(fixCommand string) string {
	return "Execute this fix and report the result:\n" + fixCommand
}

// BuildPostmortem builds the prompt for a root-cause post-mortem after an
// unrecoverable step failure.
func BuildPostmortem(pipelineName, stepID, errOutput, runLog string) string {
	return fmt.Sprintf(
		"Pipeline %q failed at step %q.\n\nError:\n%s\n\nRun log:\n%s\n\n"+
			"Provide a post-mortem analysis: root cause, impact, and prevention.",
		pipelineName, stepID, errOutput, runLog,
	)
}

// BuildVerify builds the PASS/FAIL quality-gate prompt for a verify step.
func BuildVerify(criteria, prevOutput string) string {
	return fmt.Sprintf(
		"You are a quality gate for a pipeline step.\n\n"+
			"Previous step output:\n%s\n\n"+
			"Criteria to satisfy:\n%s\n\n"+
			"Does the output satisfy the criteria? "+
			"Respond with PASS or FAIL on the first line, then your reasoning.",
		prevOutput, criteria,
	)
}

// BuildSuccessCriteria builds the YES/NO prompt that checks whether the
// pipeline goal has already been achieved, enabling early exit.
func BuildSuccessCriteria(criteria, runLog string) string {
	return fmt.Sprintf(
		"You are evaluating whether a pipeline goal has been fully achieved.\n\n"+
			"Goal:\n%s\n\n"+
			"Pipeline output so far:\n%s\n\n"+
			"Has the goal been fully achieved? "+
			"Answer YES or NO on the first line, then your reasoning.",
		criteria, runLog,
	)
}

// BuildRunQA builds the prompt for answering a user question about a run log.
func BuildRunQA(runID, log, question string) string {
	return fmt.Sprintf(
		"The following is the complete log of pipeline run %q:\n\n%s\n\nUser question: %s\n\nAnswer concisely.",
		runID, log, question,
	)
}

// BuildCreatorChat builds the prompt for the pipeline builder assistant chat.
// history is a pre-formatted string of prior turns; stepsJSON is the current
// step list serialized as JSON.
func BuildCreatorChat(skillList, stepsJSON, history, message string) string {
	const mcpList = "bitbucket (list_pull_requests, get_pr, merge_pr), " +
		"jira (search_issues, transition_issue, get_issue), " +
		"jenkins (trigger_build, get_build_status), " +
		"confluence (get_page, create_page, update_page), " +
		"postgres (query, list_tables)"

	return "You are a WZR pipeline builder assistant.\n\n" +
		"Available skills: " + skillList + "\n" +
		"Available MCP servers: " + mcpList + "\n\n" +
		"Steps added so far: " + stepsJSON + "\n\n" +
		"Chat so far:\n" + history + "\n" +
		"Latest message from human: " + message + "\n\n" +
		"Reply rules (follow exactly):\n" +
		"- If the message describes a pipeline step: output ONLY a single JSON object, no prose, no markdown fences:\n" +
		`{"id":"kebab-id","name":"Step Name","type":"skill|mcp|approval","skill":"","server":"","tool":"","params":{}}` + "\n" +
		"- If the message is conversational (greeting, question, unclear): reply with a short plain-text message.\n" +
		"- Suggest exactly ONE step per reply. Never output more than one JSON object.\n" +
		"- type must be exactly one of: skill, mcp, approval. No other values."
}

// BuildCreatorGenerate builds the prompt for generating a complete pipeline
// YAML from a plain-language description.
func BuildCreatorGenerate(description, skillList string) string {
	return fmt.Sprintf(
		"Generate a WZR pipeline YAML for: %q\n\n"+
			"Available skills: %s\n"+
			"Available MCP servers: bitbucket, jira, jenkins, confluence, postgres\n\n"+
			"STRICT RULES — violating any rule makes the pipeline unparseable:\n"+
			"1. Every value under 'params:' (both top-level and per-step) MUST be a flat string. NEVER use nested maps or lists.\n"+
			"   WRONG:  parameters:\\n    VERSION: \"1.0\"   ← nested map, forbidden\n"+
			"   RIGHT:  parameters: \"VERSION=1.0\"        ← flat string, correct\n"+
			"2. Parameter references use {{ .param_name }} syntax (with spaces, lowercase dot).\n"+
			"   WRONG:  {{.Params.version}}  or  {{.release_version}}\n"+
			"   RIGHT:  {{ .version }}\n"+
			"3. timeout_minutes is only valid on steps with type: approval.\n\n"+
			"YAML format (follow exactly):\n"+
			"name: pipeline-name\n"+
			"version: \"1.0\"\n"+
			"description: one line description\n"+
			"manual_minutes: N\n"+
			"params:\n"+
			"  param_name: \"\"\n"+
			"steps:\n"+
			"  - id: kebab-step-id\n"+
			"    name: Human readable name\n"+
			"    type: skill\n"+
			"    skill: skill-name\n"+
			"    params:\n"+
			"      key: \"{{ .param_name }}\"\n"+
			"  - id: mcp-step\n"+
			"    name: MCP step\n"+
			"    type: mcp\n"+
			"    server: bitbucket\n"+
			"    tool: list_pull_requests\n"+
			"    params:\n"+
			"      repo: \"{{ .param_name }}\"\n"+
			"      state: MERGED\n"+
			"  - id: approval-step\n"+
			"    name: Human approval\n"+
			"    type: approval\n"+
			"    timeout_minutes: 30\n\n"+
			"Respond with ONLY the YAML. No markdown fences. No explanation.",
		description, skillList,
	)
}
