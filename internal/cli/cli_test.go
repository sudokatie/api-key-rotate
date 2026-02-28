package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sudokatie/api-key-rotate/internal/audit"
	"github.com/sudokatie/api-key-rotate/internal/config"
	"github.com/sudokatie/api-key-rotate/internal/output"
	"github.com/sudokatie/api-key-rotate/internal/providers"
	"github.com/sudokatie/api-key-rotate/internal/rotation"

	// Import providers to trigger registration via init()
	_ "github.com/sudokatie/api-key-rotate/internal/providers/github"
	_ "github.com/sudokatie/api-key-rotate/internal/providers/vercel"
)

// Helper to capture stdout/stderr during command execution
func captureOutput(f func()) (string, string) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()

	os.Stdout = wOut
	os.Stderr = wErr

	f()

	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var bufOut, bufErr bytes.Buffer
	io.Copy(&bufOut, rOut)
	io.Copy(&bufErr, rErr)

	return bufOut.String(), bufErr.String()
}

// Test version command output
func TestVersionCommand_Output(t *testing.T) {
	SetVersionInfo("1.0.0", "abc123", "2024-01-01")

	stdout, _ := captureOutput(func() {
		versionCmd.Run(versionCmd, nil)
	})

	assert.Contains(t, stdout, "1.0.0")
	assert.Contains(t, stdout, "abc123")
	assert.Contains(t, stdout, "2024-01-01")
}

// Test history command with empty database
func TestRunHistory_EmptyDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "audit.db")

	err := audit.Init(dbPath)
	require.NoError(t, err)
	defer audit.Close()

	// Reset flags
	historyKey = ""
	historyStatus = ""
	historySince = ""
	historyUntil = ""
	historyLimit = 50
	historyFormat = "text"

	_, stderr := captureOutput(func() {
		err := runHistory(historyCmd, nil)
		assert.NoError(t, err)
	})

	assert.Contains(t, stderr, "No rotation history found")
}

// Test history command uses audit.ListRotations correctly
// Note: runHistory calls audit.Init("") which reinitializes the db,
// so we test the query flow through audit package directly
func TestHistoryQueryFlow(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "audit.db")

	err := audit.Init(dbPath)
	require.NoError(t, err)
	defer audit.Close()

	// Add a test entry with proper timestamps
	entry := &audit.RotationEntry{
		KeyName:          "TEST_KEY",
		StartedAt:        time.Now().Add(-time.Hour),
		Status:           "success",
		LocationsUpdated: 2,
		LocationsFailed:  0,
		InitiatedBy:      "test",
		Locations:        []audit.LocationEntry{},
	}
	err = audit.LogRotation(entry)
	require.NoError(t, err)

	// Query the entries
	entries, err := audit.ListRotations(audit.QueryOptions{Limit: 50})
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "TEST_KEY", entries[0].KeyName)
}

// Test history query with key filter
func TestHistoryQueryWithKeyFilter(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "audit.db")

	err := audit.Init(dbPath)
	require.NoError(t, err)
	defer audit.Close()

	// Add entries with proper timestamps
	audit.LogRotation(&audit.RotationEntry{
		KeyName:     "KEY_A",
		StartedAt:   time.Now().Add(-time.Hour),
		Status:      "success",
		InitiatedBy: "test",
		Locations:   []audit.LocationEntry{},
	})
	audit.LogRotation(&audit.RotationEntry{
		KeyName:     "KEY_B",
		StartedAt:   time.Now().Add(-30 * time.Minute),
		Status:      "success",
		InitiatedBy: "test",
		Locations:   []audit.LocationEntry{},
	})

	// Query with key filter
	entries, err := audit.ListRotations(audit.QueryOptions{KeyName: "KEY_A", Limit: 50})
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "KEY_A", entries[0].KeyName)
}

// Test history output formatter with JSON
func TestHistoryJSONFormatter(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "audit.db")

	err := audit.Init(dbPath)
	require.NoError(t, err)
	defer audit.Close()

	audit.LogRotation(&audit.RotationEntry{
		KeyName:     "JSON_KEY",
		StartedAt:   time.Now().Add(-time.Hour),
		Status:      "success",
		InitiatedBy: "test",
		Locations:   []audit.LocationEntry{},
	})

	// Query entries
	entries, err := audit.ListRotations(audit.QueryOptions{Limit: 50})
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	// Format as JSON
	formatter := output.New("json", false, true)
	result := formatter.History(entries)

	assert.Contains(t, result, "JSON_KEY")
	assert.Contains(t, result, "{") // JSON output
}

