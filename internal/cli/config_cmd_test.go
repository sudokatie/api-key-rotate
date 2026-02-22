package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sudokatie/api-key-rotate/internal/config"
)

func TestSetConfigValue_UIColor(t *testing.T) {
	cfg := &config.Config{UI: config.UIConfig{Color: true}}

	err := setConfigValue(cfg, "ui.color", "false")
	assert.NoError(t, err)
	assert.False(t, cfg.UI.Color)

	err = setConfigValue(cfg, "ui.color", "true")
	assert.NoError(t, err)
	assert.True(t, cfg.UI.Color)
}

func TestSetConfigValue_UIVerbose(t *testing.T) {
	cfg := &config.Config{}

	err := setConfigValue(cfg, "ui.verbose", "true")
	assert.NoError(t, err)
	assert.True(t, cfg.UI.Verbose)
}

func TestSetConfigValue_RetentionDays(t *testing.T) {
	cfg := &config.Config{}

	err := setConfigValue(cfg, "audit.retention_days", "180")
	assert.NoError(t, err)
	assert.Equal(t, 180, cfg.Audit.RetentionDays)
}

func TestSetConfigValue_RetentionDaysInvalid(t *testing.T) {
	cfg := &config.Config{}

	err := setConfigValue(cfg, "audit.retention_days", "not-a-number")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid integer")
}

func TestSetConfigValue_AuditPath(t *testing.T) {
	cfg := &config.Config{}

	err := setConfigValue(cfg, "audit.path", "/custom/path/audit.db")
	assert.NoError(t, err)
	assert.Equal(t, "/custom/path/audit.db", cfg.Audit.Path)
}

func TestSetConfigValue_UnknownKey(t *testing.T) {
	cfg := &config.Config{}

	err := setConfigValue(cfg, "unknown.key", "value")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown config key")
}

func TestGetConfigValue_ScanPaths(t *testing.T) {
	cfg := &config.Config{ScanPaths: []string{"/path/a", "/path/b"}}

	value, err := getConfigValue(cfg, "scan_paths")
	assert.NoError(t, err)
	assert.Equal(t, []string{"/path/a", "/path/b"}, value)
}

func TestGetConfigValue_UIColor(t *testing.T) {
	cfg := &config.Config{UI: config.UIConfig{Color: true}}

	value, err := getConfigValue(cfg, "ui.color")
	assert.NoError(t, err)
	assert.Equal(t, true, value)
}

func TestGetConfigValue_UISection(t *testing.T) {
	cfg := &config.Config{UI: config.UIConfig{Color: true, Verbose: false}}

	value, err := getConfigValue(cfg, "ui")
	assert.NoError(t, err)
	m := value.(map[string]interface{})
	assert.Equal(t, true, m["color"])
	assert.Equal(t, false, m["verbose"])
}

func TestGetConfigValue_AuditRetention(t *testing.T) {
	cfg := &config.Config{Audit: config.AuditConfig{RetentionDays: 365}}

	value, err := getConfigValue(cfg, "audit.retention_days")
	assert.NoError(t, err)
	assert.Equal(t, 365, value)
}

func TestGetConfigValue_UnknownKey(t *testing.T) {
	cfg := &config.Config{}

	_, err := getConfigValue(cfg, "unknown.key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown config key")
}

func TestConfigCmd_Subcommands(t *testing.T) {
	// Verify all subcommands are registered
	subcommands := make(map[string]bool)
	for _, cmd := range configCmd.Commands() {
		subcommands[cmd.Use] = true
	}

	assert.True(t, subcommands["show"])
	assert.True(t, subcommands["init"])
	assert.True(t, subcommands["path"])
	assert.True(t, subcommands["set <key> <value>"])
	assert.True(t, subcommands["get <key>"])
	assert.True(t, subcommands["scan-paths"])
}

func TestScanPathsCmd_Subcommands(t *testing.T) {
	subcommands := make(map[string]bool)
	for _, cmd := range scanPathsCmd.Commands() {
		subcommands[cmd.Use] = true
	}

	assert.True(t, subcommands["list"])
	assert.True(t, subcommands["add <path>"])
	assert.True(t, subcommands["remove <path>"])
}
