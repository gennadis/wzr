package config

import (
	"strings"
	"testing"
)

func TestDefault(t *testing.T) {
	c := Default()
	if c.Port != "8080" {
		t.Errorf("Port: got %q, want %q", c.Port, "8080")
	}
	if c.QwenBinary != "qwen" {
		t.Errorf("QwenBinary: got %q, want %q", c.QwenBinary, "qwen")
	}
	if !strings.HasSuffix(c.PipelinesDir, "pipelines") {
		t.Errorf("PipelinesDir: got %q, expected suffix 'pipelines'", c.PipelinesDir)
	}
	if !strings.HasSuffix(c.HistoryFile, "run_history.json") {
		t.Errorf("HistoryFile: got %q, expected suffix 'run_history.json'", c.HistoryFile)
	}
}
