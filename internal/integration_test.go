// +build integration

package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sudokatie/api-key-rotate/internal/config"
	"github.com/sudokatie/api-key-rotate/internal/providers"
	"github.com/sudokatie/api-key-rotate/internal/rotation"
	"github.com/sudokatie/api-key-rotate/internal/scanner"
)

// TestFullRotationFlow tests complete rotation across multiple local files
func TestFullRotationFlow(t *testing.T) {
	// Create temp directory with test .env files
	dir := t.TempDir()

	// Create project structure
	projectA := filepath.Join(dir, "project-a")
	projectB := filepath.Join(dir, "project-b")
	os.MkdirAll(projectA, 0755)
	os.MkdirAll(projectB, 0755)

	// Write .env files with same key
	envA := filepath.Join(projectA, ".env")
	envB := filepath.Join(projectB, ".env")
	os.WriteFile(envA, []byte("API_KEY=old-secret-123\nOTHER=keep"), 0600)
	os.WriteFile(envB, []byte("API_KEY=old-secret-123\nDB_URL=postgres://localhost"), 0600)

	// Scan for files
	files, err := scanner.Scan([]string{dir}, []string{}, []string{".env"})
	require.NoError(t, err)
	require.Len(t, files, 2)

	// Find key locations
	locals, err := scanner.FindKey(files, "API_KEY")
	require.NoError(t, err)
	require.Len(t, locals, 2)

	// Convert to provider locations
	var locations []providers.Location
	for _, loc := range locals {
		locations = append(locations, providers.Location{
			Type:   "local",
			Path:   loc.Path,
			Exists: true,
			Value:  loc.Value,
		})
	}

	// Execute rotation
	coordinator := rotation.NewCoordinator("integration-test")
	tx, err := coordinator.Execute("API_KEY", "new-secret-456", locations)
	require.NoError(t, err)
	assert.True(t, tx.AllSucceeded())
	assert.Equal(t, 2, tx.SuccessCount())

	// Verify files were updated
	contentA, _ := os.ReadFile(envA)
	contentB, _ := os.ReadFile(envB)
	assert.Contains(t, string(contentA), "API_KEY=new-secret-456")
	assert.Contains(t, string(contentA), "OTHER=keep")
	assert.Contains(t, string(contentB), "API_KEY=new-secret-456")
	assert.Contains(t, string(contentB), "DB_URL=postgres://localhost")
}

// TestRollbackOnFailure tests that successful updates are rolled back when one fails
func TestRollbackOnFailure(t *testing.T) {
	dir := t.TempDir()

	// Create env files
	envA := filepath.Join(dir, "a.env")
	envB := filepath.Join(dir, "b.env")

	os.WriteFile(envA, []byte("API_KEY=original\n"), 0600)
	os.WriteFile(envB, []byte("MISSING_KEY=value\n"), 0600) // Different key - will fail to update

	locations := []providers.Location{
		{Type: "local", Path: envA, Value: "original"},
		{Type: "local", Path: envB, Value: "original"}, // API_KEY doesn't exist in this file
	}

	coordinator := rotation.NewCoordinator("rollback-test")
	tx, err := coordinator.Execute("API_KEY", "new-value", locations)

	// Expect error because API_KEY doesn't exist in file B
	assert.Error(t, err)
	assert.NotNil(t, tx)
	assert.True(t, tx.HasFailures())

	// File A should be rolled back to original (from backup)
	contentA, _ := os.ReadFile(envA)
	assert.Contains(t, string(contentA), "original")
}

// TestMultipleFilesWithDifferentFormats tests rotation across files with different quote styles
func TestMultipleFilesWithDifferentFormats(t *testing.T) {
	dir := t.TempDir()

	// Different quote styles
	files := map[string]string{
		"unquoted.env": "API_KEY=secret123",
		"double.env":   `API_KEY="secret123"`,
		"single.env":   "API_KEY='secret123'",
		"exported.env": "export API_KEY=secret123",
	}

	var paths []string
	for name, content := range files {
		path := filepath.Join(dir, name)
		os.WriteFile(path, []byte(content), 0600)
		paths = append(paths, path)
	}

	// Find and rotate
	locals, _ := scanner.FindKey(paths, "API_KEY")
	require.Len(t, locals, 4)

	var locations []providers.Location
	for _, loc := range locals {
		locations = append(locations, providers.Location{
			Type:   "local",
			Path:   loc.Path,
			Value:  loc.Value,
		})
	}

	coordinator := rotation.NewCoordinator("format-test")
	tx, err := coordinator.Execute("API_KEY", "newsecret", locations)
	require.NoError(t, err)
	assert.Equal(t, 4, tx.SuccessCount())

	// Verify quote styles preserved
	c1, _ := os.ReadFile(filepath.Join(dir, "unquoted.env"))
	assert.Equal(t, "API_KEY=newsecret", string(c1))

	c2, _ := os.ReadFile(filepath.Join(dir, "double.env"))
	assert.Equal(t, `API_KEY="newsecret"`, string(c2))

	c3, _ := os.ReadFile(filepath.Join(dir, "single.env"))
	assert.Equal(t, "API_KEY='newsecret'", string(c3))

	c4, _ := os.ReadFile(filepath.Join(dir, "exported.env"))
	assert.Equal(t, "export API_KEY=newsecret", string(c4))
}

// TestDryRunDoesNotModify ensures dry run doesn't change files
func TestDryRunDoesNotModify(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	original := "API_KEY=original-value"
	os.WriteFile(envPath, []byte(original), 0600)

	locations := []providers.Location{
		{Type: "local", Path: envPath, Value: "original-value"},
	}

	// Call DryRun
	result := rotation.DryRun("API_KEY", locations)
	assert.Equal(t, "API_KEY", result.KeyName)
	assert.Len(t, result.Locations, 1)

	// File should be unchanged
	content, _ := os.ReadFile(envPath)
	assert.Equal(t, original, string(content))
}

// TestConfigLoad tests loading config with custom paths
func TestConfigLoad(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	configContent := `
scan_paths:
  - /tmp/test
exclude_patterns:
  - node_modules
file_patterns:
  - .env
ui:
  color: false
  verbose: true
audit:
  retention_days: 90
`
	os.WriteFile(configPath, []byte(configContent), 0600)

	cfg, err := config.Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, []string{"/tmp/test"}, cfg.ScanPaths)
	assert.False(t, cfg.UI.Color)
	assert.True(t, cfg.UI.Verbose)
	assert.Equal(t, 90, cfg.Audit.RetentionDays)
}

// TestScannerExcludes tests that exclude patterns work correctly
func TestScannerExcludes(t *testing.T) {
	dir := t.TempDir()

	// Create structure with node_modules
	os.MkdirAll(filepath.Join(dir, "src"), 0755)
	os.MkdirAll(filepath.Join(dir, "node_modules", "pkg"), 0755)
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)

	os.WriteFile(filepath.Join(dir, "src", ".env"), []byte("KEY=val"), 0600)
	os.WriteFile(filepath.Join(dir, "node_modules", "pkg", ".env"), []byte("KEY=val"), 0600)
	os.WriteFile(filepath.Join(dir, ".git", ".env"), []byte("KEY=val"), 0600)

	// Scan with excludes
	files, err := scanner.Scan(
		[]string{dir},
		[]string{"node_modules", ".git"},
		[]string{".env"},
	)
	require.NoError(t, err)

	// Should only find src/.env
	assert.Len(t, files, 1)
	assert.Contains(t, files[0], "src")
}
