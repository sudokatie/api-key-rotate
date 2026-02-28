package keyring

import (
	"crypto/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	rand.Read(b)
	for i := range b {
		b[i] = letters[int(b[i])%len(letters)]
	}
	return string(b)
}

func TestKeyring(t *testing.T) {
	// These tests may fail in CI without keyring access
	if os.Getenv("CI") != "" {
		t.Skip("skipping keyring test in CI")
	}

	testKey := "test-provider-" + randomString(8)

	// Set
	err := Set(testKey, "test-token")
	require.NoError(t, err)

	// Get
	token, err := Get(testKey)
	require.NoError(t, err)
	assert.Equal(t, "test-token", token)

	// Exists
	assert.True(t, Exists(testKey))

	// Delete
	err = Delete(testKey)
	require.NoError(t, err)

	// Verify deleted
	assert.False(t, Exists(testKey))
}

func TestGetNonexistent(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("skipping keyring test in CI")
	}

	_, err := Get("nonexistent-provider-" + randomString(8))
	assert.Error(t, err)
}

func TestExistsNonexistent(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("skipping keyring test in CI")
	}

	exists := Exists("nonexistent-provider-" + randomString(8))
	assert.False(t, exists)
}

func TestGetFromEnvVar_Vercel(t *testing.T) {
	// Save and restore original
	orig := os.Getenv("VERCEL_TOKEN")
	defer os.Setenv("VERCEL_TOKEN", orig)

	// Set env var
	os.Setenv("VERCEL_TOKEN", "test-vercel-token-from-env")

	// Get should return env var value
	token, err := Get("vercel")
	require.NoError(t, err)
	assert.Equal(t, "test-vercel-token-from-env", token)

	// Exists should return true
	assert.True(t, Exists("vercel"))

	// Source should return "env"
	assert.Equal(t, "env", Source("vercel"))
}

func TestGetFromEnvVar_GitHub(t *testing.T) {
	// Save and restore original
	orig := os.Getenv("GITHUB_TOKEN")
	defer os.Setenv("GITHUB_TOKEN", orig)

	// Set env var
	os.Setenv("GITHUB_TOKEN", "test-github-token-from-env")

	// Get should return env var value
	token, err := Get("github")
	require.NoError(t, err)
	assert.Equal(t, "test-github-token-from-env", token)

	// Exists should return true
	assert.True(t, Exists("github"))

	// Source should return "env"
	assert.Equal(t, "env", Source("github"))
}

func TestGetFromEnvVar_CaseInsensitive(t *testing.T) {
	// Save and restore original
	orig := os.Getenv("VERCEL_TOKEN")
	defer os.Setenv("VERCEL_TOKEN", orig)

	// Set env var
	os.Setenv("VERCEL_TOKEN", "test-token")

	// Get should work with different cases
	token, err := Get("Vercel")
	require.NoError(t, err)
	assert.Equal(t, "test-token", token)

	token, err = Get("VERCEL")
	require.NoError(t, err)
	assert.Equal(t, "test-token", token)
}

func TestGetEnvVarPriority(t *testing.T) {
	// This tests that env var takes priority over keyring
	// We can only test this if keyring is available
	if os.Getenv("CI") != "" {
		t.Skip("skipping keyring test in CI")
	}

	testProvider := "vercel"
	orig := os.Getenv("VERCEL_TOKEN")
	defer os.Setenv("VERCEL_TOKEN", orig)

	// Set a value in keyring
	err := Set(testProvider, "keyring-token")
	require.NoError(t, err)
	defer Delete(testProvider)

	// Without env var, should get keyring value
	os.Setenv("VERCEL_TOKEN", "")
	token, err := Get(testProvider)
	require.NoError(t, err)
	assert.Equal(t, "keyring-token", token)
	assert.Equal(t, "keyring", Source(testProvider))

	// With env var set, should get env value
	os.Setenv("VERCEL_TOKEN", "env-token")
	token, err = Get(testProvider)
	require.NoError(t, err)
	assert.Equal(t, "env-token", token)
	assert.Equal(t, "env", Source(testProvider))
}

func TestSourceEmpty(t *testing.T) {
	// For unknown provider with no keyring entry
	source := Source("unknown-provider-" + randomString(8))
	assert.Equal(t, "", source)
}
