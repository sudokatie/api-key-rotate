package rotation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sudokatie/api-key-rotate/internal/providers"
)

func TestDryRun(t *testing.T) {
	locations := []providers.Location{
		{
			Type:  "local",
			Path:  "/path/to/.env",
			Value: "sk_live_12345678",
		},
		{
			Type:        "vercel",
			Path:        "proj-123/production/API_KEY",
			Project:     "my-app",
			Environment: "production",
			Value:       "sk_live_12345678",
		},
	}

	result := DryRun("API_KEY", locations)

	assert.Equal(t, "API_KEY", result.KeyName)
	require.Len(t, result.Locations, 2)

	assert.Equal(t, "local", result.Locations[0].Type)
	assert.Equal(t, "/path/to/.env", result.Locations[0].Path)
	assert.Equal(t, "sk_l****", result.Locations[0].CurrentMask)

	assert.Equal(t, "vercel", result.Locations[1].Type)
	assert.Equal(t, "my-app", result.Locations[1].Project)
	assert.Equal(t, "production", result.Locations[1].Environment)
}

func TestCoordinatorExecuteEmptyLocations(t *testing.T) {
	c := NewCoordinator("test")

	_, err := c.Execute("API_KEY", "new-value", []providers.Location{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no locations")
}

func TestCoordinatorExecuteLocalFile(t *testing.T) {
	// Create a test .env file
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	err := os.WriteFile(envPath, []byte("API_KEY=old-value\nOTHER=keep"), 0600)
	require.NoError(t, err)

	c := NewCoordinator("test-user")

	locations := []providers.Location{
		{
			Type:  "local",
			Path:  envPath,
			Value: "old-value",
		},
	}

	tx, err := c.Execute("API_KEY", "new-value", locations)

	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.True(t, tx.AllSucceeded())
	assert.Equal(t, 1, tx.SuccessCount())

	// Verify file was updated
	content, _ := os.ReadFile(envPath)
	assert.Contains(t, string(content), "API_KEY=new-value")
	assert.Contains(t, string(content), "OTHER=keep")
}

func TestCoordinatorExecuteWithConfirmationCancelled(t *testing.T) {
	c := NewCoordinator("test")

	locations := []providers.Location{
		{Type: "local", Path: "/path/.env"},
	}

	tx, err := c.ExecuteWithConfirmation("API_KEY", "new", locations, func(dr *DryRunResult) bool {
		return false // User cancels
	})

	assert.Nil(t, tx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled")
}

func TestCoordinatorExecuteWithConfirmationConfirmed(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	err := os.WriteFile(envPath, []byte("API_KEY=old"), 0600)
	require.NoError(t, err)

	c := NewCoordinator("test")

	locations := []providers.Location{
		{Type: "local", Path: envPath, Value: "old"},
	}

	confirmCalled := false
	tx, err := c.ExecuteWithConfirmation("API_KEY", "new", locations, func(dr *DryRunResult) bool {
		confirmCalled = true
		assert.Equal(t, "API_KEY", dr.KeyName)
		assert.Len(t, dr.Locations, 1)
		return true
	})

	require.NoError(t, err)
	assert.True(t, confirmCalled)
	assert.NotNil(t, tx)
	assert.True(t, tx.AllSucceeded())
}

func TestCoordinatorBackupCreated(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	err := os.WriteFile(envPath, []byte("API_KEY=old"), 0600)
	require.NoError(t, err)

	c := NewCoordinator("test")

	locations := []providers.Location{
		{Type: "local", Path: envPath, Value: "old"},
	}

	tx, err := c.Execute("API_KEY", "new", locations)
	require.NoError(t, err)

	// Check backup was recorded
	locState := tx.GetLocation(envPath)
	require.NotNil(t, locState)
	assert.NotEmpty(t, locState.BackupPath)

	// Verify backup file exists
	_, err = os.Stat(locState.BackupPath)
	assert.NoError(t, err)

	// Verify backup contains original content
	backupContent, _ := os.ReadFile(locState.BackupPath)
	assert.Equal(t, "API_KEY=old", string(backupContent))
}
