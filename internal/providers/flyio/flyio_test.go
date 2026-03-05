package flyio

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sudokatie/api-key-rotate/internal/providers"
)

func TestProviderName(t *testing.T) {
	p := New()
	if p.Name() != "flyio" {
		t.Errorf("expected name 'flyio', got '%s'", p.Name())
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
	if p.SupportsRollback() {
		t.Error("expected no rollback support (Fly.io doesn't expose values)")
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
	err := p.Update(providers.Location{Path: "app/key"}, "newvalue")
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

func TestListApps(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"apps": []App{
				{ID: "app1", Name: "my-api", Status: "running", Organization: struct {
					Slug string `json:"slug"`
				}{Slug: "personal"}},
				{ID: "app2", Name: "worker", Status: "running", Organization: struct {
					Slug string `json:"slug"`
				}{Slug: "personal"}},
			},
		})
	}))
	defer server.Close()

	c := NewClient("test-token")
	c.base = server.URL

	apps, err := c.ListApps()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(apps) != 2 {
		t.Errorf("expected 2 apps, got %d", len(apps))
	}

	if apps[0].Name != "my-api" {
		t.Errorf("expected 'my-api', got '%s'", apps[0].Name)
	}
}

func TestListSecrets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]Secret{
			{Name: "API_KEY", Digest: "sha256:abc123"},
			{Name: "DATABASE_URL", Digest: "sha256:def456"},
		})
	}))
	defer server.Close()

	c := NewClient("test-token")
	c.base = server.URL

	secrets, err := c.ListSecrets("my-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(secrets) != 2 {
		t.Errorf("expected 2 secrets, got %d", len(secrets))
	}

	if secrets[0].Name != "API_KEY" {
		t.Errorf("expected 'API_KEY', got '%s'", secrets[0].Name)
	}
}

func TestSetSecrets(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["TEST_KEY"] != "newvalue" {
			t.Errorf("unexpected value: %v", body)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClient("test-token")
	c.base = server.URL

	err := c.SetSecrets("my-app", map[string]string{"TEST_KEY": "newvalue"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Error("expected API call")
	}
}

func TestUnsetSecrets(t *testing.T) {
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

	err := c.UnsetSecrets("my-app", []string{"OLD_KEY"})
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
		w.Write([]byte(`{"error": "Invalid token"}`))
	}))
	defer server.Close()

	c := NewClient("bad-token")
	c.base = server.URL

	_, err := c.ListApps()
	if err == nil {
		t.Error("expected error for unauthorized response")
	}
}

func TestFindAcrossApps(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/apps":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"apps": []App{
					{ID: "1", Name: "api", Status: "running", Organization: struct {
						Slug string `json:"slug"`
					}{Slug: "myorg"}},
					{ID: "2", Name: "worker", Status: "running", Organization: struct {
						Slug string `json:"slug"`
					}{Slug: "myorg"}},
				},
			})
		case "/apps/api/secrets":
			json.NewEncoder(w).Encode([]Secret{
				{Name: "API_KEY", Digest: "abc"},
			})
		case "/apps/worker/secrets":
			json.NewEncoder(w).Encode([]Secret{
				{Name: "OTHER_KEY", Digest: "def"},
			})
		}
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

	if locs[0].Project != "api" {
		t.Errorf("unexpected project: %s", locs[0].Project)
	}

	// Fly.io doesn't expose values
	if locs[0].Value != "" {
		t.Errorf("expected empty value for Fly.io, got '%s'", locs[0].Value)
	}
}

func TestFindSkipsSuspendedApps(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/apps":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"apps": []App{
					{ID: "1", Name: "active", Status: "running"},
					{ID: "2", Name: "paused", Status: "suspended"},
				},
			})
		case "/apps/active/secrets":
			json.NewEncoder(w).Encode([]Secret{
				{Name: "API_KEY", Digest: "abc"},
			})
		case "/apps/paused/secrets":
			// Should not be called
			t.Error("should not query suspended app secrets")
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

func TestRollbackWithoutValue(t *testing.T) {
	p := New()
	p.Configure(providers.Credentials{Token: "test"})

	err := p.Rollback(providers.Location{Path: "app/key"}, "")
	if err == nil {
		t.Error("expected error when original value is empty")
	}
}

func TestRollbackWithValue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	p := New()
	p.Configure(providers.Credentials{Token: "test"})
	p.client.base = server.URL

	err := p.Rollback(providers.Location{Path: "app/key"}, "original-value")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
