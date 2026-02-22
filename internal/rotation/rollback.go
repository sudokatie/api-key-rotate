package rotation

import (
	"fmt"
	"os"
	"strings"

	"github.com/sudokatie/api-key-rotate/internal/providers"
	"github.com/sudokatie/api-key-rotate/internal/scanner"
)

// Rollback attempts to restore all successfully updated locations to their original state
func Rollback(tx *Transaction) error {
	candidates := tx.RollbackCandidates()
	if len(candidates) == 0 {
		return nil
	}

	var errs []string

	for _, locState := range candidates {
		var err error

		if locState.Location.Type == "local" {
			err = rollbackLocalFile(locState)
		} else {
			err = rollbackProvider(locState)
		}

		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %s", locState.Location.Path, err.Error()))
		} else {
			tx.MarkRolledBack(locState.Location.Path)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("rollback errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

// rollbackLocalFile restores a local file from its backup
func rollbackLocalFile(locState LocationState) error {
	if locState.BackupPath == "" {
		return fmt.Errorf("no backup available")
	}
	return scanner.RestoreBackup(locState.BackupPath, locState.Location.Path)
}

// rollbackProvider uses the provider's rollback method
func rollbackProvider(locState LocationState) error {
	provider, ok := providers.Get(locState.Location.Provider)
	if !ok {
		return fmt.Errorf("unknown provider: %s", locState.Location.Provider)
	}

	if !provider.SupportsRollback() {
		// Not all providers support rollback - we'll update with original value
		return provider.Update(locState.Location, locState.OriginalValue)
	}

	return provider.Rollback(locState.Location, locState.OriginalValue)
}

// CleanupBackups removes backup files after successful rotation
func CleanupBackups(tx *Transaction) error {
	// Only clean up if all succeeded
	if !tx.AllSucceeded() {
		return nil
	}

	var errs []string

	for _, locState := range tx.Locations {
		if locState.BackupPath != "" {
			// Use os.Remove - backup files should be deleted after success
			if err := removeFile(locState.BackupPath); err != nil {
				errs = append(errs, fmt.Sprintf("%s: %s", locState.BackupPath, err.Error()))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

// removeFile wraps os.Remove for testing
var removeFile = osRemove

func osRemove(path string) error {
	return os.Remove(path)
}
