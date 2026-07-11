package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Store persists pipelines as YAML files in a directory.
type Store struct {
	dir string
}

// NewStore creates a Store rooted at dir.
func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

// Save marshals p to YAML and writes it to <dir>/<p.Name>.yaml.
func (s *Store) Save(p *Pipeline) error {
	data, err := yaml.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshal pipeline %q: %w", p.Name, err)
	}
	path := filepath.Join(s.dir, p.Name+".yaml")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write pipeline %q: %w", p.Name, err)
	}
	return nil
}

// Load reads and parses <dir>/<name>.yaml.
func (s *Store) Load(name string) (*Pipeline, error) {
	path := filepath.Join(s.dir, name+".yaml")
	data, err := os.ReadFile(path) //nolint:gosec // path is constructed from trusted config dir + validated name
	if err != nil {
		return nil, fmt.Errorf("read pipeline %q: %w", name, err)
	}
	p, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse pipeline %q: %w", name, err)
	}
	return p, nil
}

// List returns the names of all pipelines in the store directory.
func (s *Store) List() ([]string, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("list pipelines: %w", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
			names = append(names, strings.TrimSuffix(e.Name(), ".yaml"))
		}
	}
	return names, nil
}
