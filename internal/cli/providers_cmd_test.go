package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatProviderName(t *testing.T) {
	result := formatProviderName("vercel")
	assert.Equal(t, "vercel    ", result)
	assert.Len(t, result, 10)
}

func TestFormatProviderName_Long(t *testing.T) {
	result := formatProviderName("veryfreakinglongname")
	assert.Contains(t, result, "veryfreakinglongname")
}

func TestProvidersCmd_Subcommands(t *testing.T) {
	subcommands := make(map[string]bool)
	for _, cmd := range providersCmd.Commands() {
		subcommands[cmd.Use] = true
	}

	assert.True(t, subcommands["list"])
	assert.True(t, subcommands["add <provider>"])
	assert.True(t, subcommands["remove <provider>"])
	assert.True(t, subcommands["test [provider]"])
}

func TestProvidersListCmd(t *testing.T) {
	assert.Equal(t, "list", providersListCmd.Use)
	assert.NotEmpty(t, providersListCmd.Short)
}

func TestProvidersAddCmd(t *testing.T) {
	assert.Equal(t, "add <provider>", providersAddCmd.Use)
	assert.NotEmpty(t, providersAddCmd.Short)
	assert.NotEmpty(t, providersAddCmd.Long)
}

func TestProvidersRemoveCmd(t *testing.T) {
	assert.Equal(t, "remove <provider>", providersRemoveCmd.Use)
	assert.NotEmpty(t, providersRemoveCmd.Short)
}

func TestProvidersTestCmd(t *testing.T) {
	assert.Equal(t, "test [provider]", providersTestCmd.Use)
	assert.NotEmpty(t, providersTestCmd.Short)
}

func TestGetProviderStatus_NoColor(t *testing.T) {
	// Set noColor to true for predictable output
	oldNoColor := noColor
	noColor = true
	defer func() { noColor = oldNoColor }()

	// Without token, should show not configured
	status := getProviderStatus("nonexistent-provider")
	assert.Equal(t, "[not configured]", status)
}
