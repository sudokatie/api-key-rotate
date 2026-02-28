package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sudokatie/api-key-rotate/internal/providers"
)

// testClient creates a client with minimal retry for faster tests
func testClient(token string) *Client {
	cfg := providers.RetryConfig{
		MaxRetries:       0, // No retries for tests
		InitialBackoff:   10 * time.Millisecond,
		MaxBackoff:       50 * time.Millisecond,
		BackoffFactor:    1.5,
		Jitter:           0,
		RetryOn5xx:       false,
		RetryOnRateLimit: false,
	}
	return &Client{
		token: token,
		http:  providers.NewRetryableClient(5*time.Second, cfg),
		base:  "https://api.github.com",
	}
}

func TestClientListOrgRepos(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			json.NewEncoder(w).Encode([]Repo{
				{ID: 1, Name: "org-repo1", FullName: "testorg/org-repo1"},
				{ID: 2, Name: "org-repo2", FullName: "testorg/org-repo2"},
			})
		} else {
			json.NewEncoder(w).Encode([]Repo{})
		}
	}))
	defer server.Close()

	client := testClient("test-token")
	client.base = server.URL

	repos, err := client.ListOrgRepos("testorg")
	require.NoError(t, err)
	assert.Len(t, repos, 2)
	assert.Equal(t, "testorg/org-repo1", repos[0].FullName)
}

func TestClientListOrgReposError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "Not found"})
	}))
	defer server.Close()

	client := testClient("test-token")
	client.base = server.URL

	_, err := client.ListOrgRepos("nonexistent")
	assert.Error(t, err)
}

func TestProviderUpdate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "public-key") {
			json.NewEncoder(w).Encode(PublicKey{
				KeyID: "key-123",
				Key:   "MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDE=",
			})
		} else if r.Method == "PUT" {
			w.WriteHeader(http.StatusCreated)
		}
	}))
	defer server.Close()

	p := New()
	p.client = testClient("test-token")
	p.client.base = server.URL

	loc := providers.Location{
		Path: "owner/repo/SECRET_KEY",
	}

	err := p.Update(loc, "new-secret-value")
	assert.NoError(t, err)
}

func TestProviderUpdateInvalidPath(t *testing.T) {
	p := New()
	p.client = testClient("test-token")

	loc := providers.Location{
		Path: "invalid-path",
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

func TestProviderUpdateGetPublicKeyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	p := New()
	p.client = testClient("test-token")
	p.client.base = server.URL

	loc := providers.Location{
		Path: "owner/repo/SECRET",
	}

	err := p.Update(loc, "value")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get public key")
}

func TestProviderFindWithOrgs(t *testing.T) {
	userReposCalled := false
	orgReposCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle pagination - return empty on page > 1
		if strings.Contains(r.URL.RawQuery, "page=2") {
			json.NewEncoder(w).Encode([]Repo{})
			return
		}

		if strings.HasPrefix(r.URL.Path, "/user/repos") {
			userReposCalled = true
			json.NewEncoder(w).Encode([]Repo{})
		} else if strings.Contains(r.URL.Path, "/orgs/myorg/repos") {
			orgReposCalled = true
			json.NewEncoder(w).Encode([]Repo{
				{ID: 1, Name: "org-repo", FullName: "myorg/org-repo", Owner: struct {
					Login string `json:"login"`
				}{Login: "myorg"}},
			})
		} else if strings.Contains(r.URL.Path, "/actions/secrets") {
			json.NewEncoder(w).Encode(secretsResponse{
				Secrets: []Secret{{Name: "ORG_KEY"}},
			})
		} else {
			json.NewEncoder(w).Encode([]Repo{})
		}
	}))
	defer server.Close()

	p := New()
	p.client = testClient("test-token")
	p.client.base = server.URL
	p.orgs = []string{"myorg"}

	locs, err := p.Find("ORG_KEY")
	require.NoError(t, err)
	require.Len(t, locs, 1)
	assert.Equal(t, "myorg/org-repo", locs[0].Project)
	assert.True(t, userReposCalled)
	assert.True(t, orgReposCalled)
}

func TestProviderFindNoMatches(t *testing.T) {
	userCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle pagination
		if strings.Contains(r.URL.RawQuery, "page=2") {
			json.NewEncoder(w).Encode([]Repo{})
			return
		}

		if strings.HasPrefix(r.URL.Path, "/user/repos") {
			userCalled = true
			json.NewEncoder(w).Encode([]Repo{
				{ID: 1, Name: "repo1", FullName: "user/repo1", Owner: struct {
					Login string `json:"login"`
				}{Login: "user"}},
			})
		} else if strings.Contains(r.URL.Path, "/actions/secrets") {
			json.NewEncoder(w).Encode(secretsResponse{
				Secrets: []Secret{{Name: "OTHER_KEY"}},
			})
		} else {
			json.NewEncoder(w).Encode([]Repo{})
		}
	}))
	defer server.Close()

	p := New()
	p.client = testClient("test-token")
	p.client.base = server.URL

	locs, err := p.Find("MISSING_KEY")
	require.NoError(t, err)
	assert.Empty(t, locs)
	assert.True(t, userCalled)
}

func TestProviderTestSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]Repo{})
	}))
	defer server.Close()

	p := New()
	p.client = testClient("test-token")
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
	p.client = testClient("test-token")
	p.client.base = server.URL

	err := p.Test()
	assert.Error(t, err)
}

func TestClientListSecretsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client := testClient("test-token")
	client.base = server.URL

	_, err := client.ListSecrets("owner", "repo")
	assert.Error(t, err)
}

func TestClientGetPublicKeyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client := testClient("test-token")
	client.base = server.URL

	_, err := client.GetPublicKey("owner", "repo")
	assert.Error(t, err)
}

func TestClientUpdateSecretError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client := testClient("test-token")
	client.base = server.URL

	err := client.UpdateSecret("owner", "repo", "SECRET", "encrypted", "key-123")
	assert.Error(t, err)
}

func TestClientListUserReposError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := testClient("test-token")
	client.base = server.URL

	_, err := client.ListUserRepos()
	assert.Error(t, err)
}
