package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Cfg is the global configuration instance
var Cfg *Config

// Load reads configuration from file and environment
func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultPath()
	}

	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	SetDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Only return error if file exists but is invalid
			if _, statErr := os.Stat(path); statErr == nil {
				return nil, err
			}
		}
	}

	v.AutomaticEnv()
	v.SetEnvPrefix("API_KEY_ROTATE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	Cfg = &cfg
	return &cfg, nil
}

// MustLoad loads config or panics
func MustLoad(path string) *Config {
	cfg, err := Load(path)
	if err != nil {
		panic(err)
	}
	return cfg
}

// Save writes the configuration to the default path
func Save(cfg *Config) error {
	return SaveTo(cfg, DefaultPath())
}

// SaveTo writes the configuration to a specific path
func SaveTo(cfg *Config, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// CreateDefault creates a default config file if it doesn't exist
func CreateDefault() error {
	path := DefaultPath()
	if _, err := os.Stat(path); err == nil {
		return nil // Already exists
	}

	// Create default config
	cfg := &Config{}
	v := viper.New()
	SetDefaults(v)
	if err := v.Unmarshal(cfg); err != nil {
		return err
	}

	return SaveTo(cfg, path)
}
