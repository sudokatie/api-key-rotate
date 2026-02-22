package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/sudokatie/api-key-rotate/internal/config"
)

var (
	cfgFile string
	verbose bool
	quiet   bool
	noColor bool
	jsonOut bool

	versionInfo struct {
		Version   string
		Commit    string
		BuildDate string
	}
)

// SetVersionInfo sets version information for the CLI
func SetVersionInfo(v, c, b string) {
	versionInfo.Version = v
	versionInfo.Commit = c
	versionInfo.BuildDate = b
}

var rootCmd = &cobra.Command{
	Use:   "api-key-rotate",
	Short: "Rotate API keys across all your environments",
	Long: `API Key Rotate finds and updates API keys across local .env files
and cloud services like Vercel and GitHub Actions.

Run 'api-key-rotate find <KEY_NAME>' to see where a key exists.
Run 'api-key-rotate <KEY_NAME>' to rotate it (dry run by default).`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "version" {
			return nil
		}

		_, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("config: %w", err)
		}

		if noColor {
			color.NoColor = true
		}

		return nil
	},
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "suppress output")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colors")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "JSON output")
}
