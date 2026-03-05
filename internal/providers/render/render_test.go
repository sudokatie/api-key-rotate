package render

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sudokatie/api-key-rotate/internal/providers"
)

func TestProviderName(t *testing.T) {
	p := New()
	if p.Name() != "render" {
		t.Errorf("expected name 'render', got '%s'", p.Name())
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
	err := p.Update(providers.Location{Path: "svc/key"}, "newvalue")
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

func TestListServices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]ListServicesResponse{
			{Service: Service{ID: "srv-1", Name: "my-api", Type: "web_service"}},
			{Service: Service{ID: "srv-2", Name: "worker", Type: "background_worker"}},
		})
	}))
	defer server.Close()

	c := NewClient("test-token")
	c.base = server.URL

	services, err := c.ListServices()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(services) != 2 {
		t.Errorf("expected 2 services, got %d", len(services))
	}

	if services[0].Name != "my-api" {
		t.Errorf("expected 'my-api', got '%s'", services[0].Name)
	}
}

func TestGetEnvVars(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]EnvVar{
			{"envVar": {Key: "API_KEY", Value: "secret123"}},
			{"envVar": {Key: "DATABASE_URL", Value: "postgres://..."}},
		})
	}))
	defer server.Close()

	c := NewClient("test-token")
	c.base = server.URL

	envVars, err := c.GetEnvVars("srv-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(envVars) != 2 {
		t.Errorf("expected 2 env vars, got %d", len(envVars))
	}

	if envVars[0].Key != "API_KEY" || envVars[0].Value != "secret123" {
		t.Errorf("unexpected env var: %+v", envVars[0])
	}
}

func TestUpdateEnvVar(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["value"] != "newvalue" {
			t.Errorf("unexpected value: %s", body["value"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClient("test-token")
	c.base = server.URL

	err := c.UpdateEnvVar("srv-1", "TEST_KEY", "newvalue")
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
		w.Write([]byte(`{"message": "Invalid API key"}`))
	}))
	defer server.Close()

	c := NewClient("bad-token")
	c.base = server.URL

	_, err := c.ListServices()
	if err == nil {
		t.Error("expected error for unauthorized response")
	}
}

func TestFindAcrossServices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/services":
			json.NewEncoder(w).Encode([]ListServicesResponse{
				{Service: Service{ID: "srv-1", Name: "api", Type: "web_service"}},
				{Service: Service{ID: "srv-2", Name: "worker", Type: "background_worker"}},
			})
		case "/services/srv-1/env-vars":
			json.NewEncoder(w).Encode([]map[string]EnvVar{
				{"envVar": {Key: "API_KEY", Value: "secret-srv1"}},
			})
		case "/services/srv-2/env-vars":
			json.NewEncoder(w).Encode([]map[string]EnvVar{
				{"envVar": {Key: "OTHER_KEY", Value: "other"}},
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

	if locs[0].Project != "api" || locs[0].Value != "secret-srv1" {
		t.Errorf("unexpected location: %+v", locs[0])
	}
}

func TestListServicesPagination(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// Return 100 results with cursor
			resp := make([]ListServicesResponse, 100)
			for i := 0; i < 100; i++ {
				resp[i] = ListServicesResponse{
					Service: Service{ID: "srv-" + string(rune('a'+i)), Name: "service"},
					Cursor:  "cursor-next",
				}
			}
			json.NewEncoder(w).Encode(resp)
		} else {
			// Return fewer than 100 to end pagination
			json.NewEncoder(w).Encode([]ListServicesResponse{
				{Service: Service{ID: "srv-last", Name: "last"}},
			})
		}
	}))
	defer server.Close()

	c := NewClient("test-token")
	c.base = server.URL

	services, err := c.ListServices()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(services) != 101 {
		t.Errorf("expected 101 services, got %d", len(services))
	}

	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}
}
