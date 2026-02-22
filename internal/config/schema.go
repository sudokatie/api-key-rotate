package config

import (
	"fmt"
	"os"
)

// Config represents the application configuration
type Config struct {
	ScanPaths       []string                   `mapstructure:"scan_paths" yaml:"scan_paths"`
	ExcludePatterns []string                   `mapstructure:"exclude_patterns" yaml:"exclude_patterns"`
	FilePatterns    []string                   `mapstructure:"file_patterns" yaml:"file_patterns"`
	Providers       map[string]*ProviderConfig `mapstructure:"providers" yaml:"providers"`
	Audit           AuditConfig                `mapstructure:"audit" yaml:"audit"`
	UI              UIConfig                   `mapstructure:"ui" yaml:"ui"`
}

// ProviderConfig holds settings for a single provider
type ProviderConfig struct {
	Enabled bool     `mapstructure:"enabled" yaml:"enabled"`
	Orgs    []string `mapstructure:"orgs" yaml:"orgs"`
}

// AuditConfig holds audit trail settings
type AuditConfig struct {
	Path          string `mapstructure:"path" yaml:"path"`
	RetentionDays int    `mapstructure:"retention_days" yaml:"retention_days"`
}

// UIConfig holds UI preferences
type UIConfig struct {
	Color   bool `mapstructure:"color" yaml:"color"`
	Verbose bool `mapstructure:"verbose" yaml:"verbose"`
}

// Validate checks the configuration for errors
func (c *Config) Validate() error {
	if len(c.ScanPaths) == 0 {
		return fmt.Errorf("at least one scan_path required")
	}
	for _, path := range c.ScanPaths {
		expanded := ExpandHome(path)
		if _, err := os.Stat(expanded); os.IsNotExist(err) {
			// Warning, not error - path may not exist yet
		}
	}
	return nil
}

// ExpandHome expands ~ to home directory
func ExpandHome(path string) string {
	if len(path) > 1 && path[:2] == "~/" {
		home, _ := os.UserHomeDir()
		return home + path[1:]
	}
	return path
}
