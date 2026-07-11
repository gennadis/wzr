package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"wzr/internal/config"
)

func main() {
	var (
		port         = flag.String("port", "8080", "HTTP listen port")
		qwenBinary   = flag.String("qwen", "qwen", "path to qwen CLI binary")
		pipelinesDir = flag.String("pipelines", "./pipelines", "directory for pipeline YAML files")
		historyFile  = flag.String("history", "./run_history.json", "path to run history JSON file")
		dryRun       = flag.String("dry-run", "", "parse named pipeline, print struct, and exit")
	)
	flag.Parse()

	cfg := config.Config{
		Port:         *port,
		QwenBinary:   *qwenBinary,
		PipelinesDir: *pipelinesDir,
		HistoryFile:  *historyFile,
	}

	if *dryRun != "" {
		// Pipeline parsing is implemented in Task 2.
		// This flag is wired here so Task 12 can use it without touching main.go.
		fmt.Printf("--dry-run: pipeline %q would be loaded from %s (parser not yet implemented)\n", *dryRun, cfg.PipelinesDir)
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})

	addr := ":" + cfg.Port
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	log.Printf("WZR — WZR's Zen Runtime — listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
