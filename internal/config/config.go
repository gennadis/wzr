package config

import (
	"os"
	"path/filepath"
)

// Config holds all WZR runtime configuration.
type Config struct {
	Port         string
	QwenBinary   string
	PipelinesDir string
	HistoryFile  string
}

// Default returns a Config with sensible defaults.
// Runtime data (pipelines, history) lives in ~/.wzr/ so it stays out of the project directory.
func Default() Config {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	dataDir := filepath.Join(home, ".wzr")
	return Config{
		Port:         "8080",
		QwenBinary:   "qwen",
		PipelinesDir: filepath.Join(dataDir, "pipelines"),
		HistoryFile:  filepath.Join(dataDir, "run_history.json"),
	}
}
