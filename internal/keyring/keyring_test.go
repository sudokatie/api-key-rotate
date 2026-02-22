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
