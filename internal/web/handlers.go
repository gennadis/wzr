package web

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"wzr/internal/pipeline"
)

// --- helpers ---

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "encode error", http.StatusInternalServerError)
	}
}

func jsonErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// --- skills ---

func (s *Server) handleListSkills(w http.ResponseWriter, _ *http.Request) {
	list, err := s.deps.Skills.List()
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonOK(w, list)
}

// --- mcps ---

func (s *Server) handleListMCPs(w http.ResponseWriter, _ *http.Request) {
	jsonOK(w, s.deps.MCPs.List())
}

// --- pipelines ---

func (s *Server) handleListPipelines(w http.ResponseWriter, _ *http.Request) {
	names, err := s.deps.PipeStore.List()
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonOK(w, names)
}

func (s *Server) handleGetPipeline(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	p, err := s.deps.PipeStore.Load(name)
	if err != nil {
		jsonErr(w, http.StatusNotFound, err.Error())
		return
	}
	jsonOK(w, p)
}

func (s *Server) handleSavePipeline(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "read body: "+err.Error())
		return
	}
	p, err := pipeline.Parse(body)
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid pipeline YAML: "+err.Error())
		return
	}
	if err := s.deps.PipeStore.Save(p); err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, map[string]string{"name": p.Name})
}

func (s *Server) handleRunPipeline(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	p, err := s.deps.PipeStore.Load(name)
	if err != nil {
		jsonErr(w, http.StatusNotFound, "pipeline not found: "+err.Error())
		return
	}

	var body struct {
		Params map[string]string `json:"params"`
	}
	if decErr := json.NewDecoder(r.Body).Decode(&body); decErr != nil {
		body.Params = nil // params optional
	}

	runID := fmt.Sprintf("run-%d", time.Now().UnixNano())
	s.deps.Runner.StartAsync(runID, p, body.Params)

	w.WriteHeader(http.StatusAccepted)
	jsonOK(w, map[string]string{"run_id": runID})
}

// --- run events (SSE) ---

func (s *Server) handleRunEvents(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	sseHeaders(w)
	ch := s.deps.SSEHub.Subscribe(runID)
	defer s.deps.SSEHub.Unsubscribe(runID)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case event, open := <-ch:
			if !open {
				return
			}
			writeSSEEvent(w, flusher, event)
		}
	}
}

// --- approve ---

func (s *Server) handleApprove(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	stepID := r.PathValue("step")

	var body struct {
		Approved bool `json:"approved"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	s.deps.Approvals.Respond(runID, stepID, body.Approved)
	w.WriteHeader(http.StatusOK)
}

// --- post-run chat ---

func (s *Server) handleRunChat(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")

	var body struct {
		Question string `json:"question"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Question == "" {
		jsonErr(w, http.StatusBadRequest, "question required")
		return
	}

	record, ok := s.deps.RunStore.Get(runID)
	if !ok {
		jsonErr(w, http.StatusNotFound, "run not found")
		return
	}

	prompt := fmt.Sprintf(
		"The following is the complete log of pipeline run %q:\n\n%s\n\nUser question: %s\n\nAnswer concisely.",
		runID, record.Log, body.Question,
	)

	outputCh := make(chan string, 128)
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.deps.Qwen.Run(r.Context(), prompt, outputCh)
		close(outputCh)
	}()

	var lines []string
	for line := range outputCh {
		lines = append(lines, line)
	}
	if err := <-errCh; err != nil {
		jsonErr(w, http.StatusInternalServerError, "qwen error: "+err.Error())
		return
	}
	jsonOK(w, map[string]string{"answer": strings.Join(lines, "\n")})
}

// --- templates ---

func (s *Server) handleListTemplates(w http.ResponseWriter, _ *http.Request) {
	entries, err := fs.ReadDir(s.deps.TemplatesFS, ".")
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	type templateInfo struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		StepCount   int    `json:"step_count"`
	}

	out := make([]templateInfo, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		data, readErr := fs.ReadFile(s.deps.TemplatesFS, e.Name())
		if readErr != nil {
			continue
		}
		p, parseErr := pipeline.Parse(data)
		if parseErr != nil {
			continue
		}
		out = append(out, templateInfo{
			Name:        p.Name,
			Description: p.Description,
			StepCount:   len(p.Steps),
		})
	}
	jsonOK(w, out)
}

