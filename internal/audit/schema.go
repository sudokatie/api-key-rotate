package audit

import "time"

const createSchema = `
CREATE TABLE IF NOT EXISTS rotations (
    id TEXT PRIMARY KEY,
    key_name TEXT NOT NULL,
    started_at TEXT NOT NULL,
    completed_at TEXT NOT NULL,
    status TEXT NOT NULL,
    locations_updated INTEGER DEFAULT 0,
    locations_failed INTEGER DEFAULT 0,
    error_message TEXT,
    initiated_by TEXT,
    old_key_preview TEXT,
    new_key_preview TEXT
);

CREATE TABLE IF NOT EXISTS rotation_locations (
    id TEXT PRIMARY KEY,
    rotation_id TEXT NOT NULL,
    location_type TEXT NOT NULL,
    location_path TEXT NOT NULL,
    status TEXT NOT NULL,
    error_message TEXT,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (rotation_id) REFERENCES rotations(id)
);

CREATE INDEX IF NOT EXISTS idx_rotations_key ON rotations(key_name);
CREATE INDEX IF NOT EXISTS idx_rotations_date ON rotations(started_at);
CREATE INDEX IF NOT EXISTS idx_locations_rotation ON rotation_locations(rotation_id);
`

// RotationEntry represents an audit log entry
type RotationEntry struct {
	ID               string
	KeyName          string
	StartedAt        time.Time
	CompletedAt      time.Time
	Status           string
	LocationsUpdated int
	LocationsFailed  int
	ErrorMessage     string
	InitiatedBy      string
	OldKeyPreview    string
	NewKeyPreview    string
	Locations        []LocationEntry
}

// LocationEntry represents a location update in a rotation
type LocationEntry struct {
	ID           string
	RotationID   string
	LocationType string
	LocationPath string
	Status       string
	ErrorMessage string
	UpdatedAt    time.Time
}
