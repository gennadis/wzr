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
	"wzr/internal/prompts"
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

	outputCh := make(chan string, 128)
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.deps.Qwen.RunText(r.Context(), prompts.BuildRunQA(runID, record.Log, body.Question), outputCh)
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

	answer, err := s.runQwenCollect(r, prompts.BuildCreatorChat(skillList, string(stepsJSON), historyStr.String(), body.Message))
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

	yaml, err := s.runQwenCollect(r, prompts.BuildCreatorGenerate(body.Description, skillList))
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
		errCh <- s.deps.Qwen.RunText(r.Context(), prompt, outputCh)
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
