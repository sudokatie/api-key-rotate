package scheduler

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewHistory(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "history.json")

	h, err := NewHistory(path, 100)
	if err != nil {
		t.Fatalf("NewHistory failed: %v", err)
	}
	if h == nil {
		t.Fatal("expected non-nil history")
	}
}

func TestHistoryAdd(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "history.json")

	h, _ := NewHistory(path, 100)

	record := RotationRecord{
		JobName:       "test-job",
		KeyName:       "API_KEY",
		Timestamp:     time.Now(),
		Success:       true,
		LocationCount: 3,
		Duration:      time.Second,
	}

	err := h.Add(record)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	records := h.GetAll()
	if len(records) != 1 {
		t.Errorf("expected 1 record, got %d", len(records))
	}
}

func TestHistoryPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "history.json")

	h1, _ := NewHistory(path, 100)
	h1.Add(RotationRecord{
		JobName:   "test-job",
		KeyName:   "API_KEY",
		Timestamp: time.Now(),
		Success:   true,
	})

	// Create new instance from same file
	h2, err := NewHistory(path, 100)
	if err != nil {
		t.Fatalf("NewHistory failed: %v", err)
	}

	records := h2.GetAll()
	if len(records) != 1 {
		t.Errorf("expected 1 record from disk, got %d", len(records))
	}
}

func TestHistoryMaxSize(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "history.json")

	maxSize := 5
	h, _ := NewHistory(path, maxSize)

	// Add more records than max
	for i := 0; i < 10; i++ {
		h.Add(RotationRecord{
			JobName:   "job",
			KeyName:   "key",
			Timestamp: time.Now(),
			Success:   true,
		})
	}

	records := h.GetAll()
	if len(records) != maxSize {
		t.Errorf("expected %d records (max), got %d", maxSize, len(records))
	}
}

func TestHistoryGetByJob(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "history.json")

	h, _ := NewHistory(path, 100)

	h.Add(RotationRecord{JobName: "job1", KeyName: "key1", Timestamp: time.Now()})
	h.Add(RotationRecord{JobName: "job2", KeyName: "key2", Timestamp: time.Now()})
	h.Add(RotationRecord{JobName: "job1", KeyName: "key1", Timestamp: time.Now()})

	records := h.GetByJob("job1")
	if len(records) != 2 {
		t.Errorf("expected 2 records for job1, got %d", len(records))
	}
}

func TestHistoryGetByKey(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "history.json")

	h, _ := NewHistory(path, 100)

	h.Add(RotationRecord{JobName: "job1", KeyName: "API_KEY", Timestamp: time.Now()})
	h.Add(RotationRecord{JobName: "job2", KeyName: "SECRET", Timestamp: time.Now()})
	h.Add(RotationRecord{JobName: "job3", KeyName: "API_KEY", Timestamp: time.Now()})

	records := h.GetByKey("API_KEY")
	if len(records) != 2 {
		t.Errorf("expected 2 records for API_KEY, got %d", len(records))
	}
}

func TestHistoryGetRecent(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "history.json")

	h, _ := NewHistory(path, 100)

	for i := 0; i < 10; i++ {
		h.Add(RotationRecord{
			JobName:   "job",
			KeyName:   "key",
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
		})
	}

	records := h.GetRecent(3)
	if len(records) != 3 {
		t.Errorf("expected 3 recent records, got %d", len(records))
	}

	// Should be in reverse chronological order
	if records[0].Timestamp.Before(records[1].Timestamp) {
		t.Error("records should be in reverse chronological order")
	}
}

func TestHistoryGetSince(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "history.json")

	h, _ := NewHistory(path, 100)

	now := time.Now()
	h.Add(RotationRecord{JobName: "old", Timestamp: now.Add(-2 * time.Hour)})
	h.Add(RotationRecord{JobName: "recent", Timestamp: now.Add(-30 * time.Minute)})
	h.Add(RotationRecord{JobName: "new", Timestamp: now})

	records := h.GetSince(now.Add(-1 * time.Hour))
	if len(records) != 2 {
		t.Errorf("expected 2 records since 1h ago, got %d", len(records))
	}
}

func TestHistoryGetFailed(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "history.json")

	h, _ := NewHistory(path, 100)

	h.Add(RotationRecord{JobName: "success", Success: true})
	h.Add(RotationRecord{JobName: "fail1", Success: false, Error: "error 1"})
	h.Add(RotationRecord{JobName: "fail2", Success: false, Error: "error 2"})

	records := h.GetFailed()
	if len(records) != 2 {
		t.Errorf("expected 2 failed records, got %d", len(records))
	}
}

func TestHistoryStats(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "history.json")

	h, _ := NewHistory(path, 100)

	now := time.Now()
	h.Add(RotationRecord{
		JobName:   "job1",
		KeyName:   "key1",
		Timestamp: now.Add(-1 * time.Hour),
		Success:   true,
		Duration:  100 * time.Millisecond,
	})
	h.Add(RotationRecord{
		JobName:   "job2",
		KeyName:   "key2",
		Timestamp: now.Add(-30 * time.Minute),
		Success:   false,
		Error:     "failed",
		Duration:  200 * time.Millisecond,
	})
	h.Add(RotationRecord{
		JobName:   "job1",
		KeyName:   "key1",
		Timestamp: now,
		Success:   true,
		Duration:  150 * time.Millisecond,
	})

	stats := h.GetStats()

	if stats.TotalRotations != 3 {
		t.Errorf("expected 3 total rotations, got %d", stats.TotalRotations)
	}
	if stats.SuccessCount != 2 {
		t.Errorf("expected 2 successes, got %d", stats.SuccessCount)
	}
	if stats.FailureCount != 1 {
		t.Errorf("expected 1 failure, got %d", stats.FailureCount)
	}
	if stats.UniqueKeys != 2 {
		t.Errorf("expected 2 unique keys, got %d", stats.UniqueKeys)
	}
	if stats.UniqueJobs != 2 {
		t.Errorf("expected 2 unique jobs, got %d", stats.UniqueJobs)
	}
}

func TestHistoryClear(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "history.json")

	h, _ := NewHistory(path, 100)

	h.Add(RotationRecord{JobName: "job", Success: true})
	h.Add(RotationRecord{JobName: "job", Success: true})

	err := h.Clear()
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	records := h.GetAll()
	if len(records) != 0 {
		t.Errorf("expected 0 records after clear, got %d", len(records))
	}

	// Verify persisted
	h2, _ := NewHistory(path, 100)
	if len(h2.GetAll()) != 0 {
		t.Error("clear should persist")
	}
}

func TestHistoryMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nonexistent", "history.json")

	// Should create directory
	h, err := NewHistory(path, 100)
	if err != nil {
		t.Fatalf("NewHistory failed: %v", err)
	}

	h.Add(RotationRecord{JobName: "test"})

	// Verify file was created
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("history file should have been created")
	}
}
