package audit

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

var db *sql.DB

// Init initializes the audit database
func Init(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	var err error
	db, err = sql.Open("sqlite", path)
	if err != nil {
		return err
	}

	if _, err := db.Exec(createSchema); err != nil {
		return err
	}

	return nil
}

// Close closes the database connection
func Close() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

// LogRotation records a rotation event
func LogRotation(entry *RotationEntry) error {
	if db == nil {
		return nil // Skip if audit not initialized
	}

	entry.ID = generateID()
	entry.CompletedAt = time.Now()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO rotations (id, key_name, started_at, completed_at, status, 
			locations_updated, locations_failed, error_message, initiated_by,
			old_key_preview, new_key_preview)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID, entry.KeyName, entry.StartedAt.Format(time.RFC3339),
		entry.CompletedAt.Format(time.RFC3339), entry.Status,
		entry.LocationsUpdated, entry.LocationsFailed, entry.ErrorMessage,
		entry.InitiatedBy, entry.OldKeyPreview, entry.NewKeyPreview)
	if err != nil {
		return err
	}

	for _, loc := range entry.Locations {
		loc.ID = generateID()
		loc.RotationID = entry.ID
		_, err = tx.Exec(`
			INSERT INTO rotation_locations (id, rotation_id, location_type, 
				location_path, status, error_message, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			loc.ID, loc.RotationID, loc.LocationType, loc.LocationPath,
			loc.Status, loc.ErrorMessage, loc.UpdatedAt.Format(time.RFC3339))
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// PurgeOldEntries removes entries older than the given number of days
func PurgeOldEntries(retentionDays int) error {
	if db == nil {
		return nil
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	_, err := db.Exec("DELETE FROM rotations WHERE started_at < ?", cutoff.Format(time.RFC3339))
	return err
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
