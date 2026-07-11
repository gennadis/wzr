package config

import "testing"

func TestDefault(t *testing.T) {
	c := Default()
	if c.Port != "8080" {
		t.Errorf("Port: got %q, want %q", c.Port, "8080")
	}
	if c.QwenBinary != "qwen" {
		t.Errorf("QwenBinary: got %q, want %q", c.QwenBinary, "qwen")
	}
	if c.PipelinesDir != "./pipelines" {
		t.Errorf("PipelinesDir: got %q, want %q", c.PipelinesDir, "./pipelines")
	}
	if c.HistoryFile != "./run_history.json" {
		t.Errorf("HistoryFile: got %q, want %q", c.HistoryFile, "./run_history.json")
	}
}
