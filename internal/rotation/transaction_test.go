package rotation

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sudokatie/api-key-rotate/internal/providers"
)

func TestNewTransaction(t *testing.T) {
	tx := NewTransaction("API_KEY", "new-secret-value")

	assert.NotEmpty(t, tx.ID)
	assert.Equal(t, "API_KEY", tx.KeyName)
	assert.Equal(t, "new-secret-value", tx.NewValue)
	assert.False(t, tx.StartedAt.IsZero())
	assert.Empty(t, tx.Locations)
}

func TestAddLocation(t *testing.T) {
	tx := NewTransaction("API_KEY", "new-value")

	loc := providers.Location{
		Type:  "local",
		Path:  "/path/to/.env",
		Value: "old-value",
	}

	tx.AddLocation(loc)

	require.Len(t, tx.Locations, 1)
	assert.Equal(t, StatusPending, tx.Locations[0].Status)
	assert.Equal(t, "old-value", tx.Locations[0].OriginalValue)
	assert.Equal(t, "/path/to/.env", tx.Locations[0].Location.Path)
}

func TestRecordBackup(t *testing.T) {
	tx := NewTransaction("API_KEY", "new-value")
	tx.AddLocation(providers.Location{Path: "/path/to/.env"})

	tx.RecordBackup("/path/to/.env", "/path/to/.env.bak.123")

	assert.Equal(t, "/path/to/.env.bak.123", tx.Locations[0].BackupPath)
}

func TestRecordBackupWrongPath(t *testing.T) {
	tx := NewTransaction("API_KEY", "new-value")
	tx.AddLocation(providers.Location{Path: "/path/to/.env"})

	tx.RecordBackup("/wrong/path", "/backup/path")

	assert.Empty(t, tx.Locations[0].BackupPath)
}

func TestMarkSuccess(t *testing.T) {
	tx := NewTransaction("API_KEY", "new-value")
	tx.AddLocation(providers.Location{Path: "/path/to/.env"})

	tx.MarkSuccess("/path/to/.env")

	assert.Equal(t, StatusSuccess, tx.Locations[0].Status)
	assert.False(t, tx.Locations[0].UpdatedAt.IsZero())
}

func TestMarkFailed(t *testing.T) {
	tx := NewTransaction("API_KEY", "new-value")
	tx.AddLocation(providers.Location{Path: "/path/to/.env"})

	err := errors.New("permission denied")
	tx.MarkFailed("/path/to/.env", err)

	assert.Equal(t, StatusFailed, tx.Locations[0].Status)
	assert.Equal(t, err, tx.Locations[0].Error)
	assert.False(t, tx.Locations[0].UpdatedAt.IsZero())
}

func TestMarkRolledBack(t *testing.T) {
	tx := NewTransaction("API_KEY", "new-value")
	tx.AddLocation(providers.Location{Path: "/path/to/.env"})

	tx.MarkRolledBack("/path/to/.env")

	assert.Equal(t, StatusRolledBack, tx.Locations[0].Status)
}

func TestGetLocation(t *testing.T) {
	tx := NewTransaction("API_KEY", "new-value")
	tx.AddLocation(providers.Location{Path: "/path/one/.env"})
	tx.AddLocation(providers.Location{Path: "/path/two/.env"})

	loc := tx.GetLocation("/path/two/.env")
	require.NotNil(t, loc)
	assert.Equal(t, "/path/two/.env", loc.Location.Path)

	missing := tx.GetLocation("/nonexistent")
	assert.Nil(t, missing)
}

func TestSuccessCount(t *testing.T) {
	tx := NewTransaction("API_KEY", "new-value")
	tx.AddLocation(providers.Location{Path: "/path/one/.env"})
	tx.AddLocation(providers.Location{Path: "/path/two/.env"})
	tx.AddLocation(providers.Location{Path: "/path/three/.env"})

	tx.MarkSuccess("/path/one/.env")
	tx.MarkSuccess("/path/two/.env")
	tx.MarkFailed("/path/three/.env", errors.New("fail"))

	assert.Equal(t, 2, tx.SuccessCount())
}

func TestFailedCount(t *testing.T) {
	tx := NewTransaction("API_KEY", "new-value")
	tx.AddLocation(providers.Location{Path: "/path/one/.env"})
	tx.AddLocation(providers.Location{Path: "/path/two/.env"})

	tx.MarkFailed("/path/one/.env", errors.New("fail"))
	tx.MarkSuccess("/path/two/.env")

	assert.Equal(t, 1, tx.FailedCount())
}

func TestPendingLocations(t *testing.T) {
	tx := NewTransaction("API_KEY", "new-value")
	tx.AddLocation(providers.Location{Path: "/path/one/.env"})
	tx.AddLocation(providers.Location{Path: "/path/two/.env"})
	tx.AddLocation(providers.Location{Path: "/path/three/.env"})

	tx.MarkSuccess("/path/one/.env")

	pending := tx.PendingLocations()
	assert.Len(t, pending, 2)
}

