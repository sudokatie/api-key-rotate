package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/sudokatie/api-key-rotate/internal/config"
	"github.com/sudokatie/api-key-rotate/internal/keyring"
	"github.com/sudokatie/api-key-rotate/internal/output"
	"github.com/sudokatie/api-key-rotate/internal/providers"
	"github.com/sudokatie/api-key-rotate/internal/scanner"
)

var (
	findLocalOnly bool
	findCloudOnly bool
	findFormat    string
)

var findCmd = &cobra.Command{
	Use:   "find <KEY_NAME>",
	Short: "Find all locations where a key exists",
	Long: `Search local .env files and cloud providers for a specific key.

Examples:
  api-key-rotate find OPENAI_API_KEY
  api-key-rotate find STRIPE_SECRET_KEY --local-only
  api-key-rotate find DATABASE_URL --format json`,
	Args: cobra.ExactArgs(1),
	RunE: runFind,
}

func init() {
	findCmd.Flags().BoolVar(&findLocalOnly, "local-only", false, "only scan local files")
	findCmd.Flags().BoolVar(&findCloudOnly, "cloud-only", false, "only check cloud providers")
	findCmd.Flags().StringVar(&findFormat, "format", "text", "output format (text, json, table)")

	rootCmd.AddCommand(findCmd)
}

func runFind(cmd *cobra.Command, args []string) error {
	keyName := args[0]

	var allLocations []providers.Location

	// Scan local files
	if !findCloudOnly {
		locals, err := findLocalLocations(keyName)
		if err != nil && verbose {
			fmt.Fprintf(os.Stderr, "Warning: local scan error: %v\n", err)
		}
		allLocations = append(allLocations, locals...)
	}

	// Check cloud providers
	if !findLocalOnly {
		cloudLocs := findCloudLocations(keyName)
		allLocations = append(allLocations, cloudLocs...)
	}

	// Output results
	if len(allLocations) == 0 {
		fmt.Fprintf(os.Stderr, "Key %q not found in any location.\n", keyName)
		os.Exit(ExitKeyNotFound)
	}

	formatter := output.New(findFormat, !noColor, true)
	fmt.Println(formatter.Locations(allLocations))

	return nil
}

func findLocalLocations(keyName string) ([]providers.Location, error) {
	files, err := scanner.Scan(
		config.Cfg.ScanPaths,
		config.Cfg.ExcludePatterns,
		config.Cfg.FilePatterns,
	)
	if err != nil {
		return nil, err
	}

	locals, err := scanner.FindKey(files, keyName)
	if err != nil {
		return nil, err
	}

	var locations []providers.Location
	for _, loc := range locals {
		locations = append(locations, providers.Location{
			Type:   "local",
			Path:   loc.Path,
			Exists: true,
			Value:  loc.Value,
		})
	}

	return locations, nil
}

func findCloudLocations(keyName string) []providers.Location {
	var locations []providers.Location

	for _, name := range providers.Names() {
		p, ok := providers.Get(name)
		if !ok {
			continue
		}

		// Check if provider is enabled in config
		if config.Cfg.Providers != nil {
			pc, exists := config.Cfg.Providers[name]
			if !exists || !pc.Enabled {
				continue
			}
		}

		// Get credentials from keyring
		token, err := keyring.Get(name)
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "Warning: %s: no credentials configured\n", name)
			}
			continue
		}

		creds := providers.Credentials{Token: token}
		if err := p.Configure(creds); err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "Warning: %s: %v\n", name, err)
			}
			continue
		}

		locs, err := p.Find(keyName)
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "Warning: %s: %v\n", name, err)
			}
			continue
		}

		locations = append(locations, locs...)
	}

	return locations
}
