package supabase

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sudokatie/api-key-rotate/internal/providers"
)

func TestProviderName(t *testing.T) {
	p := New()
	if p.Name() != "supabase" {
		t.Errorf("expected name 'supabase', got '%s'", p.Name())
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
	err := p.Update(providers.Location{Path: "proj/key"}, "newvalue")
	if err == nil {
		t.Error("expected error when not configured")
	}
}

func TestUpdateInvalidPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
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
		json.NewEncoder(w).Encode([]Project{
			{ID: "proj1", Name: "My Project", Region: "us-east-1", Status: "ACTIVE_HEALTHY"},
			{ID: "proj2", Name: "Another Project", Region: "eu-west-1", Status: "ACTIVE_HEALTHY"},
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

	if projects[0].Name != "My Project" {
		t.Errorf("expected 'My Project', got '%s'", projects[0].Name)
	}
}

func TestListSecrets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]Secret{
			{Name: "API_KEY", Value: "secret123"},
			{Name: "DATABASE_URL", Value: "postgres://..."},
		})
	}))
	defer server.Close()

	c := NewClient("test-token")
	c.base = server.URL

	secrets, err := c.ListSecrets("proj1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(secrets) != 2 {
		t.Errorf("expected 2 secrets, got %d", len(secrets))
	}

	if secrets[0].Name != "API_KEY" || secrets[0].Value != "secret123" {
		t.Errorf("unexpected secret: %+v", secrets[0])
	}
}

func TestUpsertSecrets(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var secrets []Secret
		json.NewDecoder(r.Body).Decode(&secrets)
		if len(secrets) != 1 || secrets[0].Name != "TEST_KEY" {
			t.Errorf("unexpected request body: %+v", secrets)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClient("test-token")
	c.base = server.URL

	err := c.UpsertSecrets("proj1", []Secret{{Name: "TEST_KEY", Value: "newvalue"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Error("expected API call")
	}
}

func TestDeleteSecret(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClient("test-token")
	c.base = server.URL

	err := c.DeleteSecret("proj1", "OLD_KEY")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Error("expected API call")
	}
}

func TestAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message": "Invalid token"}`))
	}))
	defer server.Close()

	c := NewClient("bad-token")
	c.base = server.URL

	_, err := c.ListProjects()
	if err == nil {
		t.Error("expected error for unauthorized response")
	}
}

func TestFindAcrossProjects(t *testing.T) {
	requestNum := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/projects":
			json.NewEncoder(w).Encode([]Project{
				{ID: "proj1", Name: "Project 1", Region: "us-east-1", Status: "ACTIVE_HEALTHY"},
				{ID: "proj2", Name: "Project 2", Region: "eu-west-1", Status: "ACTIVE_HEALTHY"},
			})
		case "/v1/projects/proj1/secrets":
			json.NewEncoder(w).Encode([]Secret{
				{Name: "API_KEY", Value: "secret-proj1"},
			})
		case "/v1/projects/proj2/secrets":
			json.NewEncoder(w).Encode([]Secret{
				{Name: "OTHER_KEY", Value: "other"},
			})
		}
		requestNum++
	}))
	defer server.Close()

	p := New()
	p.Configure(providers.Credentials{Token: "test"})
	p.client.base = server.URL

	locs, err := p.Find("API_KEY")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(locs) != 1 {
		t.Errorf("expected 1 location, got %d", len(locs))
	}

	if locs[0].Project != "Project 1" || locs[0].Value != "secret-proj1" {
		t.Errorf("unexpected location: %+v", locs[0])
	}
}

func TestFindSkipsPausedProjects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/projects":
			json.NewEncoder(w).Encode([]Project{
				{ID: "proj1", Name: "Active", Region: "us-east-1", Status: "ACTIVE_HEALTHY"},
				{ID: "proj2", Name: "Paused", Region: "eu-west-1", Status: "INACTIVE"},
			})
		case "/v1/projects/proj1/secrets":
			json.NewEncoder(w).Encode([]Secret{
				{Name: "API_KEY", Value: "secret"},
			})
		case "/v1/projects/proj2/secrets":
			// Should not be called for paused project
			t.Error("should not query paused project secrets")
		}
	}))
	defer server.Close()

	p := New()
	p.Configure(providers.Credentials{Token: "test"})
	p.client.base = server.URL

	_, err := p.Find("API_KEY")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
