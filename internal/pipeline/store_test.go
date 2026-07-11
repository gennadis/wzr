package pipeline

import (
	"testing"
)

var testPipeline = &Pipeline{
	Name:    "test-pipe",
	Version: "1.0",
	Steps: []Step{
		{ID: "s1", Name: "Step one", Type: StepTypeSkill, Skill: "my-skill"},
	},
	ManualMinutes: 15,
}

func TestStore_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	if err := store.Save(testPipeline); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.Load("test-pipe")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Name != testPipeline.Name {
		t.Errorf("Name: got %q, want %q", got.Name, testPipeline.Name)
	}
	if len(got.Steps) != 1 {
		t.Fatalf("Steps: got %d, want 1", len(got.Steps))
	}
	if got.Steps[0].ID != "s1" {
		t.Errorf("Step ID: got %q", got.Steps[0].ID)
	}
	if got.ManualMinutes != 15 {
		t.Errorf("ManualMinutes: got %d", got.ManualMinutes)
	}
}

func TestStore_List(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	second := &Pipeline{
		Name:  "another-pipe",
		Steps: []Step{{ID: "s1", Type: StepTypeSkill}},
	}

	if err := store.Save(testPipeline); err != nil {
		t.Fatalf("Save testPipeline: %v", err)
	}
	if err := store.Save(second); err != nil {
		t.Fatalf("Save second: %v", err)
	}

	names, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(names) != 2 {
		t.Errorf("List: got %d names, want 2", len(names))
	}
	found := map[string]bool{}
	for _, n := range names {
		found[n] = true
	}
	if !found["test-pipe"] || !found["another-pipe"] {
		t.Errorf("List: missing expected names, got %v", names)
	}
}

func TestStore_LoadNotFound(t *testing.T) {
	store := NewStore(t.TempDir())
	_, err := store.Load("nonexistent")
	if err == nil {
		t.Error("expected error loading nonexistent pipeline")
	}
}

func TestStore_ListEmptyDir(t *testing.T) {
	store := NewStore(t.TempDir())
	names, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("expected empty list, got %v", names)
	}
}