func (s *Server) handleCloneTemplate(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	data, err := fs.ReadFile(s.deps.TemplatesFS, name+".yaml")
	if err != nil {
		jsonErr(w, http.StatusNotFound, "template not found")
		return
	}
	p, err := pipeline.Parse(data)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "parse template: "+err.Error())
		return
	}
	if err := s.deps.PipeStore.Save(p); err != nil {
		jsonErr(w, http.StatusInternalServerError, "save pipeline: "+err.Error())
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, map[string]string{"name": p.Name})
}

// --- creator ---

func (s *Server) handleCreatorMessage(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Message string              `json:"message"`
		Steps   []any               `json:"steps"`
		History []map[string]string `json:"history"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Message == "" {
		jsonErr(w, http.StatusBadRequest, "message required")
		return
	}

	skillList := s.buildSkillList()
	stepsJSON, _ := json.Marshal(body.Steps)

	var historyStr strings.Builder
	for _, h := range body.History {
		label := "Human"
		if h["role"] == "assistant" {
			label = "Assistant"
		}
		fmt.Fprintf(&historyStr, "[%s]: %s\n", label, h["text"])
	}

	prompt := "You are a WZR pipeline builder assistant.\n\n" +
		"Available skills: " + skillList + "\n" +
		"Available MCP servers: bitbucket (list_pull_requests, get_pr, merge_pr), " +
		"jira (search_issues, transition_issue, get_issue), jenkins (trigger_build, get_build_status), " +
		"confluence (get_page, create_page, update_page), postgres (query, list_tables)\n\n" +
		"Steps added so far: " + string(stepsJSON) + "\n\n" +
		"Chat so far:\n" + historyStr.String() + "\n" +
		"Latest message from human: " + body.Message + "\n\n" +
		"Reply rules (follow exactly):\n" +
		"- If the message describes a pipeline step: output ONLY a single JSON object, no prose, no markdown fences:\n" +
		`{"id":"kebab-id","name":"Step Name","type":"skill|mcp|approval","skill":"","server":"","tool":"","params":{}}` + "\n" +
		"- If the message is conversational (greeting, question, unclear): reply with a short plain-text message.\n" +
		"- Suggest exactly ONE step per reply. Never output more than one JSON object.\n" +
		"- type must be exactly one of: skill, mcp, approval. No other values."

	answer, err := s.runQwenCollect(r, prompt)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "qwen error: "+err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	_, _ = fmt.Fprint(w, answer)
}

func (s *Server) handleCreatorGenerate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Description == "" {
		jsonErr(w, http.StatusBadRequest, "description required")
		return
	}

	skillList := s.buildSkillList()

	prompt := fmt.Sprintf(
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
		body.Description, skillList,
	)

	yaml, err := s.runQwenCollect(r, prompt)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "qwen error: "+err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	_, _ = fmt.Fprint(w, yaml)
}

// --- stats ---

func (s *Server) handleStats(w http.ResponseWriter, _ *http.Request) {
	stats, err := s.deps.ROI.Stats()
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonOK(w, stats)
}

// --- internal helpers ---

func (s *Server) buildSkillList() string {
	list, err := s.deps.Skills.List()
	if err != nil {
		return ""
	}
	names := make([]string, 0, len(list))
	for _, sk := range list {
		names = append(names, sk.Name)
	}
	return strings.Join(names, ", ")
}

func (s *Server) runQwenCollect(r *http.Request, prompt string) (string, error) {
	outputCh := make(chan string, 128)
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.deps.Qwen.Run(r.Context(), prompt, outputCh)
		close(outputCh)
	}()
	var lines []string
	for line := range outputCh {
		lines = append(lines, line)
	}
	if err := <-errCh; err != nil {
		return strings.Join(lines, "\n"), err
	}
	return strings.Join(lines, "\n"), nil
}
