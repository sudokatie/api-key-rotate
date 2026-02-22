package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandHome(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{"~/projects", home + "/projects"},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"~", "~"}, // Single ~ is not expanded
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ExpandHome(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				ScanPaths: []string{"/tmp"},
			},
			wantErr: false,
		},
		{
			name: "empty scan paths",
			config: Config{
				ScanPaths: []string{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	// Test loading with defaults (no config file)
	tmpDir := t.TempDir()
	nonExistentConfig := filepath.Join(tmpDir, "config.yaml")

	cfg, err := Load(nonExistentConfig)
	require.NoError(t, err)
	assert.NotEmpty(t, cfg.ScanPaths)
	assert.NotEmpty(t, cfg.ExcludePatterns)
}

func TestLoadWithFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
scan_paths:
  - /custom/path
exclude_patterns:
  - custom_exclude
ui:
  verbose: true
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, []string{"/custom/path"}, cfg.ScanPaths)
	assert.True(t, cfg.UI.Verbose)
}

func TestDefaultPaths(t *testing.T) {
	// These should not panic
	path := DefaultPath()
	assert.NotEmpty(t, path)

	auditPath := DefaultAuditPath()
	assert.NotEmpty(t, auditPath)
}
