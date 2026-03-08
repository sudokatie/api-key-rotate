package scheduler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// RotationRecord stores details of a scheduled rotation
type RotationRecord struct {
	// JobName identifies which job ran
	JobName string `json:"job_name"`
	// KeyName is the key that was rotated
	KeyName string `json:"key_name"`
	// Timestamp when rotation occurred
	Timestamp time.Time `json:"timestamp"`
	// Success indicates if rotation completed
	Success bool `json:"success"`
	// Error message if rotation failed
	Error string `json:"error,omitempty"`
	// LocationCount is how many locations were updated
	LocationCount int `json:"location_count"`
	// Duration of the rotation operation
	Duration time.Duration `json:"duration"`
}

// History tracks rotation events
type History struct {
	path    string
	records []RotationRecord
	maxSize int
	mu      sync.RWMutex
}

// NewHistory creates a history tracker
func NewHistory(path string, maxSize int) (*History, error) {
	if maxSize <= 0 {
		maxSize = 1000
	}

	h := &History{
		path:    path,
		records: make([]RotationRecord, 0),
		maxSize: maxSize,
	}

	// Load existing history if file exists
	if err := h.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return h, nil
}

// Add records a rotation event
func (h *History) Add(record RotationRecord) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.records = append(h.records, record)

	// Trim if over max size
	if len(h.records) > h.maxSize {
		h.records = h.records[len(h.records)-h.maxSize:]
	}

	return h.save()
}

// GetAll returns all records
func (h *History) GetAll() []RotationRecord {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]RotationRecord, len(h.records))
	copy(result, h.records)
	return result
}

// GetByJob returns records for a specific job
func (h *History) GetByJob(jobName string) []RotationRecord {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var result []RotationRecord
	for _, r := range h.records {
		if r.JobName == jobName {
			result = append(result, r)
		}
	}
	return result
}

// GetByKey returns records for a specific key
func (h *History) GetByKey(keyName string) []RotationRecord {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var result []RotationRecord
	for _, r := range h.records {
		if r.KeyName == keyName {
			result = append(result, r)
		}
	}
	return result
}

// GetRecent returns the most recent n records
func (h *History) GetRecent(n int) []RotationRecord {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if n <= 0 || n > len(h.records) {
		n = len(h.records)
	}

	result := make([]RotationRecord, n)
	copy(result, h.records[len(h.records)-n:])

	// Return in reverse chronological order
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.After(result[j].Timestamp)
	})

	return result
}

// GetSince returns records since a given time
func (h *History) GetSince(since time.Time) []RotationRecord {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var result []RotationRecord
	for _, r := range h.records {
		if r.Timestamp.After(since) || r.Timestamp.Equal(since) {
			result = append(result, r)
		}
	}
	return result
}

// GetFailed returns only failed rotations
func (h *History) GetFailed() []RotationRecord {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var result []RotationRecord
	for _, r := range h.records {
		if !r.Success {
			result = append(result, r)
		}
	}
	return result
}

// Stats returns statistics about rotation history
type Stats struct {
	TotalRotations   int           `json:"total_rotations"`
	SuccessCount     int           `json:"success_count"`
	FailureCount     int           `json:"failure_count"`
	SuccessRate      float64       `json:"success_rate"`
	AverageDuration  time.Duration `json:"average_duration"`
	LastRotation     *time.Time    `json:"last_rotation,omitempty"`
	LastFailure      *time.Time    `json:"last_failure,omitempty"`
	UniqueKeys       int           `json:"unique_keys"`
	UniqueJobs       int           `json:"unique_jobs"`
}

// GetStats returns statistics about rotations
func (h *History) GetStats() Stats {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := Stats{}
	keys := make(map[string]struct{})
	jobs := make(map[string]struct{})
	var totalDuration time.Duration

	for _, r := range h.records {
		stats.TotalRotations++
		keys[r.KeyName] = struct{}{}
		jobs[r.JobName] = struct{}{}
		totalDuration += r.Duration

		if r.Success {
			stats.SuccessCount++
			if stats.LastRotation == nil || r.Timestamp.After(*stats.LastRotation) {
				t := r.Timestamp
				stats.LastRotation = &t
			}
		} else {
			stats.FailureCount++
			if stats.LastFailure == nil || r.Timestamp.After(*stats.LastFailure) {
				t := r.Timestamp
				stats.LastFailure = &t
			}
		}
	}

	stats.UniqueKeys = len(keys)
	stats.UniqueJobs = len(jobs)

	if stats.TotalRotations > 0 {
		stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalRotations)
		stats.AverageDuration = totalDuration / time.Duration(stats.TotalRotations)
	}

	return stats
}

// load reads history from disk
func (h *History) load() error {
	data, err := os.ReadFile(h.path)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &h.records)
}

// save writes history to disk
func (h *History) save() error {
	// Ensure directory exists
	dir := filepath.Dir(h.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create history dir: %w", err)
	}

	data, err := json.MarshalIndent(h.records, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal history: %w", err)
	}

	return os.WriteFile(h.path, data, 0644)
}

// Clear removes all history records
func (h *History) Clear() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.records = make([]RotationRecord, 0)
	return h.save()
}
