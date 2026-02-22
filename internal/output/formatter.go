package output

import (
	"github.com/sudokatie/api-key-rotate/internal/audit"
	"github.com/sudokatie/api-key-rotate/internal/providers"
	"github.com/sudokatie/api-key-rotate/internal/rotation"
)

// Formatter defines the interface for output formatting
type Formatter interface {
	Locations(locs []providers.Location) string
	DryRun(result *rotation.DryRunResult) string
	History(entries []audit.RotationEntry) string
	Result(success bool, msg string) string
}

// Format represents the output format type
type Format string

const (
	FormatText  Format = "text"
	FormatJSON  Format = "json"
	FormatTable Format = "table"
)

// New creates a formatter based on the format string
func New(format string, color bool, pretty bool) Formatter {
	switch Format(format) {
	case FormatJSON:
		return NewJSONFormatter(pretty)
	case FormatTable:
		return NewTableFormatter()
	default:
		return NewTextFormatter(color)
	}
}
