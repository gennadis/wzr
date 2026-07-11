package skills

import (
	"fmt"
	"io/fs"
	"strings"
)

// Skill holds a skill's name and its markdown content.
type Skill struct {
	Name    string
	Content string
}

// Registry loads skills from an fs.FS whose root contains *.md files.
type Registry struct {
	fsys fs.FS
}

// NewRegistry creates a Registry reading from fsys.
func NewRegistry(fsys fs.FS) *Registry {
	return &Registry{fsys: fsys}
}

// List returns all skills found in the registry FS.
func (r *Registry) List() ([]Skill, error) {
	entries, err := fs.ReadDir(r.fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("list skills: %w", err)
	}
	var skills []Skill
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		content, err := fs.ReadFile(r.fsys, e.Name())
		if err != nil {
			return nil, fmt.Errorf("read skill %q: %w", name, err)
		}
		skills = append(skills, Skill{Name: name, Content: string(content)})
	}
	return skills, nil
}

// Get returns the skill with the given name, or an error if not found.
func (r *Registry) Get(name string) (*Skill, error) {
	content, err := fs.ReadFile(r.fsys, name+".md")
	if err != nil {
		return nil, fmt.Errorf("skill %q not found: %w", name, err)
	}
	return &Skill{Name: name, Content: string(content)}, nil
}
