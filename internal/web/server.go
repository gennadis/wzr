package web

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"

	"wzr/internal/mcp"
	"wzr/internal/notify"
	"wzr/internal/pipeline"
	"wzr/internal/runner"
	"wzr/internal/skills"
)

// QwenRunner is the interface for calling Qwen from web handlers.
type QwenRunner interface {
	Run(ctx context.Context, prompt string, outputCh chan<- string) error
}

// Deps holds all handler dependencies injected into Server.
type Deps struct {
	StaticFS    fs.FS
	TemplatesFS fs.FS
	Skills      *skills.Registry
	MCPs        *mcp.Registry
	PipeStore   *pipeline.Store
	Runner      *runner.Runner
	SSEHub      *notify.Hub
	Approvals   *runner.ApprovalHub
	ROI         *runner.ROITracker
	RunStore    *runner.RunStore
	Qwen        QwenRunner
}

// Server is the WZR HTTP server.
type Server struct {
	mux  *http.ServeMux
	deps Deps
}

// NewServer creates a Server with the given dependencies.
func NewServer(deps Deps) *Server {
	s := &Server{mux: http.NewServeMux(), deps: deps}
	s.routes()
	return s
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	// Static assets
	s.mux.Handle("GET /static/", http.StripPrefix("/static", http.FileServerFS(s.deps.StaticFS)))

	// Health
	s.mux.HandleFunc("GET /health", s.handleHealth)

	// Page routes (HTML files)
	s.mux.HandleFunc("GET /{$}", s.handlePage("index.html"))
	s.mux.HandleFunc("GET /creator", s.handlePage("creator.html"))
	s.mux.HandleFunc("GET /dashboard", s.handlePage("dashboard.html"))
	s.mux.HandleFunc("GET /templates", s.handlePage("templates.html"))

	// Skills & MCPs
	s.mux.HandleFunc("GET /api/skills", s.handleListSkills)
	s.mux.HandleFunc("GET /api/mcps", s.handleListMCPs)

	// Pipelines
	s.mux.HandleFunc("GET /api/pipelines", s.handleListPipelines)
	s.mux.HandleFunc("GET /api/pipelines/{name}", s.handleGetPipeline)
	s.mux.HandleFunc("POST /api/pipelines", s.handleSavePipeline)
	s.mux.HandleFunc("POST /api/pipelines/{name}/run", s.handleRunPipeline)

	// Run events & controls
	s.mux.HandleFunc("GET /api/runs/{id}/events", s.handleRunEvents)
	s.mux.HandleFunc("POST /api/runs/{id}/steps/{step}/approve", s.handleApprove)
	s.mux.HandleFunc("POST /api/runs/{id}/chat", s.handleRunChat)

	// Templates gallery
	s.mux.HandleFunc("GET /api/templates", s.handleListTemplates)
	s.mux.HandleFunc("POST /api/templates/{name}/clone", s.handleCloneTemplate)

	// Creator
	s.mux.HandleFunc("POST /api/creator/message", s.handleCreatorMessage)
	s.mux.HandleFunc("POST /api/creator/generate", s.handleCreatorGenerate)

	// Stats
	s.mux.HandleFunc("GET /api/stats", s.handleStats)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "ok")
}

func (s *Server) handlePage(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, s.deps.StaticFS, name)
	}
}
