package output

import (
	"fmt"
	"strings"

	"github.com/olekukonko/tablewriter"

	"github.com/sudokatie/api-key-rotate/internal/audit"
	"github.com/sudokatie/api-key-rotate/internal/providers"
	"github.com/sudokatie/api-key-rotate/internal/rotation"
)

// TableFormatter formats output as ASCII tables
type TableFormatter struct{}

// NewTableFormatter creates a new table formatter
func NewTableFormatter() *TableFormatter {
	return &TableFormatter{}
}

// Locations formats locations as a table
func (f *TableFormatter) Locations(locs []providers.Location) string {
	if len(locs) == 0 {
		return "No locations found."
	}

	var sb strings.Builder
	table := tablewriter.NewWriter(&sb)
	table.SetHeader([]string{"Type", "Location", "Project", "Environment"})
	table.SetBorder(false)
	table.SetColumnSeparator(" ")
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for _, loc := range locs {
		location := loc.Path
		if loc.Type != "local" && loc.Project != "" {
			location = loc.Project
		}

		table.Append([]string{
			loc.Type,
			location,
			loc.Project,
			loc.Environment,
		})
	}

	table.Render()
	return strings.TrimSpace(sb.String())
}

// DryRun formats a dry run result as a table
func (f *TableFormatter) DryRun(result *rotation.DryRunResult) string {
	var sb strings.Builder

	sb.WriteString("DRY RUN - No changes will be made\n\n")
	sb.WriteString("Key: " + result.KeyName + "\n\n")

	table := tablewriter.NewWriter(&sb)
	table.SetHeader([]string{"#", "Type", "Location", "Current"})
	table.SetBorder(false)
	table.SetColumnSeparator(" ")
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for i, loc := range result.Locations {
		location := loc.Path
		if loc.Project != "" {
			location = loc.Project + "/" + loc.Environment
		}

		table.Append([]string{
			fmt.Sprintf("%d", i+1),
			loc.Type,
			location,
			loc.CurrentMask,
		})
	}

	table.Render()
	return sb.String()
}

// History formats audit history as a table
func (f *TableFormatter) History(entries []audit.RotationEntry) string {
	if len(entries) == 0 {
		return "No rotation history found."
	}

	var sb strings.Builder
	table := tablewriter.NewWriter(&sb)
	table.SetHeader([]string{"Date", "Key", "Status", "Updated", "Failed", "Error"})
	table.SetBorder(false)
	table.SetColumnSeparator(" ")
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for _, entry := range entries {
		errMsg := entry.ErrorMessage
		if len(errMsg) > 30 {
			errMsg = errMsg[:27] + "..."
		}

		table.Append([]string{
			entry.StartedAt.Format("2006-01-02 15:04"),
			entry.KeyName,
			entry.Status,
			itoa(entry.LocationsUpdated),
			itoa(entry.LocationsFailed),
			errMsg,
		})
	}

	table.Render()
	return strings.TrimSpace(sb.String())
}

// Result formats a result - tables don't make sense here, use text
func (f *TableFormatter) Result(success bool, msg string) string {
	if success {
		return "SUCCESS: " + msg
	}
	return "ERROR: " + msg
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