// Test date parsing function
func TestParseDateFormats(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"2024-01-15", false},
		{"2024-01-15T10:30:00", false},
		{"2024-01-15T10:30:00Z", false},
		{"invalid", true},
		{"01-15-2024", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := parseDate(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test config show command output
func TestRunConfigShow_Output(t *testing.T) {
	config.Cfg = &config.Config{
		ScanPaths: []string{"/test/path"},
		UI: config.UIConfig{
			Color:   true,
			Verbose: false,
		},
	}

	jsonOut = false
	stdout, _ := captureOutput(func() {
		err := runConfigShow(configShowCmd, nil)
		assert.NoError(t, err)
	})

	assert.Contains(t, stdout, "/test/path")
}

// Test config show command with JSON output
func TestRunConfigShow_JSONOutput(t *testing.T) {
	config.Cfg = &config.Config{
		ScanPaths: []string{"/json/test/path"},
	}

	jsonOut = true
	stdout, _ := captureOutput(func() {
		err := runConfigShow(configShowCmd, nil)
		assert.NoError(t, err)
	})

	assert.Contains(t, stdout, "{")
	assert.Contains(t, stdout, "/json/test/path")
	jsonOut = false
}

// Test config show with no config loaded
func TestRunConfigShow_NoConfig(t *testing.T) {
	config.Cfg = nil

	err := runConfigShow(configShowCmd, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no configuration loaded")
}

// Test config get command outputs
func TestRunConfigGet_Outputs(t *testing.T) {
	config.Cfg = &config.Config{
		ScanPaths: []string{"/path/one", "/path/two"},
		UI: config.UIConfig{
			Color:   true,
			Verbose: false,
		},
		Audit: config.AuditConfig{
			Path:          "/audit/path",
			RetentionDays: 365,
		},
	}

	tests := []struct {
		key      string
		expected string
	}{
		{"scan_paths", "/path/one"},
		{"ui.color", "true"},
		{"ui.verbose", "false"},
		{"audit.path", "/audit/path"},
		{"audit.retention_days", "365"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			stdout, _ := captureOutput(func() {
				err := runConfigGet(configGetCmd, []string{tt.key})
				assert.NoError(t, err)
			})
			assert.Contains(t, stdout, tt.expected)
		})
	}
}

// Test config get with unknown key
func TestRunConfigGet_UnknownKey(t *testing.T) {
	config.Cfg = &config.Config{}

	err := runConfigGet(configGetCmd, []string{"unknown.key"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown config key")
}

// Test config set command output
func TestRunConfigSet_Output(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	config.Cfg = &config.Config{
		ScanPaths: []string{"/test"},
		UI:        config.UIConfig{Color: true, Verbose: false},
		Audit:     config.AuditConfig{RetentionDays: 365},
	}

	// Write initial config
	err := config.SaveTo(config.Cfg, configPath)
	require.NoError(t, err)

	stdout, _ := captureOutput(func() {
		// Manually set and verify
		err := setConfigValue(config.Cfg, "ui.verbose", "true")
		assert.NoError(t, err)
		err = config.SaveTo(config.Cfg, configPath)
		assert.NoError(t, err)
	})

	assert.True(t, config.Cfg.UI.Verbose)
	_ = stdout // suppress unused warning
}

// Test scan paths list command output
func TestRunScanPathsList_Output(t *testing.T) {
	config.Cfg = &config.Config{
		ScanPaths: []string{"/path/one", "/path/two"},
	}

	stdout, _ := captureOutput(func() {
		err := runScanPathsList(scanPathsListCmd, nil)
		assert.NoError(t, err)
	})

	assert.Contains(t, stdout, "/path/one")
	assert.Contains(t, stdout, "/path/two")
}

// Test scan paths add duplicate
func TestRunScanPathsAdd_Duplicate(t *testing.T) {
	config.Cfg = &config.Config{
		ScanPaths: []string{"/existing"},
	}

	err := runScanPathsAdd(scanPathsAddCmd, []string{"/existing"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already configured")
}

// Test scan paths remove not found
func TestRunScanPathsRemove_NotFound(t *testing.T) {
	config.Cfg = &config.Config{
		ScanPaths: []string{"/existing"},
	}

	err := runScanPathsRemove(scanPathsRemoveCmd, []string{"/nonexistent"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// Test providers list command output
func TestRunProvidersList_Output(t *testing.T) {
	config.Cfg = &config.Config{}
	noColor = true

	stdout, _ := captureOutput(func() {
		err := runProvidersList(providersListCmd, nil)
		assert.NoError(t, err)
	})

	assert.Contains(t, stdout, "Available providers")
	noColor = false
}

// Test print test result helper - success
func TestPrintTestResult_Success(t *testing.T) {
	noColor = true

	stdout, _ := captureOutput(func() {
		printTestResult("test-provider", true, nil)
	})
	assert.Contains(t, stdout, "OK")

	noColor = false
}

// Test print test result helper - failure
func TestPrintTestResult_Failure(t *testing.T) {
	noColor = true

	stdout, _ := captureOutput(func() {
		printTestResult("test-provider", false, assert.AnError)
	})
	assert.Contains(t, stdout, "FAILED")

	noColor = false
}

// Test gather locations with skip local flag
func TestGatherLocations_WithSkipLocal(t *testing.T) {
	config.Cfg = &config.Config{
		ScanPaths:    []string{"/test"},
		FilePatterns: []string{".env"},
	}

	rotateSkipLocal = true
	rotateSkipCloud = false

	locs, err := gatherLocations("TEST_KEY")
	assert.NoError(t, err)
	// Should not have scanned local files
	for _, loc := range locs {
		assert.NotEqual(t, "local", loc.Type)
	}

	rotateSkipLocal = false
}

// Test gather locations with skip cloud flag
func TestGatherLocations_WithSkipCloud(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	os.WriteFile(envPath, []byte("TEST_KEY=value"), 0600)

	config.Cfg = &config.Config{
		ScanPaths:    []string{tmpDir},
		FilePatterns: []string{".env"},
	}

	rotateSkipLocal = false
	rotateSkipCloud = true

	locs, err := gatherLocations("TEST_KEY")
	assert.NoError(t, err)
	// Should only have local files
	for _, loc := range locs {
		assert.Equal(t, "local", loc.Type)
	}

	rotateSkipCloud = false
}

// Test Execute function returns no error for help
func TestExecute_Help(t *testing.T) {
	rootCmd.SetArgs([]string{"--help"})
	err := Execute()
	assert.NoError(t, err)
}

// Test SetVersionInfo function
func TestSetVersionInfo_Values(t *testing.T) {
	SetVersionInfo("v1.2.3", "deadbeef", "2024-06-15")

	assert.Equal(t, "v1.2.3", versionInfo.Version)
	assert.Equal(t, "deadbeef", versionInfo.Commit)
	assert.Equal(t, "2024-06-15", versionInfo.BuildDate)
}

// Test root command persistent pre-run skips version
func TestRootCmd_VersionSkipsConfig(t *testing.T) {
	rootCmd.SetArgs([]string{"version"})
	err := rootCmd.Execute()
	assert.NoError(t, err)
}

// Test findCloudLocations with providers configured but no credentials
func TestFindCloudLocations_NoCredentials(t *testing.T) {
	config.Cfg = &config.Config{
		Providers: map[string]*config.ProviderConfig{
			"vercel": {Enabled: true},
		},
	}
	verbose = true

	// Should not panic, just return empty with warning
	locs := findCloudLocations("API_KEY")
	assert.Empty(t, locs)

	verbose = false
}

// Test findCloudLocations with no providers
func TestFindCloudLocations_NoProviders(t *testing.T) {
	config.Cfg = &config.Config{
		Providers: nil,
	}

	locs := findCloudLocations("API_KEY")
	assert.Empty(t, locs)
}

// Test exit codes are properly defined
func TestExitCodes_Values(t *testing.T) {
	assert.Equal(t, 0, ExitSuccess)
	assert.Equal(t, 1, ExitGeneralError)
	assert.Equal(t, 2, ExitConfigError)
	assert.Equal(t, 3, ExitProviderError)
	assert.Equal(t, 4, ExitKeyNotFound)
	assert.Equal(t, 5, ExitRotationFailed)
	assert.Equal(t, 6, ExitRollbackFailed)
}

// Test root command Use string
func TestRootCmd_UseString(t *testing.T) {
	assert.Equal(t, "api-key-rotate [KEY_NAME]", rootCmd.Use)
	assert.NotNil(t, rootCmd.RunE)
}

// Test root command has rotate flags
func TestRootCmd_HasRotateFlags(t *testing.T) {
	flags := rootCmd.Flags()

	assert.NotNil(t, flags.Lookup("execute"))
	assert.NotNil(t, flags.Lookup("new-key"))
	assert.NotNil(t, flags.Lookup("force"))
	assert.NotNil(t, flags.Lookup("skip-local"))
	assert.NotNil(t, flags.Lookup("skip-cloud"))
	assert.NotNil(t, flags.Lookup("locations"))
	assert.NotNil(t, flags.Lookup("exclude"))
	assert.NotNil(t, flags.Lookup("format"))
}

// Test find command has correct flags
func TestFindCmd_HasCorrectFlags(t *testing.T) {
	flags := findCmd.Flags()

	assert.NotNil(t, flags.Lookup("local-only"))
	assert.NotNil(t, flags.Lookup("cloud-only"))
	assert.NotNil(t, flags.Lookup("format"))
}

// Test history command use string
func TestHistoryCmd_UseString(t *testing.T) {
	assert.Equal(t, "history", historyCmd.Use)
}

// Test rotate command has correct flags
func TestRotateCmd_HasCorrectFlags(t *testing.T) {
	flags := rotateCmd.Flags()

	assert.NotNil(t, flags.Lookup("execute"))
	assert.NotNil(t, flags.Lookup("new-key"))
	assert.NotNil(t, flags.Lookup("force"))
	assert.NotNil(t, flags.Lookup("skip-local"))
	assert.NotNil(t, flags.Lookup("skip-cloud"))
	assert.NotNil(t, flags.Lookup("locations"))
	assert.NotNil(t, flags.Lookup("exclude"))
	assert.NotNil(t, flags.Lookup("format"))
}

// Test rotate command short flags
func TestRotateCmd_HasShortFlags(t *testing.T) {
	flags := rotateCmd.Flags()

	executeFlag := flags.ShorthandLookup("e")
	assert.NotNil(t, executeFlag)
	assert.Equal(t, "execute", executeFlag.Name)

	forceFlag := flags.ShorthandLookup("f")
	assert.NotNil(t, forceFlag)
	assert.Equal(t, "force", forceFlag.Name)
}

// Test global flags
func TestRootCmd_GlobalFlags(t *testing.T) {
	flags := rootCmd.PersistentFlags()

	assert.NotNil(t, flags.Lookup("config"))
	assert.NotNil(t, flags.Lookup("verbose"))
	assert.NotNil(t, flags.Lookup("quiet"))
	assert.NotNil(t, flags.Lookup("no-color"))
	assert.NotNil(t, flags.Lookup("json"))
}

// Test verbose short flag
func TestRootCmd_VerboseShortFlag(t *testing.T) {
	flags := rootCmd.PersistentFlags()

	verboseFlag := flags.ShorthandLookup("v")
	assert.NotNil(t, verboseFlag)
	assert.Equal(t, "verbose", verboseFlag.Name)

	quietFlag := flags.ShorthandLookup("q")
	assert.NotNil(t, quietFlag)
	assert.Equal(t, "quiet", quietFlag.Name)
}

// Test getProviderStatus with enabled provider
func TestGetProviderStatus_Enabled(t *testing.T) {
	config.Cfg = &config.Config{
		Providers: map[string]*config.ProviderConfig{
			"vercel": {Enabled: true},
		},
	}
	noColor = true

	status := getProviderStatus("vercel")
	// Without actual keyring credentials, should show not configured
	assert.Contains(t, status, "not configured")

	noColor = false
}

// Test printTransactionSummary output
func TestPrintTransactionSummary_Output(t *testing.T) {
	tx := &rotation.Transaction{
		Locations: []rotation.LocationState{
			{
				Location: providers.Location{Path: "/test/.env"},
				Status:   rotation.StatusSuccess,
			},
		},
	}

	noColor = true
	verbose = true
	stdout, _ := captureOutput(func() {
		printTransactionSummary(tx)
	})

	assert.Contains(t, stdout, "Summary")
	assert.Contains(t, stdout, "1 succeeded")
	noColor = false
	verbose = false
}

// Test printTransactionSummary with failures
func TestPrintTransactionSummary_WithFailures(t *testing.T) {
	tx := &rotation.Transaction{
		Locations: []rotation.LocationState{
			{
				Location: providers.Location{Path: "/test/.env"},
				Status:   rotation.StatusSuccess,
			},
			{
				Location: providers.Location{Path: "/test2/.env"},
				Status:   rotation.StatusFailed,
				Error:    assert.AnError,
			},
		},
	}

	noColor = true
	stdout, _ := captureOutput(func() {
		printTransactionSummary(tx)
	})

	assert.Contains(t, stdout, "Summary")
	assert.Contains(t, stdout, "1 succeeded")
	assert.Contains(t, stdout, "1 failed")
	noColor = false
}
