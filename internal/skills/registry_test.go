package skills

import (
	"testing"
	"testing/fstest"
)

var testFS = fstest.MapFS{
	"check-release-readiness.md": {Data: []byte("# check-release-readiness\nCheck Jira tickets.")},
	"update-release-notes.md":    {Data: []byte("# update-release-notes\nUpdate Confluence page.")},
}

func TestList(t *testing.T) {
	r := NewRegistry(testFS)
	skills, err := r.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("List: got %d skills, want 2", len(skills))
	}
	found := map[string]bool{}
	for _, s := range skills {
		found[s.Name] = true
		if s.Content == "" {
			t.Errorf("skill %q has empty content", s.Name)
		}
	}
	if !found["check-release-readiness"] {
		t.Error("missing check-release-readiness")
	}
	if !found["update-release-notes"] {
		t.Error("missing update-release-notes")
	}
}

func TestGet(t *testing.T) {
	r := NewRegistry(testFS)
	s, err := r.Get("check-release-readiness")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if s.Name != "check-release-readiness" {
		t.Errorf("Name: got %q", s.Name)
	}
	if s.Content == "" {
		t.Error("Content is empty")
	}
}

func TestGet_Unknown(t *testing.T) {
	r := NewRegistry(testFS)
	_, err := r.Get("nonexistent")
	if err == nil {
		t.Error("expected error for unknown skill")
	}
}
