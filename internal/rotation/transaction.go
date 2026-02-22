package rotation

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/sudokatie/api-key-rotate/internal/audit"
	"github.com/sudokatie/api-key-rotate/internal/providers"
)

// Status represents the state of a location in a transaction
type Status string

const (
	StatusPending    Status = "pending"
	StatusSuccess    Status = "success"
	StatusFailed     Status = "failed"
	StatusRolledBack Status = "rolledback"
)

// Transaction tracks the state of a key rotation operation
type Transaction struct {
	ID        string
	KeyName   string
	NewValue  string
	StartedAt time.Time
	Locations []LocationState
}

// LocationState tracks the state of a single location in the rotation
type LocationState struct {
	Location      providers.Location
	BackupPath    string // For local files only
	OriginalValue string
	Status        Status
	Error         error
	UpdatedAt     time.Time
}

// NewTransaction creates a new transaction for rotating a key
func NewTransaction(keyName string, newValue string) *Transaction {
	return &Transaction{
		ID:        generateID(),
		KeyName:   keyName,
		NewValue:  newValue,
		StartedAt: time.Now(),
		Locations: make([]LocationState, 0),
	}
}

// AddLocation adds a location to be updated in this transaction
func (tx *Transaction) AddLocation(loc providers.Location) {
	tx.Locations = append(tx.Locations, LocationState{
		Location:      loc,
		OriginalValue: loc.Value,
		Status:        StatusPending,
	})
}

// RecordBackup stores the backup path for a local file location
func (tx *Transaction) RecordBackup(path string, backupPath string) {
	for i := range tx.Locations {
		if tx.Locations[i].Location.Path == path {
			tx.Locations[i].BackupPath = backupPath
			break
		}
	}
}

// MarkSuccess marks a location as successfully updated
func (tx *Transaction) MarkSuccess(path string) {
	for i := range tx.Locations {
		if tx.Locations[i].Location.Path == path {
			tx.Locations[i].Status = StatusSuccess
			tx.Locations[i].UpdatedAt = time.Now()
			break
		}
	}
}

// MarkFailed marks a location as failed with an error
func (tx *Transaction) MarkFailed(path string, err error) {
	for i := range tx.Locations {
		if tx.Locations[i].Location.Path == path {
			tx.Locations[i].Status = StatusFailed
			tx.Locations[i].Error = err
			tx.Locations[i].UpdatedAt = time.Now()
			break
		}
	}
}

// MarkRolledBack marks a location as rolled back
func (tx *Transaction) MarkRolledBack(path string) {
	for i := range tx.Locations {
		if tx.Locations[i].Location.Path == path {
			tx.Locations[i].Status = StatusRolledBack
			tx.Locations[i].UpdatedAt = time.Now()
			break
		}
	}
}

// GetLocation returns the state for a specific location path
func (tx *Transaction) GetLocation(path string) *LocationState {
	for i := range tx.Locations {
		if tx.Locations[i].Location.Path == path {
			return &tx.Locations[i]
		}
	}
	return nil
}

// SuccessCount returns the number of successfully updated locations
func (tx *Transaction) SuccessCount() int {
	count := 0
	for _, loc := range tx.Locations {
		if loc.Status == StatusSuccess {
			count++
		}
	}
	return count
}

// FailedCount returns the number of failed locations
func (tx *Transaction) FailedCount() int {
	count := 0
	for _, loc := range tx.Locations {
		if loc.Status == StatusFailed {
			count++
		}
	}
	return count
}

// PendingLocations returns all locations still pending update
func (tx *Transaction) PendingLocations() []LocationState {
	var pending []LocationState
	for _, loc := range tx.Locations {
		if loc.Status == StatusPending {
			pending = append(pending, loc)
		}
	}
	return pending
}

// FailedLocations returns all locations that failed
func (tx *Transaction) FailedLocations() []LocationState {
	var failed []LocationState
	for _, loc := range tx.Locations {
		if loc.Status == StatusFailed {
			failed = append(failed, loc)
		}
	}
	return failed
}

// RollbackCandidates returns locations that can be rolled back
// (successfully updated or local files with backups)
func (tx *Transaction) RollbackCandidates() []LocationState {
	var candidates []LocationState
	for _, loc := range tx.Locations {
		if loc.Status == StatusSuccess {
			candidates = append(candidates, loc)
		}
	}
	return candidates
}

// HasFailures returns true if any location failed
func (tx *Transaction) HasFailures() bool {
	return tx.FailedCount() > 0
}

// AllSucceeded returns true if all locations succeeded
func (tx *Transaction) AllSucceeded() bool {
	for _, loc := range tx.Locations {
		if loc.Status != StatusSuccess {
			return false
		}
	}
	return len(tx.Locations) > 0
}

// OverallStatus returns the overall transaction status for audit
func (tx *Transaction) OverallStatus() string {
	if len(tx.Locations) == 0 {
		return "empty"
	}
	if tx.AllSucceeded() {
		return "success"
	}
	if tx.SuccessCount() == 0 {
		return "failed"
	}
	return "partial"
}

// ToAuditEntry converts the transaction to an audit log entry
func (tx *Transaction) ToAuditEntry(initiatedBy string) *audit.RotationEntry {
	entry := &audit.RotationEntry{
		KeyName:          tx.KeyName,
		StartedAt:        tx.StartedAt,
		CompletedAt:      time.Now(),
		Status:           tx.OverallStatus(),
		LocationsUpdated: tx.SuccessCount(),
		LocationsFailed:  tx.FailedCount(),
		InitiatedBy:      initiatedBy,
		OldKeyPreview:    maskKey(tx.Locations[0].OriginalValue),
		NewKeyPreview:    maskKey(tx.NewValue),
		Locations:        make([]audit.LocationEntry, 0, len(tx.Locations)),
	}

	// Collect error messages
	var errMsgs []string
	for _, loc := range tx.Locations {
		if loc.Error != nil {
			errMsgs = append(errMsgs, loc.Error.Error())
		}

		locEntry := audit.LocationEntry{
			LocationType: loc.Location.Type,
			LocationPath: loc.Location.Path,
			Status:       string(loc.Status),
			UpdatedAt:    loc.UpdatedAt,
		}
		if loc.Error != nil {
			locEntry.ErrorMessage = loc.Error.Error()
		}
		entry.Locations = append(entry.Locations, locEntry)
	}

	if len(errMsgs) > 0 {
		entry.ErrorMessage = errMsgs[0]
	}

	return entry
}

// maskKey returns a preview of a key value (first 4 chars + ***)
func maskKey(value string) string {
	if len(value) <= 4 {
		return "****"
	}
	return value[:4] + "****"
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
