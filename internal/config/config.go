package config

// Config holds all WZR runtime configuration.
type Config struct {
	Port         string
	UseGigacode  bool
	PipelinesDir string
	HistoryFile  string
}

// Default returns a Config with sensible defaults.
func Default() Config {
	return Config{
		Port:         "8080",
		PipelinesDir: "pipelines",
		HistoryFile:  "run_history.json",
	}
}
