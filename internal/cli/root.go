package cli

import (
	"github.com/spf13/cobra"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

// SetVersionInfo sets version information for the CLI
func SetVersionInfo(v, c, b string) {
	version = v
	commit = c
	buildDate = b
}

var rootCmd = &cobra.Command{
	Use:   "api-key-rotate",
	Short: "Rotate API keys across all your environments",
	Long: `api-key-rotate scans for API keys in .env files and environment variables,
then helps you rotate them across services like Vercel, GitHub Secrets, and more.

Features:
  - Scan directories for .env files containing API keys
  - Rotate keys with a single command
  - Update Vercel environment variables
  - Update GitHub repository secrets
  - Audit trail of all rotations
  - Dry-run mode for safety`,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().String("config", "", "Config file path")
}
