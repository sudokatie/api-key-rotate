package github

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
	assert.Equal(t, "github", p.Name())
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

func TestConfigureWithOrgs(t *testing.T) {
	p := New()
	err := p.Configure(providers.Credentials{
		Token: "test-token",
		ExtraData: map[string]string{
			"orgs": "org1,org2",
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, []string{"org1", "org2"}, p.orgs)
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
	assert.False(t, p.SupportsRollback())
}

func TestRollbackReturnsError(t *testing.T) {
	p := New()
	err := p.Rollback(providers.Location{}, "value")
	assert.Error(t, err)
}

func TestClientListUserRepos(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.Header.Get("Authorization"), "Bearer")
		callCount++
		if callCount == 1 {
			json.NewEncoder(w).Encode([]Repo{
				{ID: 1, Name: "repo1", FullName: "user/repo1"},
				{ID: 2, Name: "repo2", FullName: "user/repo2"},
			})
		} else {
			// Return empty on second call to end pagination
			json.NewEncoder(w).Encode([]Repo{})
		}
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.base = server.URL

	repos, err := client.ListUserRepos()
	require.NoError(t, err)
	assert.Len(t, repos, 2)
}

func TestClientListSecrets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(secretsResponse{
			Secrets: []Secret{
				{Name: "API_KEY"},
				{Name: "DB_PASSWORD"},
			},
			TotalCount: 2,
		})
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.base = server.URL

	secrets, err := client.ListSecrets("owner", "repo")
	require.NoError(t, err)
	assert.Len(t, secrets, 2)
	assert.Equal(t, "API_KEY", secrets[0].Name)
}

func TestClientGetPublicKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(PublicKey{
			KeyID: "key-123",
			Key:   "BASE64ENCODEDKEY==",
		})
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.base = server.URL

	key, err := client.GetPublicKey("owner", "repo")
	require.NoError(t, err)
	assert.Equal(t, "key-123", key.KeyID)
}

func TestClientUpdateSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.base = server.URL

	err := client.UpdateSecret("owner", "repo", "SECRET", "encrypted", "key-123")
	assert.NoError(t, err)
}

func TestProviderFind(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user/repos" {
			callCount++
			if callCount == 1 {
				json.NewEncoder(w).Encode([]Repo{
					{ID: 1, Name: "repo1", FullName: "user/repo1", Owner: struct {
						Login string `json:"login"`
					}{Login: "user"}},
				})
			} else {
				json.NewEncoder(w).Encode([]Repo{})
			}
		} else {
			json.NewEncoder(w).Encode(secretsResponse{
				Secrets: []Secret{{Name: "API_KEY"}},
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
	assert.Equal(t, "github", locs[0].Type)
	assert.Equal(t, "user/repo1", locs[0].Project)
}

func TestEncryptSecret(t *testing.T) {
	// Use a valid 32-byte public key (base64 encoded)
	// This is just for testing - not a real key
	publicKey := "MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDE=" // 32 bytes base64

	encrypted, err := EncryptSecret(publicKey, "secret-value")
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)
}

func TestEncryptSecretInvalidKey(t *testing.T) {
	_, err := EncryptSecret("not-valid-base64!!!", "secret")
	assert.Error(t, err)
}

func TestEncryptSecretWrongKeyLength(t *testing.T) {
	// Too short
	_, err := EncryptSecret("dG9vc2hvcnQ=", "secret")
	assert.Error(t, err)
}