func TestFailedLocations(t *testing.T) {
	tx := NewTransaction("API_KEY", "new-value")
	tx.AddLocation(providers.Location{Path: "/path/one/.env"})
	tx.AddLocation(providers.Location{Path: "/path/two/.env"})

	tx.MarkFailed("/path/one/.env", errors.New("err1"))
	tx.MarkFailed("/path/two/.env", errors.New("err2"))

	failed := tx.FailedLocations()
	assert.Len(t, failed, 2)
}

func TestRollbackCandidates(t *testing.T) {
	tx := NewTransaction("API_KEY", "new-value")
	tx.AddLocation(providers.Location{Path: "/path/one/.env"})
	tx.AddLocation(providers.Location{Path: "/path/two/.env"})
	tx.AddLocation(providers.Location{Path: "/path/three/.env"})

	tx.MarkSuccess("/path/one/.env")
	tx.MarkSuccess("/path/two/.env")
	tx.MarkFailed("/path/three/.env", errors.New("fail"))

	candidates := tx.RollbackCandidates()
	assert.Len(t, candidates, 2)
}

func TestHasFailures(t *testing.T) {
	tx := NewTransaction("API_KEY", "new-value")
	tx.AddLocation(providers.Location{Path: "/path/.env"})

	assert.False(t, tx.HasFailures())

	tx.MarkFailed("/path/.env", errors.New("fail"))
	assert.True(t, tx.HasFailures())
}

func TestAllSucceeded(t *testing.T) {
	tx := NewTransaction("API_KEY", "new-value")
	tx.AddLocation(providers.Location{Path: "/path/one/.env"})
	tx.AddLocation(providers.Location{Path: "/path/two/.env"})

	assert.False(t, tx.AllSucceeded())

	tx.MarkSuccess("/path/one/.env")
	assert.False(t, tx.AllSucceeded())

	tx.MarkSuccess("/path/two/.env")
	assert.True(t, tx.AllSucceeded())
}

func TestAllSucceededEmptyTransaction(t *testing.T) {
	tx := NewTransaction("API_KEY", "new-value")
	assert.False(t, tx.AllSucceeded())
}

func TestOverallStatus(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*Transaction)
		expected string
	}{
		{
			name:     "empty transaction",
			setup:    func(tx *Transaction) {},
			expected: "empty",
		},
		{
			name: "all success",
			setup: func(tx *Transaction) {
				tx.AddLocation(providers.Location{Path: "/a"})
				tx.AddLocation(providers.Location{Path: "/b"})
				tx.MarkSuccess("/a")
				tx.MarkSuccess("/b")
			},
			expected: "success",
		},
		{
			name: "all failed",
			setup: func(tx *Transaction) {
				tx.AddLocation(providers.Location{Path: "/a"})
				tx.MarkFailed("/a", errors.New("err"))
			},
			expected: "failed",
		},
		{
			name: "partial",
			setup: func(tx *Transaction) {
				tx.AddLocation(providers.Location{Path: "/a"})
				tx.AddLocation(providers.Location{Path: "/b"})
				tx.MarkSuccess("/a")
				tx.MarkFailed("/b", errors.New("err"))
			},
			expected: "partial",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := NewTransaction("KEY", "val")
			tt.setup(tx)
			assert.Equal(t, tt.expected, tx.OverallStatus())
		})
	}
}

func TestToAuditEntry(t *testing.T) {
	tx := NewTransaction("API_KEY", "new-secret-12345")
	tx.AddLocation(providers.Location{
		Type:  "local",
		Path:  "/path/one/.env",
		Value: "old-secret-12345",
	})
	tx.AddLocation(providers.Location{
		Type:  "vercel",
		Path:  "project-id/production/API_KEY",
		Value: "old-secret-12345",
	})

	tx.MarkSuccess("/path/one/.env")
	tx.MarkFailed("project-id/production/API_KEY", errors.New("rate limit"))

	entry := tx.ToAuditEntry("katie")

	assert.Equal(t, "API_KEY", entry.KeyName)
	assert.Equal(t, "partial", entry.Status)
	assert.Equal(t, 1, entry.LocationsUpdated)
	assert.Equal(t, 1, entry.LocationsFailed)
	assert.Equal(t, "katie", entry.InitiatedBy)
	assert.Equal(t, "rate limit", entry.ErrorMessage)
	assert.Equal(t, "old-****", entry.OldKeyPreview)
	assert.Equal(t, "new-****", entry.NewKeyPreview)
	assert.Len(t, entry.Locations, 2)
}

func TestMaskKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "****"},
		{"abc", "****"},
		{"abcd", "****"},
		{"abcde", "abcd****"},
		{"sk_live_12345678", "sk_l****"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, maskKey(tt.input))
		})
	}
}
