package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"time"

	"wzr/assets"
	"wzr/internal/config"
	"wzr/internal/pipeline"
	"wzr/internal/web"
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
		store := pipeline.NewStore(cfg.PipelinesDir)
		p, err := store.Load(*dryRun)
		if err != nil {
			log.Fatalf("dry-run: %v", err)
		}
		fmt.Printf("Pipeline: %s (v%s)\n", p.Name, p.Version)
		fmt.Printf("Steps: %d\n", len(p.Steps))
		for _, s := range p.Steps {
			fmt.Printf("  [%s] %s (%s)\n", s.Type, s.Name, s.ID)
		}
		return
	}

	staticFS, err := fs.Sub(assets.WebStaticFS, "web/static")
	if err != nil {
		log.Fatalf("sub web/static FS: %v", err)
	}

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      web.NewServer(staticFS),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	log.Printf("WZR — WZR's Zen Runtime — listening on :%s", cfg.Port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
