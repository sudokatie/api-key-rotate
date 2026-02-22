package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScan(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, ".env"), []byte("KEY=val"), 0600)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, ".env.local"), []byte("KEY=val"), 0600)
	require.NoError(t, err)

	files, err := Scan([]string{dir}, nil, []string{".env", ".env.*"})
	require.NoError(t, err)
	assert.Len(t, files, 2)
}

func TestScanExcludes(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, ".env"), []byte("KEY=val"), 0600)
	require.NoError(t, err)

	nodeModules := filepath.Join(dir, "node_modules")
	err = os.MkdirAll(nodeModules, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(nodeModules, ".env"), []byte("KEY=val"), 0600)
	require.NoError(t, err)

	files, err := Scan([]string{dir}, []string{"node_modules"}, []string{".env"})
	require.NoError(t, err)
	assert.Len(t, files, 1)
}

func TestScanNestedDirs(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "project", "config")
	err := os.MkdirAll(subdir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(dir, ".env"), []byte("ROOT=1"), 0600)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(subdir, ".env"), []byte("NESTED=1"), 0600)
	require.NoError(t, err)

	files, err := Scan([]string{dir}, nil, []string{".env"})
	require.NoError(t, err)
	assert.Len(t, files, 2)
}

func TestScanNonExistentPath(t *testing.T) {
	files, err := Scan([]string{"/nonexistent/path"}, nil, []string{".env"})
	require.NoError(t, err) // Should not error, just return empty
	assert.Empty(t, files)
}

func TestScanMultiplePaths(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	err := os.WriteFile(filepath.Join(dir1, ".env"), []byte("KEY1=val"), 0600)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir2, ".env"), []byte("KEY2=val"), 0600)
	require.NoError(t, err)

	files, err := Scan([]string{dir1, dir2}, nil, []string{".env"})
	require.NoError(t, err)
	assert.Len(t, files, 2)
}

func TestFindKey(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, ".env"), []byte("API_KEY=secret\nOTHER=val"), 0600)
	require.NoError(t, err)

	files, err := Scan([]string{dir}, nil, []string{".env"})
	require.NoError(t, err)

	locs, err := FindKey(files, "API_KEY")
	require.NoError(t, err)
	require.Len(t, locs, 1)
	assert.Equal(t, "secret", locs[0].Value)
	assert.Equal(t, 1, locs[0].Line)
}

func TestFindKeyMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, ".env"), []byte("API_KEY=secret1"), 0600)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, ".env.local"), []byte("API_KEY=secret2"), 0600)
	require.NoError(t, err)

	files, err := Scan([]string{dir}, nil, []string{".env", ".env.*"})
	require.NoError(t, err)

	locs, err := FindKey(files, "API_KEY")
	require.NoError(t, err)
	assert.Len(t, locs, 2)
}

func TestFindKeyNotFound(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, ".env"), []byte("OTHER=val"), 0600)
	require.NoError(t, err)

	files, err := Scan([]string{dir}, nil, []string{".env"})
	require.NoError(t, err)

	locs, err := FindKey(files, "API_KEY")
	require.NoError(t, err)
	assert.Empty(t, locs)
}

func TestFindAllKeys(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, ".env"), []byte("KEY1=val1\nKEY2=val2"), 0600)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, ".env.local"), []byte("KEY2=override\nKEY3=val3"), 0600)
	require.NoError(t, err)

	files, err := Scan([]string{dir}, nil, []string{".env", ".env.*"})
	require.NoError(t, err)

	keys, err := FindAllKeys(files)
	require.NoError(t, err)
	assert.Len(t, keys, 3) // KEY1, KEY2, KEY3 (deduplicated)
}

func TestExpandHome(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{"~/projects", filepath.Join(home, "projects")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}

	for _, tt := range tests {
		result := expandHome(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}
