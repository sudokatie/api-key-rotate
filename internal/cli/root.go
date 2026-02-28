package cli

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/sudokatie/api-key-rotate/internal/config"
)

// Exit codes per spec
const (
	ExitSuccess         = 0
	ExitGeneralError    = 1
	ExitConfigError     = 2
	ExitProviderError   = 3
	ExitKeyNotFound     = 4
	ExitRotationFailed  = 5
	ExitRollbackFailed  = 6
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
	Use:   "api-key-rotate [KEY_NAME]",
	Short: "Rotate API keys across all your environments",
	Long: `API Key Rotate finds and updates API keys across local .env files
and cloud services like Vercel and GitHub Actions.

Run 'api-key-rotate find <KEY_NAME>' to see where a key exists.
Run 'api-key-rotate <KEY_NAME>' to rotate it (dry run by default).`,
	SilenceUsage:  true,
	SilenceErrors: true,
	Args:          cobra.MaximumNArgs(1),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "version" {
			return nil
		}

		_, err := config.Load(cfgFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(ExitConfigError)
		}

		if noColor {
			color.NoColor = true
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// If a key name is provided directly, delegate to rotate command
		if len(args) == 1 {
			return runRotate(cmd, args)
		}
		// No args - show help
		return cmd.Help()
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

	// Add rotate flags to root for when key is passed directly
	rootCmd.Flags().BoolVarP(&rotateExecute, "execute", "e", false, "actually perform the rotation")
	rootCmd.Flags().StringVar(&rotateNewKey, "new-key", "", "new key value (prompted if not provided)")
	rootCmd.Flags().BoolVarP(&rotateForce, "force", "f", false, "skip confirmation prompt")
	rootCmd.Flags().BoolVar(&rotateSkipLocal, "skip-local", false, "skip local file updates")
	rootCmd.Flags().BoolVar(&rotateSkipCloud, "skip-cloud", false, "skip cloud provider updates")
	rootCmd.Flags().StringSliceVar(&rotateLocations, "locations", nil, "only update specific locations")
	rootCmd.Flags().StringSliceVar(&rotateExclude, "exclude", nil, "exclude specific locations")
	rootCmd.Flags().StringVar(&rotateFormat, "format", "text", "output format (text, json, table)")
}
