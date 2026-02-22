package rotation

import (
	"fmt"

	"github.com/sudokatie/api-key-rotate/internal/audit"
	"github.com/sudokatie/api-key-rotate/internal/providers"
	"github.com/sudokatie/api-key-rotate/internal/scanner"
)

// Coordinator orchestrates the key rotation workflow
type Coordinator struct {
	initiatedBy string
}

// NewCoordinator creates a new rotation coordinator
func NewCoordinator(initiatedBy string) *Coordinator {
	return &Coordinator{
		initiatedBy: initiatedBy,
	}
}

// Execute performs the key rotation across all locations
// Returns the transaction for inspection and any error
func (c *Coordinator) Execute(keyName string, newValue string, locations []providers.Location) (*Transaction, error) {
	if len(locations) == 0 {
		return nil, fmt.Errorf("no locations to update")
	}

	tx := NewTransaction(keyName, newValue)

	// Add all locations to transaction
	for _, loc := range locations {
		tx.AddLocation(loc)
	}

	// Phase 1: Backup/record current state
	if err := c.backupPhase(tx); err != nil {
		return tx, err
	}

	// Phase 2: Execute updates
	if err := c.updatePhase(tx, keyName, newValue); err != nil {
		// Rollback already attempted inside updatePhase
		c.logToAudit(tx)
		return tx, err
	}

	// Phase 3: Log success
	c.logToAudit(tx)
	return tx, nil
}

// backupPhase creates backups for local files and records original values
func (c *Coordinator) backupPhase(tx *Transaction) error {
	for i := range tx.Locations {
		loc := &tx.Locations[i]

		if loc.Location.Type == "local" {
			backupPath, err := scanner.BackupFile(loc.Location.Path)
			if err != nil {
				return fmt.Errorf("backup %s: %w", loc.Location.Path, err)
			}
			tx.RecordBackup(loc.Location.Path, backupPath)
		}
		// Original value is already recorded in AddLocation via loc.Value
	}
	return nil
}

// updatePhase applies the new value to all locations
func (c *Coordinator) updatePhase(tx *Transaction, keyName string, newValue string) error {
	for i := range tx.Locations {
		loc := &tx.Locations[i]

		var err error
		if loc.Location.Type == "local" {
			err = scanner.UpdateKey(loc.Location.Path, keyName, newValue)
		} else {
			provider, ok := providers.Get(loc.Location.Provider)
			if !ok {
				err = fmt.Errorf("unknown provider: %s", loc.Location.Provider)
			} else {
				err = provider.Update(loc.Location, newValue)
			}
		}

		if err != nil {
			tx.MarkFailed(loc.Location.Path, err)

			// Attempt rollback of successful updates
			rollbackErr := Rollback(tx)

			if rollbackErr != nil {
				return fmt.Errorf("rotation failed: %w; rollback also failed: %v", err, rollbackErr)
			}
			return fmt.Errorf("rotation failed (rolled back): %w", err)
		}
		tx.MarkSuccess(loc.Location.Path)
	}
	return nil
}

// logToAudit records the rotation result to the audit log
func (c *Coordinator) logToAudit(tx *Transaction) {
	entry := tx.ToAuditEntry(c.initiatedBy)
	audit.LogRotation(entry)
}

// DryRunResult contains the results of a dry run
type DryRunResult struct {
	KeyName   string
	Locations []LocationPreview
}

// LocationPreview shows what would be updated
type LocationPreview struct {
	Type        string
	Path        string
	Project     string // For cloud providers
	Environment string // For cloud providers
	CurrentMask string // Masked current value
}

// DryRun simulates a rotation without making changes
func DryRun(keyName string, locations []providers.Location) *DryRunResult {
	result := &DryRunResult{
		KeyName:   keyName,
		Locations: make([]LocationPreview, 0, len(locations)),
	}

	for _, loc := range locations {
		preview := LocationPreview{
			Type:        loc.Type,
			Path:        loc.Path,
			Project:     loc.Project,
			Environment: loc.Environment,
			CurrentMask: maskKey(loc.Value),
		}
		result.Locations = append(result.Locations, preview)
	}

	return result
}

// ExecuteWithConfirmation wraps Execute with a confirmation callback
func (c *Coordinator) ExecuteWithConfirmation(
	keyName string,
	newValue string,
	locations []providers.Location,
	confirm func(dryRun *DryRunResult) bool,
) (*Transaction, error) {
	dryRun := DryRun(keyName, locations)

	if !confirm(dryRun) {
		return nil, fmt.Errorf("rotation cancelled by user")
	}

	return c.Execute(keyName, newValue, locations)
}
