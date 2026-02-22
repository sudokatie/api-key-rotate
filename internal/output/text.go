package output

import (
	"fmt"
	"strings"

	"github.com/fatih/color"

	"github.com/sudokatie/api-key-rotate/internal/audit"
	"github.com/sudokatie/api-key-rotate/internal/providers"
	"github.com/sudokatie/api-key-rotate/internal/rotation"
)

// TextFormatter formats output as human-readable text
type TextFormatter struct {
	color bool
}

// NewTextFormatter creates a new text formatter
func NewTextFormatter(useColor bool) *TextFormatter {
	if !useColor {
		color.NoColor = true
	}
	return &TextFormatter{color: useColor}
}

// Locations formats a list of locations grouped by type
func (f *TextFormatter) Locations(locs []providers.Location) string {
	if len(locs) == 0 {
		return "No locations found."
	}

	var sb strings.Builder

	// Group by type
	groups := make(map[string][]providers.Location)
	for _, loc := range locs {
		groups[loc.Type] = append(groups[loc.Type], loc)
	}

	// Order: local, then cloud providers
	typeOrder := []string{"local", "vercel", "github"}
	for _, t := range typeOrder {
		if locations, ok := groups[t]; ok {
			f.writeLocationGroup(&sb, t, locations)
		}
	}

	// Any other types
	for t, locations := range groups {
		if t != "local" && t != "vercel" && t != "github" {
			f.writeLocationGroup(&sb, t, locations)
		}
	}

	return strings.TrimSpace(sb.String())
}

func (f *TextFormatter) writeLocationGroup(sb *strings.Builder, locType string, locs []providers.Location) {
	header := f.formatHeader(locType)
	sb.WriteString(header)
	sb.WriteString("\n")

	for _, loc := range locs {
		sb.WriteString("  ")
		if locType == "local" {
			sb.WriteString(loc.Path)
		} else {
			// Cloud: project/environment
			if loc.Project != "" {
				sb.WriteString(loc.Project)
				if loc.Environment != "" {
					sb.WriteString("/")
					sb.WriteString(loc.Environment)
				}
			} else {
				sb.WriteString(loc.Path)
			}
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
}

func (f *TextFormatter) formatHeader(locType string) string {
	headers := map[string]string{
		"local":  "Local Files",
		"vercel": "Vercel",
		"github": "GitHub Actions",
	}
	name := headers[locType]
	if name == "" {
		name = strings.Title(locType)
	}

	if f.color {
		return color.CyanString("=== %s ===", name)
	}
	return fmt.Sprintf("=== %s ===", name)
}

// DryRun formats a dry run result
func (f *TextFormatter) DryRun(result *rotation.DryRunResult) string {
	var sb strings.Builder

	header := "DRY RUN - No changes will be made"
	if f.color {
		sb.WriteString(color.YellowString(header))
	} else {
		sb.WriteString(header)
	}
	sb.WriteString("\n\n")

	sb.WriteString(fmt.Sprintf("Key: %s\n", result.KeyName))
	sb.WriteString(fmt.Sprintf("Locations to update: %d\n\n", len(result.Locations)))

	for i, loc := range result.Locations {
		sb.WriteString(fmt.Sprintf("%d. [%s] ", i+1, loc.Type))
		if loc.Project != "" {
			sb.WriteString(fmt.Sprintf("%s/%s", loc.Project, loc.Environment))
		} else {
			sb.WriteString(loc.Path)
		}
		sb.WriteString(fmt.Sprintf(" (current: %s)\n", loc.CurrentMask))
	}

	return sb.String()
}

// History formats audit history entries
func (f *TextFormatter) History(entries []audit.RotationEntry) string {
	if len(entries) == 0 {
		return "No rotation history found."
	}

	var sb strings.Builder

	for _, entry := range entries {
		// Header line with timestamp and key
		timestamp := entry.StartedAt.Format("2006-01-02 15:04:05")
		status := f.formatStatus(entry.Status)

		sb.WriteString(fmt.Sprintf("[%s] %s - %s\n", timestamp, entry.KeyName, status))

		// Details
		sb.WriteString(fmt.Sprintf("  Updated: %d, Failed: %d\n", 
			entry.LocationsUpdated, entry.LocationsFailed))

		if entry.ErrorMessage != "" {
			sb.WriteString(fmt.Sprintf("  Error: %s\n", entry.ErrorMessage))
		}

		sb.WriteString("\n")
	}

	return strings.TrimSpace(sb.String())
}

func (f *TextFormatter) formatStatus(status string) string {
	if !f.color {
		return status
	}

	switch status {
	case "success":
		return color.GreenString(status)
	case "failed":
		return color.RedString(status)
	case "partial":
		return color.YellowString(status)
	default:
		return status
	}
}

// Result formats a success or error result
func (f *TextFormatter) Result(success bool, msg string) string {
	if !f.color {
		if success {
			return "SUCCESS: " + msg
		}
		return "ERROR: " + msg
	}

	if success {
		return color.GreenString("SUCCESS: ") + msg
	}
	return color.RedString("ERROR: ") + msg
}
