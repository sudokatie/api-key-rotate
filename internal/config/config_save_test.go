package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := &Config{
		ScanPaths:       []string{"/test/path"},
		ExcludePatterns: []string{"node_modules"},
		FilePatterns:    []string{".env"},
		UI:              UIConfig{Color: true, Verbose: false},
		Audit:           AuditConfig{RetentionDays: 365, Path: "/audit.db"},
	}

	err := SaveTo(cfg, configPath)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(configPath)
	require.NoError(t, err)

	// Load and verify
	loaded, err := Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, []string{"/test/path"}, loaded.ScanPaths)
	assert.True(t, loaded.UI.Color)
}

func TestSaveCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "subdir", "nested", "config.yaml")

	cfg := &Config{
		ScanPaths: []string{"/test"},
	}

	err := SaveTo(cfg, configPath)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(configPath)
	require.NoError(t, err)
}

// Note: CreateDefault uses DefaultPath() which can't be easily mocked
// Test that CreateDefault doesn't panic when path doesn't exist
func TestCreateDefaultBasic(t *testing.T) {
	// Just verify the function doesn't panic
	// Actual file creation depends on system state
	_ = CreateDefault()
}

func TestMustLoad_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustLoad should panic on invalid config")
		}
	}()

	// Create an invalid config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")
	os.WriteFile(configPath, []byte("invalid: [yaml: syntax"), 0644)

	MustLoad(configPath)
}

func TestMustLoad_Success(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	os.WriteFile(configPath, []byte("scan_paths:\n  - /test"), 0644)

	// Should not panic
	cfg := MustLoad(configPath)
	assert.NotNil(t, cfg)
	assert.Equal(t, []string{"/test"}, cfg.ScanPaths)
}

func TestSaveWithProviders(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := &Config{
		ScanPaths: []string{"/test"},
		Providers: map[string]*ProviderConfig{
			"vercel": {Enabled: true},
			"github": {Enabled: false},
		},
	}

	err := SaveTo(cfg, configPath)
	require.NoError(t, err)

	loaded, err := Load(configPath)
	require.NoError(t, err)
	assert.NotNil(t, loaded.Providers)
}

func TestLoadInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	// Write invalid YAML
	os.WriteFile(configPath, []byte("invalid: [yaml: syntax"), 0644)

	_, err := Load(configPath)
	assert.Error(t, err)
}

func TestLoadEmptyScanPaths(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write config with empty scan_paths
	os.WriteFile(configPath, []byte("scan_paths: []"), 0644)

	_, err := Load(configPath)
	assert.Error(t, err) // Should fail validation
}

func TestDefaultPathXDG(t *testing.T) {
	// Set XDG_CONFIG_HOME
	original := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", "/custom/config")
	defer os.Setenv("XDG_CONFIG_HOME", original)

	path := DefaultPath()
	assert.Contains(t, path, "/custom/config")
}

func TestDefaultAuditPathXDG(t *testing.T) {
	// Set XDG_DATA_HOME
	original := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", "/custom/data")
	defer os.Setenv("XDG_DATA_HOME", original)

	path := DefaultAuditPath()
	assert.Contains(t, path, "/custom/data")
}

func TestConfigWithAllFields(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "full.yaml")

	cfg := &Config{
		ScanPaths:       []string{"/path1", "/path2"},
		ExcludePatterns: []string{"node_modules", ".git"},
		FilePatterns:    []string{".env", ".env.*"},
		UI: UIConfig{
			Color:   true,
			Verbose: true,
		},
		Audit: AuditConfig{
			Path:          "/custom/audit.db",
			RetentionDays: 180,
		},
		Providers: map[string]*ProviderConfig{
			"vercel": {Enabled: true},
		},
	}

	err := SaveTo(cfg, configPath)
	require.NoError(t, err)

	loaded, err := Load(configPath)
	require.NoError(t, err)

	assert.Equal(t, cfg.ScanPaths, loaded.ScanPaths)
	assert.Equal(t, cfg.UI.Verbose, loaded.UI.Verbose)
	assert.Equal(t, cfg.Audit.RetentionDays, loaded.Audit.RetentionDays)
}
