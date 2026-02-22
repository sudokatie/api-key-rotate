package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// SetDefaults configures default values
func SetDefaults(v *viper.Viper) {
	home, _ := os.UserHomeDir()

	v.SetDefault("scan_paths", []string{filepath.Join(home, "projects")})
	v.SetDefault("exclude_patterns", []string{
		"node_modules", ".git", "vendor", "__pycache__",
		".venv", "venv", "target", "dist", "build",
	})
	v.SetDefault("file_patterns", []string{".env", ".env.*", "*.env"})
	v.SetDefault("audit.path", DefaultAuditPath())
	v.SetDefault("audit.retention_days", 365)
	v.SetDefault("ui.color", true)
	v.SetDefault("ui.verbose", false)
}

// DefaultAuditPath returns the default path for the audit database
func DefaultAuditPath() string {
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataDir, "api-key-rotate", "audit.db")
}

// DefaultPath returns the default config file path
func DefaultPath() string {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "api-key-rotate", "config.yaml")
}
