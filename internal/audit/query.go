package audit

import (
	"time"
)

// QueryOptions for filtering audit entries
type QueryOptions struct {
	KeyName   string
	Status    string
	Since     time.Time
	Until     time.Time
	Limit     int
	Offset    int
}

// ListRotations returns rotation entries matching the filter
func ListRotations(opts QueryOptions) ([]RotationEntry, error) {
	if db == nil {
		return nil, nil
	}

	query := `SELECT id, key_name, started_at, completed_at, status, 
		locations_updated, locations_failed, error_message, initiated_by,
		old_key_preview, new_key_preview FROM rotations WHERE 1=1`
	args := []interface{}{}

	if opts.KeyName != "" {
		query += " AND key_name = ?"
		args = append(args, opts.KeyName)
	}
	if opts.Status != "" {
		query += " AND status = ?"
		args = append(args, opts.Status)
	}
	if !opts.Since.IsZero() {
		query += " AND started_at >= ?"
		args = append(args, opts.Since.Format(time.RFC3339))
	}
	if !opts.Until.IsZero() {
		query += " AND started_at <= ?"
		args = append(args, opts.Until.Format(time.RFC3339))
	}

	query += " ORDER BY started_at DESC"

	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
	}
	if opts.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, opts.Offset)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []RotationEntry
	for rows.Next() {
		var e RotationEntry
		var startedAt, completedAt string
		err := rows.Scan(&e.ID, &e.KeyName, &startedAt, &completedAt, &e.Status,
			&e.LocationsUpdated, &e.LocationsFailed, &e.ErrorMessage, &e.InitiatedBy,
			&e.OldKeyPreview, &e.NewKeyPreview)
		if err != nil {
			return nil, err
		}
		e.StartedAt, _ = time.Parse(time.RFC3339, startedAt)
		e.CompletedAt, _ = time.Parse(time.RFC3339, completedAt)
		entries = append(entries, e)
	}

	return entries, rows.Err()
}

// GetRotation returns a single rotation with its locations
func GetRotation(id string) (*RotationEntry, error) {
	if db == nil {
		return nil, nil
	}

	var e RotationEntry
	var startedAt, completedAt string
	err := db.QueryRow(`SELECT id, key_name, started_at, completed_at, status, 
		locations_updated, locations_failed, error_message, initiated_by,
		old_key_preview, new_key_preview FROM rotations WHERE id = ?`, id).
		Scan(&e.ID, &e.KeyName, &startedAt, &completedAt, &e.Status,
			&e.LocationsUpdated, &e.LocationsFailed, &e.ErrorMessage, &e.InitiatedBy,
			&e.OldKeyPreview, &e.NewKeyPreview)
	if err != nil {
		return nil, err
	}
	e.StartedAt, _ = time.Parse(time.RFC3339, startedAt)
	e.CompletedAt, _ = time.Parse(time.RFC3339, completedAt)

	// Get locations
	rows, err := db.Query(`SELECT id, rotation_id, location_type, location_path, 
		status, error_message, updated_at FROM rotation_locations WHERE rotation_id = ?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var loc LocationEntry
		var updatedAt string
		err := rows.Scan(&loc.ID, &loc.RotationID, &loc.LocationType, &loc.LocationPath,
			&loc.Status, &loc.ErrorMessage, &updatedAt)
		if err != nil {
			return nil, err
		}
		loc.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		e.Locations = append(e.Locations, loc)
	}

	return &e, rows.Err()
}
