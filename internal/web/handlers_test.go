package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"wzr/internal/mcp"
	"wzr/internal/pipeline"
	"wzr/internal/runner"
	"wzr/internal/skills"
)

func newTestROI(t *testing.T) *runner.ROITracker {
	t.Helper()
	return runner.NewROITracker(filepath.Join(t.TempDir(), "history.json"))
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	skillFS := fstest.MapFS{
		"check-release-readiness.md": {Data: []byte("# check-release-readiness\nCheck Jira.")},
		"update-release-notes.md":    {Data: []byte("# update-release-notes\nUpdate Confluence.")},
	}
	tmplFS := fstest.MapFS{
		"release-manager.yaml": {Data: []byte("name: release-manager\ndescription: Release\nsteps:\n  - id: s1\n    name: Step\n    type: skill\n    skill: foo\n")},
	}
	deps := Deps{
		StaticFS:    fstest.MapFS{},
		TemplatesFS: tmplFS,
		Skills:      skills.NewRegistry(skillFS),
		MCPs:        mcp.NewRegistry(),
		PipeStore:   pipeline.NewStore(t.TempDir()),
	}
	return NewServer(deps)
}

func TestHandleListSkills(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/skills", http.NoBody)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var result []any
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("got %d skills, want 2", len(result))
	}
}

func TestHandleListMCPs(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/mcps", http.NoBody)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var result []any
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result) != 5 {
		t.Errorf("got %d MCPs, want 5", len(result))
	}
}

func TestHandlePipelineRoundTrip(t *testing.T) {
	srv := newTestServer(t)

	yaml := "name: my-pipe\nsteps:\n  - id: s1\n    name: Step\n    type: skill\n    skill: foo\n"

	// Save
	req := httptest.NewRequest(http.MethodPost, "/api/pipelines", strings.NewReader(yaml))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("save status: got %d, body: %s", w.Code, w.Body.String())
	}

	// List
	req = httptest.NewRequest(http.MethodGet, "/api/pipelines", http.NoBody)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list status: got %d", w.Code)
	}
	var names []string
	if err := json.NewDecoder(w.Body).Decode(&names); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(names) != 1 || names[0] != "my-pipe" {
		t.Errorf("list: got %v", names)
	}

	// Get
	req = httptest.NewRequest(http.MethodGet, "/api/pipelines/my-pipe", http.NoBody)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get status: got %d", w.Code)
	}
}

func TestHandleListTemplates(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/templates", http.NoBody)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var result []any
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("got %d templates, want 1", len(result))
	}
}

func TestHandleSavePipeline_InvalidYAML(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/pipelines", bytes.NewReader([]byte("not: valid: yaml: [")))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleStats_Empty(t *testing.T) {
	dir := t.TempDir()
	deps := Deps{
		StaticFS:  fstest.MapFS{},
		Skills:    skills.NewRegistry(fstest.MapFS{}),
		MCPs:      mcp.NewRegistry(),
		PipeStore: pipeline.NewStore(dir),
		ROI:       newTestROI(t),
	}
	srv := NewServer(deps)
	req := httptest.NewRequest(http.MethodGet, "/api/stats", http.NoBody)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
}
