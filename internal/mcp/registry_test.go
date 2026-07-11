package mcp

import "testing"

func TestList(t *testing.T) {
	r := NewRegistry()
	servers := r.List()
	if len(servers) != 5 {
		t.Errorf("List: got %d servers, want 5", len(servers))
	}
}

func TestGet_Known(t *testing.T) {
	r := NewRegistry()
	for _, name := range []string{"bitbucket", "jira", "jenkins", "confluence", "postgres"} {
		s, err := r.Get(name)
		if err != nil {
			t.Errorf("Get(%q): unexpected error: %v", name, err)
			continue
		}
		if s.Name != name {
			t.Errorf("Get(%q): got name %q", name, s.Name)
		}
		if len(s.Tools) == 0 {
			t.Errorf("Get(%q): no tools", name)
		}
	}
}

func TestGet_Unknown(t *testing.T) {
	r := NewRegistry()
	_, err := r.Get("nonexistent")
	if err == nil {
		t.Error("expected error for unknown server")
	}
}
