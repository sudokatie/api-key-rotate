package rotation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sudokatie/api-key-rotate/internal/providers"
)

func TestRollbackNoCandidates(t *testing.T) {
	tx := NewTransaction("API_KEY", "new-value")
	tx.AddLocation(providers.Location{Path: "/path/.env"})
	// Location is still pending, not a rollback candidate

	err := Rollback(tx)

	assert.NoError(t, err)
}

func TestRollbackLocalFile(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	backupPath := filepath.Join(dir, ".env.bak")

	// Create original and backup
	err := os.WriteFile(envPath, []byte("API_KEY=new-value"), 0600)
	require.NoError(t, err)
	err = os.WriteFile(backupPath, []byte("API_KEY=old-value"), 0600)
	require.NoError(t, err)

	tx := NewTransaction("API_KEY", "new-value")
	tx.AddLocation(providers.Location{
		Type:  "local",
		Path:  envPath,
		Value: "old-value",
	})
	tx.RecordBackup(envPath, backupPath)
	tx.MarkSuccess(envPath)

	err = Rollback(tx)
	require.NoError(t, err)

	// Verify file was restored
	content, _ := os.ReadFile(envPath)
	assert.Equal(t, "API_KEY=old-value", string(content))

	// Verify status was updated
	locState := tx.GetLocation(envPath)
	assert.Equal(t, StatusRolledBack, locState.Status)
}

func TestRollbackLocalFileNoBackup(t *testing.T) {
	tx := NewTransaction("API_KEY", "new-value")
	tx.AddLocation(providers.Location{
		Type: "local",
		Path: "/nonexistent/.env",
	})
	tx.MarkSuccess("/nonexistent/.env")
	// No backup recorded

	err := Rollback(tx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no backup")
}

func TestRollbackMultipleLocations(t *testing.T) {
	dir := t.TempDir()
	env1 := filepath.Join(dir, ".env.1")
	env2 := filepath.Join(dir, ".env.2")
	backup1 := filepath.Join(dir, ".env.1.bak")
	backup2 := filepath.Join(dir, ".env.2.bak")

	// Set up files
	os.WriteFile(env1, []byte("KEY=new"), 0600)
	os.WriteFile(env2, []byte("KEY=new"), 0600)
	os.WriteFile(backup1, []byte("KEY=old1"), 0600)
	os.WriteFile(backup2, []byte("KEY=old2"), 0600)

	tx := NewTransaction("KEY", "new")
	tx.AddLocation(providers.Location{Type: "local", Path: env1, Value: "old1"})
	tx.AddLocation(providers.Location{Type: "local", Path: env2, Value: "old2"})
	tx.RecordBackup(env1, backup1)
	tx.RecordBackup(env2, backup2)
	tx.MarkSuccess(env1)
	tx.MarkSuccess(env2)

	err := Rollback(tx)
	require.NoError(t, err)

	content1, _ := os.ReadFile(env1)
	content2, _ := os.ReadFile(env2)
	assert.Equal(t, "KEY=old1", string(content1))
	assert.Equal(t, "KEY=old2", string(content2))
}

func TestCleanupBackupsOnlyOnSuccess(t *testing.T) {
	tx := NewTransaction("KEY", "new")
	tx.AddLocation(providers.Location{Type: "local", Path: "/path/.env"})
	tx.MarkFailed("/path/.env", assert.AnError)

	// Should not clean up if not all succeeded
	err := CleanupBackups(tx)
	assert.NoError(t, err)
}

func TestCleanupBackupsRemovesFiles(t *testing.T) {
	dir := t.TempDir()
	backupPath := filepath.Join(dir, ".env.bak")
	os.WriteFile(backupPath, []byte("backup"), 0600)

	tx := NewTransaction("KEY", "new")
	tx.Locations = []LocationState{
		{
			Location:   providers.Location{Type: "local", Path: filepath.Join(dir, ".env")},
			BackupPath: backupPath,
			Status:     StatusSuccess,
		},
	}

	err := CleanupBackups(tx)
	require.NoError(t, err)

	// Verify backup was deleted
	_, err = os.Stat(backupPath)
	assert.True(t, os.IsNotExist(err))
}
