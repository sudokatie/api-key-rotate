package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/sudokatie/api-key-rotate/internal/audit"
	"github.com/sudokatie/api-key-rotate/internal/output"
)

var (
	historyKey    string
	historyStatus string
	historySince  string
	historyUntil  string
	historyLimit  int
	historyFormat string
)

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "View rotation history",
	Long: `View the history of key rotations from the audit log.

Examples:
  api-key-rotate history                        # Show recent rotations
  api-key-rotate history --key OPENAI_API_KEY   # Filter by key name
  api-key-rotate history --since 2024-01-01     # Since date
  api-key-rotate history --status failed        # Only failed rotations
  api-key-rotate history --limit 100            # More results`,
	RunE: runHistory,
}

func init() {
	historyCmd.Flags().StringVar(&historyKey, "key", "", "filter by key name")
	historyCmd.Flags().StringVar(&historyStatus, "status", "", "filter by status (success, failed, partial)")
	historyCmd.Flags().StringVar(&historySince, "since", "", "show entries since date (YYYY-MM-DD)")
	historyCmd.Flags().StringVar(&historyUntil, "until", "", "show entries until date (YYYY-MM-DD)")
	historyCmd.Flags().IntVar(&historyLimit, "limit", 50, "maximum entries to show")
	historyCmd.Flags().StringVar(&historyFormat, "format", "text", "output format (text, json, table)")

	rootCmd.AddCommand(historyCmd)
}

func runHistory(cmd *cobra.Command, args []string) error {
	// Audit DB is initialized in PersistentPreRunE
	// Build query options
	opts := audit.QueryOptions{
		KeyName: historyKey,
		Status:  historyStatus,
		Limit:   historyLimit,
	}

	// Parse date filters
	if historySince != "" {
		t, err := parseDate(historySince)
		if err != nil {
			return fmt.Errorf("invalid --since date: %w", err)
		}
		opts.Since = t
	}

	if historyUntil != "" {
		t, err := parseDate(historyUntil)
		if err != nil {
			return fmt.Errorf("invalid --until date: %w", err)
		}
		// End of day
		opts.Until = t.Add(24*time.Hour - time.Second)
	}

	// Query entries
	entries, err := audit.ListRotations(opts)
	if err != nil {
		return fmt.Errorf("failed to query history: %w", err)
	}

	if len(entries) == 0 {
		fmt.Fprintln(os.Stderr, "No rotation history found.")
		return nil
	}

	// Format output
	formatter := output.New(historyFormat, !noColor, true)
	fmt.Println(formatter.History(entries))

	return nil
}

// parseDate parses a date string in YYYY-MM-DD format
func parseDate(s string) (time.Time, error) {
	// Try common formats
	formats := []string{
		"2006-01-02",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z07:00",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("cannot parse %q (expected YYYY-MM-DD)", s)
}
