package railway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sudokatie/api-key-rotate/internal/providers"
)

func TestProviderName(t *testing.T) {
	p := New()
	if p.Name() != "railway" {
		t.Errorf("expected name 'railway', got '%s'", p.Name())
	}
}

func TestConfigureRequiresToken(t *testing.T) {
	p := New()
	err := p.Configure(providers.Credentials{})
	if err == nil {
		t.Error("expected error when token is missing")
	}
}

func TestConfigureSuccess(t *testing.T) {
	p := New()
	err := p.Configure(providers.Credentials{Token: "test-token"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSupportsRollback(t *testing.T) {
	p := New()
	if !p.SupportsRollback() {
		t.Error("expected rollback support")
	}
}

func TestFindNotConfigured(t *testing.T) {
	p := New()
	_, err := p.Find("TEST_KEY")
	if err == nil {
		t.Error("expected error when not configured")
	}
}

func TestTestNotConfigured(t *testing.T) {
	p := New()
	err := p.Test()
	if err == nil {
		t.Error("expected error when not configured")
	}
}

func TestUpdateNotConfigured(t *testing.T) {
	p := New()
	err := p.Update(providers.Location{Path: "proj/env/key"}, "newvalue")
	if err == nil {
		t.Error("expected error when not configured")
	}
}

func TestUpdateInvalidPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{},
		})
	}))
	defer server.Close()

	p := New()
	p.Configure(providers.Credentials{Token: "test"})
	p.client.base = server.URL

	err := p.Update(providers.Location{Path: "invalid"}, "value")
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestListProjects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"me": map[string]interface{}{
					"projects": map[string]interface{}{
						"edges": []map[string]interface{}{
							{"node": map[string]interface{}{"id": "proj1", "name": "Project 1"}},
							{"node": map[string]interface{}{"id": "proj2", "name": "Project 2"}},
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	c := NewClient("test-token")
	c.base = server.URL

	projects, err := c.ListProjects()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
	}

	if projects[0].Name != "Project 1" {
		t.Errorf("expected 'Project 1', got '%s'", projects[0].Name)
	}
}

func TestListEnvironments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"project": map[string]interface{}{
					"environments": map[string]interface{}{
						"edges": []map[string]interface{}{
							{"node": map[string]interface{}{"id": "env1", "name": "production"}},
							{"node": map[string]interface{}{"id": "env2", "name": "staging"}},
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	c := NewClient("test-token")
	c.base = server.URL

	envs, err := c.ListEnvironments("proj1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(envs) != 2 {
		t.Errorf("expected 2 environments, got %d", len(envs))
	}
}

func TestGetVariables(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"variables": map[string]string{
					"API_KEY":     "secret123",
					"DATABASE_URL": "postgres://...",
				},
			},
		})
	}))
	defer server.Close()

	c := NewClient("test-token")
	c.base = server.URL

	vars, err := c.GetVariables("proj1", "env1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if vars["API_KEY"] != "secret123" {
		t.Errorf("expected 'secret123', got '%s'", vars["API_KEY"])
	}
}

func TestUpsertVariable(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"variableUpsert": true,
			},
		})
	}))
	defer server.Close()

	c := NewClient("test-token")
	c.base = server.URL

	err := c.UpsertVariable("proj1", "env1", "TEST_KEY", "newvalue")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Error("expected API call")
	}
}

func TestGraphQLError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"errors": []map[string]string{
				{"message": "Authentication failed"},
			},
		})
	}))
	defer server.Close()

	c := NewClient("bad-token")
	c.base = server.URL

	_, err := c.ListProjects()
	if err == nil {
		t.Error("expected error for GraphQL error response")
	}
}
