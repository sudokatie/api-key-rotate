package scanner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	err := os.WriteFile(path, []byte("KEY=old\nOTHER=keep"), 0600)
	require.NoError(t, err)

	err = UpdateKey(path, "KEY", "new")
	require.NoError(t, err)

	content, _ := os.ReadFile(path)
	assert.Contains(t, string(content), "KEY=new")
	assert.Contains(t, string(content), "OTHER=keep")
}

func TestUpdatePreservesDoubleQuotes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	err := os.WriteFile(path, []byte(`KEY="old value"`), 0600)
	require.NoError(t, err)

	err = UpdateKey(path, "KEY", "new value")
	require.NoError(t, err)

	content, _ := os.ReadFile(path)
	assert.Equal(t, `KEY="new value"`, strings.TrimSpace(string(content)))
}

func TestUpdatePreservesSingleQuotes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	err := os.WriteFile(path, []byte(`KEY='old value'`), 0600)
	require.NoError(t, err)

	err = UpdateKey(path, "KEY", "new value")
	require.NoError(t, err)

	content, _ := os.ReadFile(path)
	assert.Equal(t, `KEY='new value'`, strings.TrimSpace(string(content)))
}

func TestUpdatePreservesExport(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	err := os.WriteFile(path, []byte("export KEY=old"), 0600)
	require.NoError(t, err)

	err = UpdateKey(path, "KEY", "new")
	require.NoError(t, err)

	content, _ := os.ReadFile(path)
	assert.Equal(t, "export KEY=new", strings.TrimSpace(string(content)))
}

func TestUpdateKeyNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	err := os.WriteFile(path, []byte("OTHER=value"), 0600)
	require.NoError(t, err)

	err = UpdateKey(path, "KEY", "new")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestBackupFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	err := os.WriteFile(path, []byte("KEY=value"), 0600)
	require.NoError(t, err)

	backupPath, err := BackupFile(path)
	require.NoError(t, err)
	assert.Contains(t, backupPath, ".bak.")

	// Verify backup content
	content, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	assert.Equal(t, "KEY=value", string(content))
}

func TestRestoreBackup(t *testing.T) {
	dir := t.TempDir()
	originalPath := filepath.Join(dir, ".env")
	backupPath := filepath.Join(dir, ".env.bak")

	err := os.WriteFile(backupPath, []byte("RESTORED=value"), 0600)
	require.NoError(t, err)

	err = RestoreBackup(backupPath, originalPath)
	require.NoError(t, err)

	content, err := os.ReadFile(originalPath)
	require.NoError(t, err)
	assert.Equal(t, "RESTORED=value", string(content))

	// Backup should be gone (moved)
	_, err = os.Stat(backupPath)
	assert.True(t, os.IsNotExist(err))
}

func TestUpdateFileNotFound(t *testing.T) {
	err := UpdateKey("/nonexistent/path/.env", "KEY", "value")
	assert.Error(t, err)
}
