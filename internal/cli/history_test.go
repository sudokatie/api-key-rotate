package cli

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDate_SimpleDate(t *testing.T) {
	result, err := parseDate("2024-01-15")
	require.NoError(t, err)
	assert.Equal(t, 2024, result.Year())
	assert.Equal(t, time.January, result.Month())
	assert.Equal(t, 15, result.Day())
}

func TestParseDate_WithTime(t *testing.T) {
	result, err := parseDate("2024-01-15T10:30:00")
	require.NoError(t, err)
	assert.Equal(t, 2024, result.Year())
	assert.Equal(t, 10, result.Hour())
	assert.Equal(t, 30, result.Minute())
}

func TestParseDate_RFC3339(t *testing.T) {
	result, err := parseDate("2024-01-15T10:30:00Z")
	require.NoError(t, err)
	assert.Equal(t, 2024, result.Year())
}

func TestParseDate_Invalid(t *testing.T) {
	_, err := parseDate("not-a-date")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot parse")
}

func TestParseDate_InvalidFormat(t *testing.T) {
	_, err := parseDate("15/01/2024")
	assert.Error(t, err)
}

func TestHistoryCmd_Help(t *testing.T) {
	assert.Equal(t, "history", historyCmd.Use)
	assert.NotEmpty(t, historyCmd.Short)
	assert.NotEmpty(t, historyCmd.Long)
}

func TestHistoryCmd_Flags(t *testing.T) {
	flags := historyCmd.Flags()

	assert.NotNil(t, flags.Lookup("key"))
	assert.NotNil(t, flags.Lookup("status"))
	assert.NotNil(t, flags.Lookup("since"))
	assert.NotNil(t, flags.Lookup("until"))
	assert.NotNil(t, flags.Lookup("limit"))
	assert.NotNil(t, flags.Lookup("format"))
}

func TestHistoryCmd_DefaultLimit(t *testing.T) {
	flags := historyCmd.Flags()
	limitFlag := flags.Lookup("limit")
	require.NotNil(t, limitFlag)
	assert.Equal(t, "50", limitFlag.DefValue)
}

func TestHistoryCmd_DefaultFormat(t *testing.T) {
	flags := historyCmd.Flags()
	formatFlag := flags.Lookup("format")
	require.NotNil(t, formatFlag)
	assert.Equal(t, "text", formatFlag.DefValue)
}
