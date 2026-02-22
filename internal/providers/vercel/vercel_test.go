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

func TestProviderName(t *testing.T) {
	p := New()
	assert.Equal(t, "vercel", p.Name())
}

func TestConfigureWithoutToken(t *testing.T) {
	p := New()
	err := p.Configure(providers.Credentials{})
	assert.Error(t, err)
}

func TestConfigureWithToken(t *testing.T) {
	p := New()
	err := p.Configure(providers.Credentials{Token: "test-token"})
	assert.NoError(t, err)
	assert.NotNil(t, p.client)
}

func TestTestNotConfigured(t *testing.T) {
	p := New()
	err := p.Test()
	assert.Error(t, err)
}

func TestFindNotConfigured(t *testing.T) {
	p := New()
	_, err := p.Find("API_KEY")
	assert.Error(t, err)
}

func TestSupportsRollback(t *testing.T) {
	p := New()
	assert.True(t, p.SupportsRollback())
}

func TestClientListProjects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		json.NewEncoder(w).Encode(projectsResponse{
			Projects: []Project{
				{ID: "proj1", Name: "my-project"},
				{ID: "proj2", Name: "other-project"},
			},
		})
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.base = server.URL

	projects, err := client.ListProjects()
	require.NoError(t, err)
	assert.Len(t, projects, 2)
	assert.Equal(t, "proj1", projects[0].ID)
}

func TestClientGetEnvVars(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(envsResponse{
			Envs: []EnvVar{
				{ID: "env1", Key: "API_KEY", Value: "secret123"},
				{ID: "env2", Key: "DB_URL", Value: "postgres://"},
			},
		})
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.base = server.URL

	envs, err := client.GetEnvVars("proj1")
	require.NoError(t, err)
	assert.Len(t, envs, 2)
	assert.Equal(t, "API_KEY", envs[0].Key)
}

func TestClientUpdateEnvVar(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.base = server.URL

	err := client.UpdateEnvVar("proj1", "env1", "new-value")
	assert.NoError(t, err)
}

func TestProviderFind(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v9/projects" {
			json.NewEncoder(w).Encode(projectsResponse{
				Projects: []Project{{ID: "proj1", Name: "my-project"}},
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
	require.Len(t, locs, 1)
	assert.Equal(t, "vercel", locs[0].Type)
	assert.Equal(t, "my-project", locs[0].Project)
	assert.Equal(t, "secret", locs[0].Value)
}
