package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/sudokatie/api-key-rotate/internal/output"
	"github.com/sudokatie/api-key-rotate/internal/providers"
	"github.com/sudokatie/api-key-rotate/internal/rotation"
)

var (
	rotateExecute   bool
	rotateNewKey    string
	rotateForce     bool
	rotateLocalOnly bool
	rotateCloudOnly bool
	rotateLocations []string
	rotateExclude   []string
	rotateFormat    string
)

var rotateCmd = &cobra.Command{
	Use:   "rotate <KEY_NAME>",
	Short: "Rotate an API key across all locations",
	Long: `Find and update an API key in all local .env files and cloud providers.

By default, performs a dry run showing what would be changed.
Use --execute to perform the actual rotation.

Examples:
  api-key-rotate rotate OPENAI_API_KEY                    # Dry run
  api-key-rotate rotate OPENAI_API_KEY --execute          # Execute
  api-key-rotate rotate STRIPE_KEY --new-key=sk_live_xxx  # Provide new key
  api-key-rotate rotate DB_URL --local-only --force       # Skip confirmation`,
	Args: cobra.ExactArgs(1),
	RunE: runRotate,
}

func init() {
	rotateCmd.Flags().BoolVarP(&rotateExecute, "execute", "e", false, "actually perform the rotation")
	rotateCmd.Flags().StringVar(&rotateNewKey, "new-key", "", "new key value (prompted if not provided)")
	rotateCmd.Flags().BoolVarP(&rotateForce, "force", "f", false, "skip confirmation prompt")
	rotateCmd.Flags().BoolVar(&rotateLocalOnly, "local-only", false, "only update local files")
	rotateCmd.Flags().BoolVar(&rotateCloudOnly, "cloud-only", false, "only update cloud providers")
	rotateCmd.Flags().StringSliceVar(&rotateLocations, "locations", nil, "only update specific locations")
	rotateCmd.Flags().StringSliceVar(&rotateExclude, "exclude", nil, "exclude specific locations")
	rotateCmd.Flags().StringVar(&rotateFormat, "format", "text", "output format (text, json, table)")

	rootCmd.AddCommand(rotateCmd)
}

func runRotate(cmd *cobra.Command, args []string) error {
	keyName := args[0]

	// Find all locations
	locations, err := gatherLocations(keyName)
	if err != nil {
		return err
	}

	if len(locations) == 0 {
		fmt.Fprintf(os.Stderr, "Key %q not found in any location.\n", keyName)
		os.Exit(4)
	}

	// Filter locations
	locations = filterLocations(locations)

	if len(locations) == 0 {
		fmt.Fprintf(os.Stderr, "No locations match the specified filters.\n")
		return nil
	}

	// Dry run mode (default)
	if !rotateExecute {
		formatter := output.New(rotateFormat, !noColor, true)
		dryRun := rotation.DryRun(keyName, locations)
		fmt.Println(formatter.DryRun(dryRun))
		fmt.Println()
		fmt.Println("Run with --execute to perform the rotation.")
		return nil
	}

	// Get new key value
	newKey := rotateNewKey
	if newKey == "" {
		var err error
		newKey, err = promptForNewKey(keyName)
		if err != nil {
			return err
		}
	}

	if newKey == "" {
		return fmt.Errorf("new key value cannot be empty")
	}

	// Confirmation
	if !rotateForce {
		dryRun := rotation.DryRun(keyName, locations)
		if !confirmRotation(dryRun) {
			fmt.Println("Rotation cancelled.")
			return nil
		}
	}

	// Execute rotation
	coordinator := rotation.NewCoordinator("cli")
	tx, err := coordinator.Execute(keyName, newKey, locations)

	// Output results
	formatter := output.New(rotateFormat, !noColor, true)

	if err != nil {
		fmt.Fprintln(os.Stderr, formatter.Result(false, err.Error()))
		if tx != nil {
			printTransactionSummary(tx)
		}
		os.Exit(5)
	}

	fmt.Println(formatter.Result(true, fmt.Sprintf("Rotated %s in %d locations", keyName, tx.SuccessCount())))
	printTransactionSummary(tx)

	return nil
}

// gatherLocations finds all locations for a key
func gatherLocations(keyName string) ([]providers.Location, error) {
	var allLocations []providers.Location

	// Scan local files
	if !rotateCloudOnly {
		locals, err := findLocalLocations(keyName)
		if err != nil && verbose {
			fmt.Fprintf(os.Stderr, "Warning: local scan error: %v\n", err)
		}
		allLocations = append(allLocations, locals...)
	}

	// Check cloud providers
	if !rotateLocalOnly {
		cloudLocs := findCloudLocations(keyName)
		allLocations = append(allLocations, cloudLocs...)
	}

	return allLocations, nil
}

// filterLocations applies --locations and --exclude filters
func filterLocations(locs []providers.Location) []providers.Location {
	if len(rotateLocations) == 0 && len(rotateExclude) == 0 {
		return locs
	}

	var filtered []providers.Location
	for _, loc := range locs {
		path := locationPath(loc)

		// Include filter
		if len(rotateLocations) > 0 {
			found := false
			for _, include := range rotateLocations {
				if strings.Contains(path, include) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Exclude filter
		excluded := false
		for _, exclude := range rotateExclude {
			if strings.Contains(path, exclude) {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		filtered = append(filtered, loc)
	}

	return filtered
}

// locationPath returns a string representation of a location for filtering
func locationPath(loc providers.Location) string {
	if loc.Type == "local" {
		return loc.Path
	}
	if loc.Project != "" {
		return fmt.Sprintf("%s/%s/%s", loc.Provider, loc.Project, loc.Environment)
	}
	return fmt.Sprintf("%s/%s", loc.Provider, loc.Path)
}

// promptForNewKey prompts the user to enter a new key value (no echo)
func promptForNewKey(keyName string) (string, error) {
	fmt.Printf("Enter new value for %s: ", keyName)

	// Use term for password-style input
	if term.IsTerminal(int(os.Stdin.Fd())) {
		password, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println() // New line after hidden input
		if err != nil {
			return "", fmt.Errorf("failed to read key value: %w", err)
		}
		return strings.TrimSpace(string(password)), nil
	}

	// Fallback for non-terminal (piped input)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read key value: %w", err)
	}
	return strings.TrimSpace(line), nil
}

// confirmRotation asks the user to confirm the rotation
func confirmRotation(dryRun *rotation.DryRunResult) bool {
	formatter := output.New("text", !noColor, true)
	fmt.Println(formatter.DryRun(dryRun))
	fmt.Println()

	warning := "This will update %d locations. Continue?"
	if !noColor {
		warning = color.YellowString(warning)
	}
	fmt.Printf(warning+" [y/N] ", len(dryRun.Locations))

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// printTransactionSummary outputs details about the transaction
func printTransactionSummary(tx *rotation.Transaction) {
	successColor := color.New(color.FgGreen)
	failColor := color.New(color.FgRed)

	fmt.Println()
	fmt.Printf("Summary: %d succeeded, %d failed\n", tx.SuccessCount(), tx.FailedCount())

	if verbose || tx.HasFailures() {
		fmt.Println()
		for _, loc := range tx.Locations {
			status := string(loc.Status)
			path := locationPath(loc.Location)

			if !noColor {
				switch loc.Status {
				case rotation.StatusSuccess:
					status = successColor.Sprint(status)
				case rotation.StatusFailed:
					status = failColor.Sprint(status)
				}
			}

			fmt.Printf("  [%s] %s", status, path)
			if loc.Error != nil {
				fmt.Printf(" - %v", loc.Error)
			}
			fmt.Println()
		}
	}
}
