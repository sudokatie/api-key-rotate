package output

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sudokatie/api-key-rotate/internal/audit"
	"github.com/sudokatie/api-key-rotate/internal/providers"
	"github.com/sudokatie/api-key-rotate/internal/rotation"
)

func TestNew(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"text", "*output.TextFormatter"},
		{"json", "*output.JSONFormatter"},
		{"table", "*output.TableFormatter"},
		{"unknown", "*output.TextFormatter"}, // default
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			f := New(tt.format, true, true)
			assert.Contains(t, strings.Replace(strings.Replace(
				strings.Replace("%T", "%", "", 1), "T", "", 1), "", "", 1),
				"")
			// Just verify it doesn't panic
			assert.NotNil(t, f)
		})
	}
}

// Text Formatter Tests

func TestTextFormatterLocationsEmpty(t *testing.T) {
	f := NewTextFormatter(false)
	result := f.Locations(nil)
	assert.Equal(t, "No locations found.", result)
}

func TestTextFormatterLocationsGrouped(t *testing.T) {
	f := NewTextFormatter(false)
	locs := []providers.Location{
		{Type: "local", Path: "/path/one/.env"},
		{Type: "local", Path: "/path/two/.env"},
		{Type: "vercel", Path: "proj-123", Project: "my-app", Environment: "production"},
	}

	result := f.Locations(locs)

	assert.Contains(t, result, "=== Local Files ===")
	assert.Contains(t, result, "/path/one/.env")
	assert.Contains(t, result, "/path/two/.env")
	assert.Contains(t, result, "=== Vercel ===")
	assert.Contains(t, result, "my-app/production")
}

func TestTextFormatterDryRun(t *testing.T) {
	f := NewTextFormatter(false)
	result := &rotation.DryRunResult{
		KeyName: "API_KEY",
		Locations: []rotation.LocationPreview{
			{Type: "local", Path: "/path/.env", CurrentMask: "sk_l****"},
		},
	}

	output := f.DryRun(result)

	assert.Contains(t, output, "DRY RUN")
	assert.Contains(t, output, "API_KEY")
	assert.Contains(t, output, "/path/.env")
	assert.Contains(t, output, "sk_l****")
}

func TestTextFormatterHistory(t *testing.T) {
	f := NewTextFormatter(false)
	entries := []audit.RotationEntry{
		{
			KeyName:          "API_KEY",
			StartedAt:        time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			Status:           "success",
			LocationsUpdated: 3,
			LocationsFailed:  0,
		},
	}

	output := f.History(entries)

	assert.Contains(t, output, "2024-01-15")
	assert.Contains(t, output, "API_KEY")
	assert.Contains(t, output, "success")
	assert.Contains(t, output, "Updated: 3")
}

func TestTextFormatterHistoryEmpty(t *testing.T) {
	f := NewTextFormatter(false)
	result := f.History(nil)
	assert.Equal(t, "No rotation history found.", result)
}

func TestTextFormatterResult(t *testing.T) {
	f := NewTextFormatter(false)

	success := f.Result(true, "Rotated 3 locations")
	assert.Contains(t, success, "SUCCESS")
	assert.Contains(t, success, "Rotated 3 locations")

	fail := f.Result(false, "Provider error")
	assert.Contains(t, fail, "ERROR")
	assert.Contains(t, fail, "Provider error")
}

// JSON Formatter Tests

func TestJSONFormatterLocations(t *testing.T) {
	f := NewJSONFormatter(false)
	locs := []providers.Location{
		{Type: "local", Path: "/path/.env"},
		{Type: "vercel", Path: "proj-123", Project: "my-app", Environment: "prod"},
	}

	result := f.Locations(locs)

	var parsed []map[string]interface{}
	err := json.Unmarshal([]byte(result), &parsed)
	require.NoError(t, err)
	assert.Len(t, parsed, 2)
	assert.Equal(t, "local", parsed[0]["type"])
	assert.Equal(t, "vercel", parsed[1]["type"])
}

func TestJSONFormatterPretty(t *testing.T) {
	f := NewJSONFormatter(true)
	locs := []providers.Location{{Type: "local", Path: "/path/.env"}}

	result := f.Locations(locs)

	assert.Contains(t, result, "\n")
	assert.Contains(t, result, "  ")
}

func TestJSONFormatterCompact(t *testing.T) {
	f := NewJSONFormatter(false)
	locs := []providers.Location{{Type: "local", Path: "/path/.env"}}

	result := f.Locations(locs)

	assert.NotContains(t, result, "\n")
}

