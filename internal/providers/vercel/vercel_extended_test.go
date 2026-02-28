package vercel

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sudokatie/api-key-rotate/internal/providers"
)

func TestProviderUpdate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	p := New()
	p.client = NewClient("test-token")
	p.client.base = server.URL

	loc := providers.Location{
		Path: "proj1/env1",
	}

	err := p.Update(loc, "new-value")
	assert.NoError(t, err)
}

func TestProviderUpdateInvalidPath(t *testing.T) {
	p := New()
	p.client = NewClient("test-token")

	loc := providers.Location{
		Path: "invalid-no-slash",
	}

	err := p.Update(loc, "value")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid location path")
}

func TestProviderUpdateNotConfigured(t *testing.T) {
	p := New()

	err := p.Update(providers.Location{}, "value")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestProviderRollback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	p := New()
	p.client = NewClient("test-token")
	p.client.base = server.URL

	loc := providers.Location{
		Path: "proj1/env1",
	}

	err := p.Rollback(loc, "original-value")
	assert.NoError(t, err)
}

func TestProviderTestSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(projectsResponse{
			Projects: []Project{},
		})
	}))
	defer server.Close()

	p := New()
	p.client = NewClient("test-token")
	p.client.base = server.URL

	err := p.Test()
	assert.NoError(t, err)
}

func TestProviderTestFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	p := New()
	p.client = NewClient("test-token")
	p.client.base = server.URL

	err := p.Test()
	assert.Error(t, err)
}

func TestProviderFindNoMatches(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v9/projects" {
			json.NewEncoder(w).Encode(projectsResponse{
				Projects: []Project{{ID: "proj1", Name: "my-project"}},
			})
		} else {
			json.NewEncoder(w).Encode(envsResponse{
				Envs: []EnvVar{
					{ID: "env1", Key: "OTHER_KEY", Value: "value"},
				},
			})
		}
	}))
	defer server.Close()

	p := New()
	p.client = NewClient("test-token")
	p.client.base = server.URL

	locs, err := p.Find("MISSING_KEY")
	require.NoError(t, err)
	assert.Empty(t, locs)
}

func TestProviderFindMultipleProjects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v9/projects" {
			json.NewEncoder(w).Encode(projectsResponse{
				Projects: []Project{
					{ID: "proj1", Name: "project-one"},
					{ID: "proj2", Name: "project-two"},
				},
			})
		} else {
			json.NewEncoder(w).Encode(envsResponse{
				Envs: []EnvVar{
					{ID: "env1", Key: "API_KEY", Value: "secret", Target: []string{"production"}},
				},
			})
		}
	}))
	defer server.Close()

	p := New()
	p.client = NewClient("test-token")
	p.client.base = server.URL

	locs, err := p.Find("API_KEY")
	require.NoError(t, err)
	assert.Len(t, locs, 2) // Found in both projects
}

func TestClientListProjectsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.base = server.URL

	_, err := client.ListProjects()
	assert.Error(t, err)
}

func TestClientGetEnvVarsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.base = server.URL

	_, err := client.GetEnvVars("proj1")
	assert.Error(t, err)
}

func TestClientUpdateEnvVarError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.base = server.URL

	err := client.UpdateEnvVar("proj1", "env1", "value")
	assert.Error(t, err)
}

func TestProviderFindSkipsInaccessibleProjects(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v9/projects" {
			json.NewEncoder(w).Encode(projectsResponse{
				Projects: []Project{
					{ID: "proj1", Name: "accessible"},
					{ID: "proj2", Name: "forbidden"},
				},
			})
		} else {
			callCount++
			if callCount == 1 {
				// First project - return envs
				json.NewEncoder(w).Encode(envsResponse{
					Envs: []EnvVar{{ID: "env1", Key: "API_KEY", Value: "secret"}},
				})
			} else {
				// Second project - forbidden
				w.WriteHeader(http.StatusForbidden)
			}
		}
	}))
	defer server.Close()

	p := New()
	p.client = NewClient("test-token")
	p.client.base = server.URL

	locs, err := p.Find("API_KEY")
	require.NoError(t, err)
	assert.Len(t, locs, 1) // Only found in first project
}
