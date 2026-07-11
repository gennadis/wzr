package web

import (
	"fmt"
	"io/fs"
	"net/http"
)

// Server is the WZR HTTP server. API handlers are registered in handlers.go.
type Server struct {
	mux      *http.ServeMux
	staticFS fs.FS
}

// NewServer creates a Server that serves static files from staticFS.
func NewServer(staticFS fs.FS) *Server {
	s := &Server{mux: http.NewServeMux(), staticFS: staticFS}
	s.routes()
	return s
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.Handle("GET /static/", http.StripPrefix("/static", http.FileServerFS(s.staticFS)))
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /{$}", s.handlePage("index.html"))
	s.mux.HandleFunc("GET /creator", s.handlePage("creator.html"))
	s.mux.HandleFunc("GET /dashboard", s.handlePage("dashboard.html"))
	s.mux.HandleFunc("GET /templates", s.handlePage("templates.html"))
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "ok")
}

func (s *Server) handlePage(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, s.staticFS, name)
	}
}
