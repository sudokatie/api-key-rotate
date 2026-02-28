package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/sudokatie/api-key-rotate/internal/config"
	"github.com/sudokatie/api-key-rotate/internal/keyring"
	"github.com/sudokatie/api-key-rotate/internal/providers"
)

var providersCmd = &cobra.Command{
	Use:   "providers",
	Short: "Manage cloud providers",
	Long: `Configure and manage cloud provider connections.

Providers store API keys securely in your system keychain.`,
}

var providersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured providers",
	RunE:  runProvidersList,
}

var providersAddCmd = &cobra.Command{
	Use:   "add <provider>",
	Short: "Add or update a provider",
	Long: `Configure a cloud provider by storing its API token.

Available providers:
  vercel    - Vercel deployments and environment variables
  github    - GitHub Actions secrets

Examples:
  api-key-rotate providers add vercel
  api-key-rotate providers add github`,
	Args: cobra.ExactArgs(1),
	RunE: runProvidersAdd,
}

var providersRemoveCmd = &cobra.Command{
	Use:   "remove <provider>",
	Short: "Remove a provider",
	Args:  cobra.ExactArgs(1),
	RunE:  runProvidersRemove,
}

var providersTestCmd = &cobra.Command{
	Use:   "test [provider]",
	Short: "Test provider connection",
	Long: `Test connectivity to a provider.

If no provider is specified, tests all configured providers.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runProvidersTest,
}

func init() {
	providersCmd.AddCommand(providersListCmd)
	providersCmd.AddCommand(providersAddCmd)
	providersCmd.AddCommand(providersRemoveCmd)
	providersCmd.AddCommand(providersTestCmd)

	rootCmd.AddCommand(providersCmd)
}

func runProvidersList(cmd *cobra.Command, args []string) error {
	available := providers.Names()
	if len(available) == 0 {
		fmt.Println("No providers registered.")
		return nil
	}

	fmt.Println("Available providers:")
	fmt.Println()

	for _, name := range available {
		status := getProviderStatus(name)
		fmt.Printf("  %s %s\n", formatProviderName(name), status)
	}

	return nil
}

func getProviderStatus(name string) string {
	// Check if credentials exist
	hasToken := keyring.Exists(name)

	// Check if enabled in config
	enabled := false
	if config.Cfg != nil && config.Cfg.Providers != nil {
		if pc, ok := config.Cfg.Providers[name]; ok {
			enabled = pc.Enabled
		}
	}

	if !hasToken {
		if noColor {
			return "[not configured]"
		}
		return color.HiBlackString("[not configured]")
	}

	if enabled {
		if noColor {
			return "[enabled]"
		}
		return color.GreenString("[enabled]")
	}

	if noColor {
		return "[credentials stored]"
	}
	return color.YellowString("[credentials stored]")
}

func formatProviderName(name string) string {
	// Pad to consistent width
	return fmt.Sprintf("%-10s", name)
}

func runProvidersAdd(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Check if provider exists
	p, ok := providers.Get(name)
	if !ok {
		return fmt.Errorf("unknown provider: %s\nAvailable: %s", name, strings.Join(providers.Names(), ", "))
	}

	// Prompt for token
	fmt.Printf("Enter API token for %s: ", name)
	token, err := readPassword()
	if err != nil {
		return fmt.Errorf("failed to read token: %w", err)
	}

	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	// Test the token
	creds := providers.Credentials{Token: token}
	if err := p.Configure(creds); err != nil {
		return fmt.Errorf("invalid credentials: %w", err)
	}

	if err := p.Test(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: token validation failed: %v\n", err)
		fmt.Print("Save anyway? [y/N] ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			return fmt.Errorf("cancelled")
		}
	}

	// Store in keyring
	if err := keyring.Set(name, token); err != nil {
		return fmt.Errorf("failed to store credentials: %w", err)
	}

	// Enable in config
	if config.Cfg != nil {
		if config.Cfg.Providers == nil {
			config.Cfg.Providers = make(map[string]*config.ProviderConfig)
		}
		config.Cfg.Providers[name] = &config.ProviderConfig{Enabled: true}
		if err := config.Save(config.Cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not update config: %v\n", err)
		}
	}

	fmt.Printf("Provider %s configured successfully.\n", name)
	return nil
}

func runProvidersRemove(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Check if provider exists
	if _, ok := providers.Get(name); !ok {
		return fmt.Errorf("unknown provider: %s", name)
	}

	// Remove from keyring
	if err := keyring.Delete(name); err != nil {
		// Ignore not-found errors
		if !strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("failed to remove credentials: %w", err)
		}
	}

	// Disable in config
	if config.Cfg != nil && config.Cfg.Providers != nil {
		if pc, ok := config.Cfg.Providers[name]; ok {
			pc.Enabled = false
			if err := config.Save(config.Cfg); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not update config: %v\n", err)
			}
		}
	}

	fmt.Printf("Provider %s removed.\n", name)
	return nil
}

func runProvidersTest(cmd *cobra.Command, args []string) error {
	var providerNames []string

	if len(args) > 0 {
		// Test specific provider
		providerNames = []string{args[0]}
	} else {
		// Test all configured providers
		providerNames = providers.Names()
	}

	if len(providerNames) == 0 {
		fmt.Println("No providers to test.")
		return nil
	}

	success := true
	for _, name := range providerNames {
		p, ok := providers.Get(name)
		if !ok {
			printTestResult(name, false, fmt.Errorf("unknown provider"))
			success = false
			continue
		}

		// Get credentials
		token, err := keyring.Get(name)
		if err != nil {
			printTestResult(name, false, fmt.Errorf("no credentials"))
			success = false
			continue
		}

		// Configure and test
		creds := providers.Credentials{Token: token}
		if err := p.Configure(creds); err != nil {
			printTestResult(name, false, err)
			success = false
			continue
		}

		if err := p.Test(); err != nil {
			printTestResult(name, false, err)
			success = false
			continue
		}

		printTestResult(name, true, nil)
	}

	if !success {
		os.Exit(ExitProviderError)
	}

	return nil
}

func printTestResult(name string, ok bool, err error) {
	if ok {
		if noColor {
			fmt.Printf("%s: OK\n", name)
		} else {
			fmt.Printf("%s: %s\n", name, color.GreenString("OK"))
		}
	} else {
		if noColor {
			fmt.Printf("%s: FAILED - %v\n", name, err)
		} else {
			fmt.Printf("%s: %s - %v\n", name, color.RedString("FAILED"), err)
		}
	}
}

func readPassword() (string, error) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		password, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println() // New line after hidden input
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(password)), nil
	}

	// Fallback for non-terminal
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