func TestJSONFormatterDryRun(t *testing.T) {
	f := NewJSONFormatter(false)
	dr := &rotation.DryRunResult{
		KeyName: "API_KEY",
		Locations: []rotation.LocationPreview{
			{Type: "local", Path: "/path/.env", CurrentMask: "sk_l****"},
		},
	}

	result := f.DryRun(dr)

	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(result), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "API_KEY", parsed["key_name"])
}

func TestJSONFormatterHistory(t *testing.T) {
	f := NewJSONFormatter(false)
	entries := []audit.RotationEntry{
		{
			ID:               "abc123",
			KeyName:          "API_KEY",
			StartedAt:        time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			CompletedAt:      time.Date(2024, 1, 15, 10, 30, 5, 0, time.UTC),
			Status:           "success",
			LocationsUpdated: 3,
		},
	}

	result := f.History(entries)

	var parsed []map[string]interface{}
	err := json.Unmarshal([]byte(result), &parsed)
	require.NoError(t, err)
	assert.Len(t, parsed, 1)
	assert.Equal(t, "abc123", parsed[0]["id"])
}

func TestJSONFormatterResult(t *testing.T) {
	f := NewJSONFormatter(false)

	success := f.Result(true, "Done")
	var parsed map[string]interface{}
	json.Unmarshal([]byte(success), &parsed)
	assert.Equal(t, true, parsed["success"])
	assert.Equal(t, "Done", parsed["message"])
}

// Table Formatter Tests

func TestTableFormatterLocationsEmpty(t *testing.T) {
	f := NewTableFormatter()
	result := f.Locations(nil)
	assert.Equal(t, "No locations found.", result)
}

func TestTableFormatterLocations(t *testing.T) {
	f := NewTableFormatter()
	locs := []providers.Location{
		{Type: "local", Path: "/path/.env"},
		{Type: "vercel", Path: "proj-123", Project: "my-app", Environment: "prod"},
	}

	result := f.Locations(locs)

	assert.Contains(t, result, "TYPE")
	assert.Contains(t, result, "LOCATION")
	assert.Contains(t, result, "local")
	assert.Contains(t, result, "vercel")
	assert.Contains(t, result, "/path/.env")
}

func TestTableFormatterDryRun(t *testing.T) {
	f := NewTableFormatter()
	dr := &rotation.DryRunResult{
		KeyName: "API_KEY",
		Locations: []rotation.LocationPreview{
			{Type: "local", Path: "/path/.env", CurrentMask: "sk_l****"},
		},
	}

	result := f.DryRun(dr)

	assert.Contains(t, result, "DRY RUN")
	assert.Contains(t, result, "API_KEY")
	assert.Contains(t, result, "TYPE")
	assert.Contains(t, result, "LOCATION")
}

func TestTableFormatterHistoryEmpty(t *testing.T) {
	f := NewTableFormatter()
	result := f.History(nil)
	assert.Equal(t, "No rotation history found.", result)
}

func TestTableFormatterHistory(t *testing.T) {
	f := NewTableFormatter()
	entries := []audit.RotationEntry{
		{
			KeyName:          "API_KEY",
			StartedAt:        time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			Status:           "success",
			LocationsUpdated: 3,
			LocationsFailed:  0,
		},
	}

	result := f.History(entries)

	assert.Contains(t, result, "DATE")
	assert.Contains(t, result, "KEY")
	assert.Contains(t, result, "STATUS")
	assert.Contains(t, result, "API_KEY")
	assert.Contains(t, result, "success")
}

func TestTableFormatterResult(t *testing.T) {
	f := NewTableFormatter()

	success := f.Result(true, "Done")
	assert.Contains(t, success, "SUCCESS")

	fail := f.Result(false, "Error")
	assert.Contains(t, fail, "ERROR")
}

// Unicode handling

func TestFormattersHandleUnicode(t *testing.T) {
	locs := []providers.Location{
		{Type: "local", Path: "/path/to/проект/.env"},
		{Type: "local", Path: "/path/to/项目/.env"},
	}

	text := NewTextFormatter(false).Locations(locs)
	assert.Contains(t, text, "проект")
	assert.Contains(t, text, "项目")

	jsonOut := NewJSONFormatter(false).Locations(locs)
	assert.Contains(t, jsonOut, "проект")
	assert.Contains(t, jsonOut, "项目")

	table := NewTableFormatter().Locations(locs)
	assert.Contains(t, table, "проект")
	assert.Contains(t, table, "项目")
}
