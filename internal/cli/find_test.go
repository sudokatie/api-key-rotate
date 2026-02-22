package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sudokatie/api-key-rotate/internal/config"
	"github.com/sudokatie/api-key-rotate/internal/providers"
	"github.com/sudokatie/api-key-rotate/internal/scanner"
)

func TestFindLocalLocations(t *testing.T) {
	// Create test environment
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	err := os.WriteFile(envPath, []byte("API_KEY=test-secret\nOTHER=value"), 0600)
	require.NoError(t, err)

	// Set up config
	config.Cfg = &config.Config{
		ScanPaths:       []string{dir},
		ExcludePatterns: []string{"node_modules"},
		FilePatterns:    []string{".env", ".env.*"},
	}

	// Find the key
	locs, err := findLocalLocations("API_KEY")
	require.NoError(t, err)
	require.Len(t, locs, 1)

	assert.Equal(t, "local", locs[0].Type)
	assert.Equal(t, envPath, locs[0].Path)
	assert.Equal(t, "test-secret", locs[0].Value)
	assert.True(t, locs[0].Exists)
}

func TestFindLocalLocationsNotFound(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	os.WriteFile(envPath, []byte("OTHER=value"), 0600)

	config.Cfg = &config.Config{
		ScanPaths:       []string{dir},
		ExcludePatterns: []string{},
		FilePatterns:    []string{".env"},
	}

	locs, err := findLocalLocations("MISSING_KEY")
	require.NoError(t, err)
	assert.Empty(t, locs)
}

func TestFindLocalLocationsMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "project")
	os.MkdirAll(subdir, 0755)

	os.WriteFile(filepath.Join(dir, ".env"), []byte("API_KEY=root-key"), 0600)
	os.WriteFile(filepath.Join(subdir, ".env"), []byte("API_KEY=project-key"), 0600)

	config.Cfg = &config.Config{
		ScanPaths:       []string{dir},
		ExcludePatterns: []string{},
		FilePatterns:    []string{".env"},
	}

	locs, err := findLocalLocations("API_KEY")
	require.NoError(t, err)
	assert.Len(t, locs, 2)
}

func TestFindLocalLocationsExcludesNodeModules(t *testing.T) {
	dir := t.TempDir()
	nodeModules := filepath.Join(dir, "node_modules", "pkg")
	os.MkdirAll(nodeModules, 0755)

	os.WriteFile(filepath.Join(dir, ".env"), []byte("API_KEY=root-key"), 0600)
	os.WriteFile(filepath.Join(nodeModules, ".env"), []byte("API_KEY=module-key"), 0600)

	config.Cfg = &config.Config{
		ScanPaths:       []string{dir},
		ExcludePatterns: []string{"node_modules"},
		FilePatterns:    []string{".env"},
	}

	locs, err := findLocalLocations("API_KEY")
	require.NoError(t, err)
	assert.Len(t, locs, 1)
	assert.Equal(t, filepath.Join(dir, ".env"), locs[0].Path)
}

// Test that scanner.FindKey works correctly
func TestScannerFindKeyIntegration(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	os.WriteFile(envPath, []byte("KEY1=value1\nKEY2=value2\nKEY1=duplicate"), 0600)

	files, err := scanner.Scan([]string{dir}, nil, []string{".env"})
	require.NoError(t, err)

	locs, err := scanner.FindKey(files, "KEY1")
	require.NoError(t, err)
	// FindKey returns all occurrences
	assert.GreaterOrEqual(t, len(locs), 1)
}

// Mock provider for testing
type mockProvider struct {
	name      string
	locations []providers.Location
	findErr   error
}

func (m *mockProvider) Name() string                                          { return m.name }
func (m *mockProvider) Configure(creds providers.Credentials) error           { return nil }
func (m *mockProvider) Test() error                                           { return nil }
func (m *mockProvider) Find(keyName string) ([]providers.Location, error)     { return m.locations, m.findErr }
func (m *mockProvider) Update(loc providers.Location, newValue string) error  { return nil }
func (m *mockProvider) SupportsRollback() bool                                { return false }
func (m *mockProvider) Rollback(loc providers.Location, origValue string) error { return nil }

func TestFindCloudLocationsNoProviders(t *testing.T) {
	// With no providers configured, should return empty
	config.Cfg = &config.Config{
		Providers: nil,
	}

	locs := findCloudLocations("API_KEY")
	assert.Empty(t, locs)
}
