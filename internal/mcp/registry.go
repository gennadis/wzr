package mcp

import "fmt"

// Tool is a single callable tool exposed by an MCP server.
type Tool struct {
	Name        string
	Description string
}

// Server represents a registered MCP server and its tools.
type Server struct {
	Name        string
	Description string
	Tools       []Tool
}

// Registry holds the hardcoded set of MCP servers available to pipelines.
type Registry struct {
	servers map[string]Server
}

// NewRegistry returns a Registry pre-loaded with the Sber infra MCP servers.
func NewRegistry() *Registry {
	servers := []Server{
		{
			Name:        "bitbucket",
			Description: "Bitbucket source control — pull requests and repository operations",
			Tools: []Tool{
				{Name: "list_pull_requests", Description: "List pull requests for a repository, optionally filtered by state"},
				{Name: "get_pr", Description: "Get details and diff for a single pull request"},
				{Name: "merge_pr", Description: "Merge a pull request"},
			},
		},
		{
			Name:        "jira",
			Description: "Jira issue tracker — search, update, and transition issues",
			Tools: []Tool{
				{Name: "search_issues", Description: "Search issues using JQL"},
				{Name: "transition_issue", Description: "Move an issue to a new status"},
				{Name: "get_issue", Description: "Get full details of a single issue"},
			},
		},
		{
			Name:        "jenkins",
			Description: "Jenkins CI — trigger builds and query build status",
			Tools: []Tool{
				{Name: "trigger_build", Description: "Trigger a Jenkins job build with optional parameters"},
				{Name: "get_build_status", Description: "Get the status and logs of a build"},
				{Name: "list_jobs", Description: "List available Jenkins jobs"},
			},
		},
		{
			Name:        "confluence",
			Description: "Confluence wiki — create and update pages",
			Tools: []Tool{
				{Name: "get_page", Description: "Get a Confluence page by title or ID"},
				{Name: "create_page", Description: "Create a new Confluence page in a space"},
				{Name: "update_page", Description: "Append or replace content on an existing page"},
			},
		},
		{
			Name:        "postgres",
			Description: "PostgreSQL — run read-only queries against the application database",
			Tools: []Tool{
				{Name: "query", Description: "Execute a SELECT query and return results as JSON"},
				{Name: "list_tables", Description: "List tables in a schema"},
			},
		},
	}

	m := make(map[string]Server, len(servers))
	for _, s := range servers {
		m[s.Name] = s
	}
	return &Registry{servers: m}
}

// List returns all registered MCP servers.
func (r *Registry) List() []Server {
	out := make([]Server, 0, len(r.servers))
	for _, s := range r.servers {
		out = append(out, s)
	}
	return out
}

// Get returns the server with the given name, or an error if unknown.
func (r *Registry) Get(name string) (*Server, error) {
	s, ok := r.servers[name]
	if !ok {
		return nil, fmt.Errorf("unknown MCP server %q", name)
	}
	return &s, nil
}
