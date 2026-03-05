package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sudokatie/api-key-rotate/internal/config"
	"github.com/sudokatie/api-key-rotate/internal/providers"
	"github.com/sudokatie/api-key-rotate/internal/rotation"

	// Import providers to trigger registration
	_ "github.com/sudokatie/api-key-rotate/internal/providers/github"
	_ "github.com/sudokatie/api-key-rotate/internal/providers/railway"
	_ "github.com/sudokatie/api-key-rotate/internal/providers/supabase"
	_ "github.com/sudokatie/api-key-rotate/internal/providers/vercel"
)

// Test runFind with local files
func TestRunFind_LocalFiles(t *testing.T) {
	// Create test environment
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	err := os.WriteFile(envPath, []byte("TEST_KEY=secret-value\nOTHER=value"), 0600)
	require.NoError(t, err)

	// Set up config
	config.Cfg = &config.Config{
		ScanPaths:       []string{tmpDir},
		ExcludePatterns: []string{"node_modules"},
		FilePatterns:    []string{".env"},
	}

	// Set flags
	findLocalOnly = true
	findCloudOnly = false
	findFormat = "text"
	noColor = true

	// Capture output - runFind calls os.Exit on not found, so we test the underlying functions
	locs, err := findLocalLocations("TEST_KEY")
	require.NoError(t, err)
	require.Len(t, locs, 1)
	assert.Equal(t, "secret-value", locs[0].Value)

	// Reset flags
	findLocalOnly = false
	noColor = false
}

// Test runFind with no results (tests the not found path)
func TestRunFind_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	os.WriteFile(envPath, []byte("OTHER_KEY=value"), 0600)

	config.Cfg = &config.Config{
		ScanPaths:       []string{tmpDir},
		ExcludePatterns: []string{},
		FilePatterns:    []string{".env"},
	}

	findLocalOnly = true
	findCloudOnly = false

	locs, err := findLocalLocations("NONEXISTENT_KEY")
	require.NoError(t, err)
	assert.Empty(t, locs)

	findLocalOnly = false
}

// Test config commands with temp directory
func TestRunConfigInit_CreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "new-config.yaml")

	// Call the underlying functions instead of runConfigInit
	// (which uses DefaultPath)
	cfg := &config.Config{
		ScanPaths:       []string{"~/projects"},
		ExcludePatterns: []string{"node_modules"},
		FilePatterns:    []string{".env"},
	}

	err := config.SaveTo(cfg, configPath)
	require.NoError(t, err)

	// Verify
	_, err = os.Stat(configPath)
	assert.NoError(t, err)
}

// Test config set functionality
func TestRunConfigSet_ModifiesConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	config.Cfg = &config.Config{
		ScanPaths: []string{"/test"},
		UI:        config.UIConfig{Color: true, Verbose: false},
	}

	// Save initial config
	err := config.SaveTo(config.Cfg, configPath)
	require.NoError(t, err)

	// Modify using setConfigValue
	err = setConfigValue(config.Cfg, "ui.verbose", "true")
	require.NoError(t, err)

	assert.True(t, config.Cfg.UI.Verbose)

	// Save again
	err = config.SaveTo(config.Cfg, configPath)
	require.NoError(t, err)

	// Reload and verify
	loaded, err := config.Load(configPath)
	require.NoError(t, err)
	assert.True(t, loaded.UI.Verbose)
}

// Test scan paths operations
func TestScanPathsOperations(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	config.Cfg = &config.Config{
		ScanPaths: []string{"/path/one"},
	}

	// Add a path
	config.Cfg.ScanPaths = append(config.Cfg.ScanPaths, "/path/two")
	err := config.SaveTo(config.Cfg, configPath)
	require.NoError(t, err)

	assert.Contains(t, config.Cfg.ScanPaths, "/path/two")

	// Remove a path
	newPaths := []string{}
	for _, p := range config.Cfg.ScanPaths {
		if p != "/path/one" {
			newPaths = append(newPaths, p)
		}
	}
	config.Cfg.ScanPaths = newPaths
	err = config.SaveTo(config.Cfg, configPath)
	require.NoError(t, err)

	assert.NotContains(t, config.Cfg.ScanPaths, "/path/one")
	assert.Contains(t, config.Cfg.ScanPaths, "/path/two")
}

// Test providers list with no providers registered
func TestProvidersListEmpty(t *testing.T) {
	config.Cfg = &config.Config{}
	noColor = true

	// Test getProviderStatus
	status := getProviderStatus("vercel")
	assert.Contains(t, status, "not configured")

	noColor = false
}

// Test history query options
func TestHistoryQueryOptions(t *testing.T) {
	// Test date parsing with various formats
	_, err := parseDate("2024-01-15")
	assert.NoError(t, err)

	_, err = parseDate("2024-01-15T10:30:00Z")
	assert.NoError(t, err)

	_, err = parseDate("invalid-date")
	assert.Error(t, err)
}

// Test gatherLocations with all flags
func TestGatherLocationsAllFlags(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	os.WriteFile(envPath, []byte("KEY=value"), 0600)

	config.Cfg = &config.Config{
		ScanPaths:    []string{tmpDir},
		FilePatterns: []string{".env"},
	}

	// Test with both skips - should return empty
	rotateSkipLocal = true
	rotateSkipCloud = true

	locs, err := gatherLocations("KEY")
	assert.NoError(t, err)
	assert.Empty(t, locs)

	rotateSkipLocal = false
	rotateSkipCloud = false
}

// Test location filtering edge cases
func TestFilterLocationsEdgeCases(t *testing.T) {
	// Empty input
	result := filterLocations(nil)
	assert.Nil(t, result)

	// Empty slice
	result = filterLocations([]providers.Location{})
	assert.Empty(t, result)
}

// Test printTransactionSummary with various states
func TestPrintTransactionSummaryStates(t *testing.T) {
	// All success
	tx := &rotation.Transaction{
		Locations: []rotation.LocationState{
			{Location: providers.Location{Path: "/a"}, Status: rotation.StatusSuccess},
			{Location: providers.Location{Path: "/b"}, Status: rotation.StatusSuccess},
		},
	}

	noColor = true
	stdout, _ := captureOutput(func() {
		printTransactionSummary(tx)
	})
	assert.Contains(t, stdout, "2 succeeded")
	assert.Contains(t, stdout, "0 failed")

	// Mixed results with verbose
	tx.Locations[1].Status = rotation.StatusFailed
	tx.Locations[1].Error = assert.AnError
	verbose = true

	stdout, _ = captureOutput(func() {
		printTransactionSummary(tx)
	})
	assert.Contains(t, stdout, "1 succeeded")
	assert.Contains(t, stdout, "1 failed")

	verbose = false
	noColor = false
}
