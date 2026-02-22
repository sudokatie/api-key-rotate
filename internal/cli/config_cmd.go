package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/sudokatie/api-key-rotate/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long: `View and modify api-key-rotate configuration.

The config file is stored at ~/.config/api-key-rotate/config.yaml by default.
Override with --config flag or API_KEY_ROTATE_CONFIG env var.`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE:  runConfigShow,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create default configuration file",
	RunE:  runConfigInit,
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show configuration file path",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(config.DefaultPath())
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value by key path.

Examples:
  api-key-rotate config set ui.color false
  api-key-rotate config set audit.retention_days 180`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Long: `Get a configuration value by key path.

Examples:
  api-key-rotate config get scan_paths
  api-key-rotate config get ui.color`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigGet,
}

// Scan paths subcommands
var scanPathsCmd = &cobra.Command{
	Use:   "scan-paths",
	Short: "Manage scan paths",
}

var scanPathsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured scan paths",
	RunE:  runScanPathsList,
}

var scanPathsAddCmd = &cobra.Command{
	Use:   "add <path>",
	Short: "Add a scan path",
	Args:  cobra.ExactArgs(1),
	RunE:  runScanPathsAdd,
}

var scanPathsRemoveCmd = &cobra.Command{
	Use:   "remove <path>",
	Short: "Remove a scan path",
	Args:  cobra.ExactArgs(1),
	RunE:  runScanPathsRemove,
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)

	scanPathsCmd.AddCommand(scanPathsListCmd)
	scanPathsCmd.AddCommand(scanPathsAddCmd)
	scanPathsCmd.AddCommand(scanPathsRemoveCmd)
	configCmd.AddCommand(scanPathsCmd)

	rootCmd.AddCommand(configCmd)
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	if config.Cfg == nil {
		return fmt.Errorf("no configuration loaded")
	}

	var output []byte
	var err error

	if jsonOut {
		output, err = json.MarshalIndent(config.Cfg, "", "  ")
	} else {
		output, err = yaml.Marshal(config.Cfg)
	}

	if err != nil {
		return err
	}

	fmt.Println(string(output))
	return nil
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	path := config.DefaultPath()

	// Check if exists
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config already exists at %s", path)
	}

	if err := config.CreateDefault(); err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}

	fmt.Printf("Created default config at %s\n", path)
	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	if config.Cfg == nil {
		return fmt.Errorf("no configuration loaded")
	}

	// Parse and set the value
	if err := setConfigValue(config.Cfg, key, value); err != nil {
		return err
	}

	// Save
	if err := config.Save(config.Cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Set %s = %s\n", key, value)
	return nil
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]

	if config.Cfg == nil {
		return fmt.Errorf("no configuration loaded")
	}

	value, err := getConfigValue(config.Cfg, key)
	if err != nil {
		return err
	}

	// Format output
	switch v := value.(type) {
	case []string:
		for _, s := range v {
			fmt.Println(s)
		}
	case map[string]interface{}:
		data, _ := yaml.Marshal(v)
		fmt.Print(string(data))
	default:
		fmt.Println(v)
	}

	return nil
}

func runScanPathsList(cmd *cobra.Command, args []string) error {
	if config.Cfg == nil {
		return fmt.Errorf("no configuration loaded")
	}

	for _, path := range config.Cfg.ScanPaths {
		fmt.Println(path)
	}
	return nil
}

func runScanPathsAdd(cmd *cobra.Command, args []string) error {
	path := args[0]

	if config.Cfg == nil {
		return fmt.Errorf("no configuration loaded")
	}

	// Check for duplicates
	for _, existing := range config.Cfg.ScanPaths {
		if existing == path {
			return fmt.Errorf("path %q already configured", path)
		}
	}

	config.Cfg.ScanPaths = append(config.Cfg.ScanPaths, path)

	if err := config.Save(config.Cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Added scan path: %s\n", path)
	return nil
}

func runScanPathsRemove(cmd *cobra.Command, args []string) error {
	path := args[0]

	if config.Cfg == nil {
		return fmt.Errorf("no configuration loaded")
	}

	found := false
	newPaths := make([]string, 0, len(config.Cfg.ScanPaths))
	for _, existing := range config.Cfg.ScanPaths {
		if existing == path {
			found = true
			continue
		}
		newPaths = append(newPaths, existing)
	}

	if !found {
		return fmt.Errorf("path %q not found in config", path)
	}

	config.Cfg.ScanPaths = newPaths

	if err := config.Save(config.Cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Removed scan path: %s\n", path)
	return nil
}

// setConfigValue sets a config value by dot-notation path
func setConfigValue(cfg *config.Config, key string, value string) error {
	switch key {
	case "ui.color":
		cfg.UI.Color = value == "true" || value == "1"
	case "ui.verbose":
		cfg.UI.Verbose = value == "true" || value == "1"
	case "audit.retention_days":
		var days int
		if _, err := fmt.Sscanf(value, "%d", &days); err != nil {
			return fmt.Errorf("invalid integer: %s", value)
		}
		cfg.Audit.RetentionDays = days
	case "audit.path":
		cfg.Audit.Path = value
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
	return nil
}

// getConfigValue gets a config value by dot-notation path
func getConfigValue(cfg *config.Config, key string) (interface{}, error) {
	parts := strings.Split(key, ".")

	switch parts[0] {
	case "scan_paths":
		return cfg.ScanPaths, nil
	case "exclude_patterns":
		return cfg.ExcludePatterns, nil
	case "file_patterns":
		return cfg.FilePatterns, nil
	case "ui":
		if len(parts) == 1 {
			return map[string]interface{}{
				"color":   cfg.UI.Color,
				"verbose": cfg.UI.Verbose,
			}, nil
		}
		switch parts[1] {
		case "color":
			return cfg.UI.Color, nil
		case "verbose":
			return cfg.UI.Verbose, nil
		}
	case "audit":
		if len(parts) == 1 {
			return map[string]interface{}{
				"path":           cfg.Audit.Path,
				"retention_days": cfg.Audit.RetentionDays,
			}, nil
		}
		switch parts[1] {
		case "path":
			return cfg.Audit.Path, nil
		case "retention_days":
			return cfg.Audit.RetentionDays, nil
		}
	case "providers":
		if len(parts) == 1 {
			return cfg.Providers, nil
		}
	}

	return nil, fmt.Errorf("unknown config key: %s", key)
}
