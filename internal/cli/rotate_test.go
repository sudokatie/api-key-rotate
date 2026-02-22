package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sudokatie/api-key-rotate/internal/providers"
)

func TestFilterLocations_NoFilters(t *testing.T) {
	// Reset global flags
	rotateLocations = nil
	rotateExclude = nil

	locs := []providers.Location{
		{Type: "local", Path: "/project/a/.env"},
		{Type: "local", Path: "/project/b/.env"},
		{Type: "vercel", Provider: "vercel", Project: "my-app", Environment: "production"},
	}

	result := filterLocations(locs)
	assert.Len(t, result, 3)
}

func TestFilterLocations_IncludeFilter(t *testing.T) {
	rotateLocations = []string{"project/a"}
	rotateExclude = nil

	locs := []providers.Location{
		{Type: "local", Path: "/project/a/.env"},
		{Type: "local", Path: "/project/b/.env"},
	}

	result := filterLocations(locs)
	assert.Len(t, result, 1)
	assert.Equal(t, "/project/a/.env", result[0].Path)

	// Reset
	rotateLocations = nil
}

func TestFilterLocations_ExcludeFilter(t *testing.T) {
	rotateLocations = nil
	rotateExclude = []string{"project/b"}

	locs := []providers.Location{
		{Type: "local", Path: "/project/a/.env"},
		{Type: "local", Path: "/project/b/.env"},
	}

	result := filterLocations(locs)
	assert.Len(t, result, 1)
	assert.Equal(t, "/project/a/.env", result[0].Path)

	// Reset
	rotateExclude = nil
}

func TestFilterLocations_BothFilters(t *testing.T) {
	rotateLocations = []string{"project"}
	rotateExclude = []string{"staging"}

	locs := []providers.Location{
		{Type: "local", Path: "/project/prod/.env"},
		{Type: "local", Path: "/project/staging/.env"},
		{Type: "local", Path: "/other/prod/.env"},
	}

	result := filterLocations(locs)
	assert.Len(t, result, 1)
	assert.Equal(t, "/project/prod/.env", result[0].Path)

	// Reset
	rotateLocations = nil
	rotateExclude = nil
}

func TestFilterLocations_CloudProviders(t *testing.T) {
	rotateLocations = []string{"my-app"}
	rotateExclude = nil

	locs := []providers.Location{
		{Type: "vercel", Provider: "vercel", Project: "my-app", Environment: "production"},
		{Type: "vercel", Provider: "vercel", Project: "other-app", Environment: "production"},
		{Type: "github", Provider: "github", Project: "my-app", Environment: ""},
	}

	result := filterLocations(locs)
	assert.Len(t, result, 2) // my-app on vercel and github

	// Reset
	rotateLocations = nil
}

func TestLocationPath_Local(t *testing.T) {
	loc := providers.Location{Type: "local", Path: "/home/user/project/.env"}
	assert.Equal(t, "/home/user/project/.env", locationPath(loc))
}

func TestLocationPath_CloudWithProject(t *testing.T) {
	loc := providers.Location{
		Type:        "vercel",
		Provider:    "vercel",
		Project:     "my-app",
		Environment: "production",
	}
	assert.Equal(t, "vercel/my-app/production", locationPath(loc))
}

func TestLocationPath_CloudWithoutProject(t *testing.T) {
	loc := providers.Location{
		Type:     "github",
		Provider: "github",
		Path:     "owner/repo",
	}
	assert.Equal(t, "github/owner/repo", locationPath(loc))
}

func TestRotateCmd_Help(t *testing.T) {
	// Test that the command is properly configured
	assert.Equal(t, "rotate <KEY_NAME>", rotateCmd.Use)
	assert.NotEmpty(t, rotateCmd.Short)
	assert.NotEmpty(t, rotateCmd.Long)
}

func TestRotateCmd_Flags(t *testing.T) {
	// Test that all flags are registered
	flags := rotateCmd.Flags()

	assert.NotNil(t, flags.Lookup("execute"))
	assert.NotNil(t, flags.Lookup("new-key"))
	assert.NotNil(t, flags.Lookup("force"))
	assert.NotNil(t, flags.Lookup("local-only"))
	assert.NotNil(t, flags.Lookup("cloud-only"))
	assert.NotNil(t, flags.Lookup("locations"))
	assert.NotNil(t, flags.Lookup("exclude"))
	assert.NotNil(t, flags.Lookup("format"))
}

func TestRotateCmd_ShortFlags(t *testing.T) {
	flags := rotateCmd.Flags()

	// Check short flags
	executeFlag := flags.ShorthandLookup("e")
	assert.NotNil(t, executeFlag)
	assert.Equal(t, "execute", executeFlag.Name)

	forceFlag := flags.ShorthandLookup("f")
	assert.NotNil(t, forceFlag)
	assert.Equal(t, "force", forceFlag.Name)
}
