package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"wzr/assets"
	"wzr/internal/config"
	"wzr/internal/mcp"
	"wzr/internal/notify"
	"wzr/internal/pipeline"
	"wzr/internal/qwen"
	"wzr/internal/runner"
	"wzr/internal/skills"
	"wzr/internal/web"
)

func main() {
	defaults := config.Default()
	var (
		port         = flag.String("port", defaults.Port, "HTTP listen port")
		qwenBinary   = flag.String("qwen", defaults.QwenBinary, "path to qwen CLI binary")
		pipelinesDir = flag.String("pipelines", defaults.PipelinesDir, "directory for pipeline YAML files")
		historyFile  = flag.String("history", defaults.HistoryFile, "path to run history JSON file")
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

	if err := os.MkdirAll(cfg.PipelinesDir, 0o750); err != nil {
		log.Fatalf("create pipelines dir %s: %v", cfg.PipelinesDir, err)
	}

	staticFS, err := fs.Sub(assets.WebStaticFS, "web/static")
	if err != nil {
		log.Fatalf("sub web/static FS: %v", err)
	}
	templatesFS, err := fs.Sub(assets.TemplatesFS, "templates")
	if err != nil {
		log.Fatalf("sub templates FS: %v", err)
	}
	skillsFS, err := fs.Sub(assets.SkillsFS, "skills")
	if err != nil {
		log.Fatalf("sub skills FS: %v", err)
	}

	skillReg := skills.NewRegistry(skillsFS)
	mcpReg := mcp.NewRegistry()
	pipeStore := pipeline.NewStore(cfg.PipelinesDir)
	sseHub := notify.NewHub()
	approvalHub := runner.NewApprovalHub()
	roiTracker := runner.NewROITracker(cfg.HistoryFile)
	runStore := runner.NewRunStore()
	qwenClient := qwen.NewClient(cfg.QwenBinary)
	notifier := notify.NewSSENotifier(sseHub)

	r := runner.NewRunner(skillReg, qwenClient, notifier, approvalHub, roiTracker, runStore)

	deps := web.Deps{
		StaticFS:    staticFS,
		TemplatesFS: templatesFS,
		Skills:      skillReg,
		MCPs:        mcpReg,
		PipeStore:   pipeStore,
		Runner:      r,
		SSEHub:      sseHub,
		Approvals:   approvalHub,
		ROI:         roiTracker,
		RunStore:    runStore,
		Qwen:        qwenClient,
	}

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      web.NewServer(deps),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	log.Printf("WZR — WZR's Zen Runtime — listening on :%s", cfg.Port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
