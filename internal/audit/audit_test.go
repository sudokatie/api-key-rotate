package audit

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitAndClose(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	err := Init(dbPath)
	require.NoError(t, err)
	assert.NotNil(t, db)

	err = Close()
	require.NoError(t, err)
}

func TestLogRotation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	err := Init(dbPath)
	require.NoError(t, err)
	defer Close()

	entry := &RotationEntry{
		KeyName:          "API_KEY",
		StartedAt:        time.Now().Add(-time.Minute),
		Status:           "success",
		LocationsUpdated: 2,
		LocationsFailed:  0,
		InitiatedBy:      "user",
		OldKeyPreview:    "sk-abc...",
		NewKeyPreview:    "sk-xyz...",
		Locations: []LocationEntry{
			{
				LocationType: "local",
				LocationPath: "/app/.env",
				Status:       "success",
				UpdatedAt:    time.Now(),
			},
		},
	}

	err = LogRotation(entry)
	require.NoError(t, err)
	assert.NotEmpty(t, entry.ID)
}

func TestListRotations(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	err := Init(dbPath)
	require.NoError(t, err)
	defer Close()

	// Log a few entries
	for i := 0; i < 3; i++ {
		entry := &RotationEntry{
			KeyName:   "API_KEY",
			StartedAt: time.Now().Add(-time.Duration(i) * time.Hour),
			Status:    "success",
		}
		err = LogRotation(entry)
		require.NoError(t, err)
	}

	// List all
	entries, err := ListRotations(QueryOptions{})
	require.NoError(t, err)
	assert.Len(t, entries, 3)

	// Filter by key
	entries, err = ListRotations(QueryOptions{KeyName: "API_KEY"})
	require.NoError(t, err)
	assert.Len(t, entries, 3)

	// Filter by non-existent key
	entries, err = ListRotations(QueryOptions{KeyName: "OTHER"})
	require.NoError(t, err)
	assert.Empty(t, entries)

	// Limit
	entries, err = ListRotations(QueryOptions{Limit: 2})
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestGetRotation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	err := Init(dbPath)
	require.NoError(t, err)
	defer Close()

	entry := &RotationEntry{
		KeyName:   "API_KEY",
		StartedAt: time.Now(),
		Status:    "success",
		Locations: []LocationEntry{
			{LocationType: "local", LocationPath: "/app/.env", Status: "success", UpdatedAt: time.Now()},
			{LocationType: "vercel", LocationPath: "proj/env1", Status: "success", UpdatedAt: time.Now()},
		},
	}
	err = LogRotation(entry)
	require.NoError(t, err)

	// Fetch it
	fetched, err := GetRotation(entry.ID)
	require.NoError(t, err)
	assert.Equal(t, entry.ID, fetched.ID)
	assert.Equal(t, "API_KEY", fetched.KeyName)
	assert.Len(t, fetched.Locations, 2)
}

func TestPurgeOldEntries(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	err := Init(dbPath)
	require.NoError(t, err)
	defer Close()

	// Log an old entry
	oldEntry := &RotationEntry{
		KeyName:   "OLD_KEY",
		StartedAt: time.Now().AddDate(0, 0, -100), // 100 days ago
		Status:    "success",
	}
	err = LogRotation(oldEntry)
	require.NoError(t, err)

	// Log a recent entry
	newEntry := &RotationEntry{
		KeyName:   "NEW_KEY",
		StartedAt: time.Now(),
		Status:    "success",
	}
	err = LogRotation(newEntry)
	require.NoError(t, err)

	// Purge entries older than 30 days
	err = PurgeOldEntries(30)
	require.NoError(t, err)

	// Should only have the new entry
	entries, err := ListRotations(QueryOptions{})
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "NEW_KEY", entries[0].KeyName)
}
