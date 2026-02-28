package audit

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListRotationsWithStatusFilter(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	err := Init(dbPath)
	require.NoError(t, err)
	defer Close()

	// Log entries with different statuses
	entries := []struct {
		keyName string
		status  string
	}{
		{"KEY1", "success"},
		{"KEY2", "failed"},
		{"KEY3", "success"},
		{"KEY4", "partial"},
	}

	for _, e := range entries {
		entry := &RotationEntry{
			KeyName:   e.keyName,
			StartedAt: time.Now().Add(-time.Hour),
			Status:    e.status,
		}
		err = LogRotation(entry)
		require.NoError(t, err)
	}

	// Filter by status
	results, err := ListRotations(QueryOptions{Status: "success"})
	require.NoError(t, err)
	assert.Len(t, results, 2)

	results, err = ListRotations(QueryOptions{Status: "failed"})
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestListRotationsWithDateFilters(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	err := Init(dbPath)
	require.NoError(t, err)
	defer Close()

	// Log entries at different times
	now := time.Now()
	timestamps := []time.Time{
		now.Add(-48 * time.Hour),
		now.Add(-24 * time.Hour),
		now.Add(-1 * time.Hour),
	}

	for _, ts := range timestamps {
		entry := &RotationEntry{
			KeyName:   "KEY",
			StartedAt: ts,
			Status:    "success",
		}
		err = LogRotation(entry)
		require.NoError(t, err)
	}

	// Filter since yesterday
	results, err := ListRotations(QueryOptions{
		Since: now.Add(-25 * time.Hour),
	})
	require.NoError(t, err)
	assert.Len(t, results, 2)

	// Filter until yesterday
	results, err = ListRotations(QueryOptions{
		Until: now.Add(-23 * time.Hour),
	})
	require.NoError(t, err)
	assert.Len(t, results, 2)

	// Filter with both
	results, err = ListRotations(QueryOptions{
		Since: now.Add(-30 * time.Hour),
		Until: now.Add(-20 * time.Hour),
	})
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestListRotationsWithOffset(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	err := Init(dbPath)
	require.NoError(t, err)
	defer Close()

	// Log 5 entries
	for i := 0; i < 5; i++ {
		entry := &RotationEntry{
			KeyName:   "KEY",
			StartedAt: time.Now().Add(-time.Duration(i) * time.Hour),
			Status:    "success",
		}
		err = LogRotation(entry)
		require.NoError(t, err)
	}

	// Get first 2
	results, err := ListRotations(QueryOptions{Limit: 2})
	require.NoError(t, err)
	assert.Len(t, results, 2)

	// Get next 2 with offset
	results, err = ListRotations(QueryOptions{Limit: 2, Offset: 2})
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestLogRotationNoDb(t *testing.T) {
	// Close any existing connection
	Close()
	db = nil

	entry := &RotationEntry{
		KeyName:   "KEY",
		StartedAt: time.Now(),
		Status:    "success",
	}

	// Should not error when db is nil
	err := LogRotation(entry)
	assert.NoError(t, err)
}

func TestListRotationsNoDb(t *testing.T) {
	Close()
	db = nil

	results, err := ListRotations(QueryOptions{})
	assert.NoError(t, err)
	assert.Nil(t, results)
}

func TestGetRotationNoDb(t *testing.T) {
	Close()
	db = nil

	result, err := GetRotation("some-id")
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestPurgeOldEntriesNoDb(t *testing.T) {
	Close()
	db = nil

	err := PurgeOldEntries(30)
	assert.NoError(t, err)
}

func TestCloseNoDb(t *testing.T) {
	db = nil

	err := Close()
	assert.NoError(t, err)
}

func TestGetRotationNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	err := Init(dbPath)
	require.NoError(t, err)
	defer Close()

	_, err = GetRotation("nonexistent-id")
	assert.Error(t, err)
}

func TestLogRotationWithErrorMessage(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	err := Init(dbPath)
	require.NoError(t, err)
	defer Close()

	entry := &RotationEntry{
		KeyName:          "FAILED_KEY",
		StartedAt:        time.Now(),
		Status:           "failed",
		LocationsUpdated: 1,
		LocationsFailed:  1,
		ErrorMessage:     "Connection refused",
		Locations: []LocationEntry{
			{
				LocationType: "vercel",
				LocationPath: "proj/env1",
				Status:       "failed",
				ErrorMessage: "API error 500",
				UpdatedAt:    time.Now(),
			},
		},
	}

	err = LogRotation(entry)
	require.NoError(t, err)

	// Fetch and verify
	fetched, err := GetRotation(entry.ID)
	require.NoError(t, err)
	assert.Equal(t, "failed", fetched.Status)
	assert.Equal(t, "Connection refused", fetched.ErrorMessage)
	assert.Len(t, fetched.Locations, 1)
	assert.Equal(t, "API error 500", fetched.Locations[0].ErrorMessage)
}

func TestInitCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "subdir", "nested", "test.db")

	err := Init(dbPath)
	require.NoError(t, err)
	defer Close()

	// Should have created the directory and file
	assert.NotNil(t, db)
}
