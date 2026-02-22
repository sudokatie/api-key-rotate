package config

import (
	"os"
	"strings"

	"github.com/spf13/viper"
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
