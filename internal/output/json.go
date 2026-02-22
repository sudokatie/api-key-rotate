package output

import (
	"encoding/json"

	"github.com/sudokatie/api-key-rotate/internal/audit"
	"github.com/sudokatie/api-key-rotate/internal/providers"
	"github.com/sudokatie/api-key-rotate/internal/rotation"
)

// JSONFormatter formats output as JSON
type JSONFormatter struct {
	pretty bool
}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter(pretty bool) *JSONFormatter {
	return &JSONFormatter{pretty: pretty}
}

// Locations formats locations as JSON
func (f *JSONFormatter) Locations(locs []providers.Location) string {
	type locationOutput struct {
		Type        string `json:"type"`
		Path        string `json:"path"`
		Project     string `json:"project,omitempty"`
		Environment string `json:"environment,omitempty"`
	}

	output := make([]locationOutput, 0, len(locs))
	for _, loc := range locs {
		output = append(output, locationOutput{
			Type:        loc.Type,
			Path:        loc.Path,
			Project:     loc.Project,
			Environment: loc.Environment,
		})
	}

	return f.marshal(output)
}

// DryRun formats a dry run result as JSON
func (f *JSONFormatter) DryRun(result *rotation.DryRunResult) string {
	type dryRunOutput struct {
		KeyName   string `json:"key_name"`
		Locations []struct {
			Type        string `json:"type"`
			Path        string `json:"path"`
			Project     string `json:"project,omitempty"`
			Environment string `json:"environment,omitempty"`
			CurrentMask string `json:"current_mask"`
		} `json:"locations"`
	}

	output := dryRunOutput{
		KeyName: result.KeyName,
	}

	for _, loc := range result.Locations {
		output.Locations = append(output.Locations, struct {
			Type        string `json:"type"`
			Path        string `json:"path"`
			Project     string `json:"project,omitempty"`
			Environment string `json:"environment,omitempty"`
			CurrentMask string `json:"current_mask"`
		}{
			Type:        loc.Type,
			Path:        loc.Path,
			Project:     loc.Project,
			Environment: loc.Environment,
			CurrentMask: loc.CurrentMask,
		})
	}

	return f.marshal(output)
}

// History formats audit history as JSON
func (f *JSONFormatter) History(entries []audit.RotationEntry) string {
	type historyOutput struct {
		ID               string `json:"id"`
		KeyName          string `json:"key_name"`
		StartedAt        string `json:"started_at"`
		CompletedAt      string `json:"completed_at"`
		Status           string `json:"status"`
		LocationsUpdated int    `json:"locations_updated"`
		LocationsFailed  int    `json:"locations_failed"`
		ErrorMessage     string `json:"error_message,omitempty"`
		InitiatedBy      string `json:"initiated_by,omitempty"`
	}

	output := make([]historyOutput, 0, len(entries))
	for _, entry := range entries {
		output = append(output, historyOutput{
			ID:               entry.ID,
			KeyName:          entry.KeyName,
			StartedAt:        entry.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
			CompletedAt:      entry.CompletedAt.Format("2006-01-02T15:04:05Z07:00"),
			Status:           entry.Status,
			LocationsUpdated: entry.LocationsUpdated,
			LocationsFailed:  entry.LocationsFailed,
			ErrorMessage:     entry.ErrorMessage,
			InitiatedBy:      entry.InitiatedBy,
		})
	}

	return f.marshal(output)
}

// Result formats a result as JSON
func (f *JSONFormatter) Result(success bool, msg string) string {
	output := struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}{
		Success: success,
		Message: msg,
	}

	return f.marshal(output)
}

func (f *JSONFormatter) marshal(v interface{}) string {
	var data []byte
	var err error

	if f.pretty {
		data, err = json.MarshalIndent(v, "", "  ")
	} else {
		data, err = json.Marshal(v)
	}

	if err != nil {
		return "{\"error\": \"failed to marshal JSON\"}"
	}

	return string(data)
}
