package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/sudokatie/api-key-rotate/internal/audit"
	"github.com/sudokatie/api-key-rotate/internal/providers"
	"github.com/sudokatie/api-key-rotate/internal/rotation"
)

// RotationJob represents a scheduled rotation
type RotationJob struct {
	// Name is a unique identifier for this job
	Name string `json:"name" yaml:"name"`
	// KeyName is the key to rotate
	KeyName string `json:"key_name" yaml:"key_name"`
	// Schedule is a cron expression (e.g., "0 0 * * 0" for weekly)
	Schedule string `json:"schedule" yaml:"schedule"`
	// Locations to update when rotating
	Locations []providers.Location `json:"locations" yaml:"locations"`
	// Generator for creating new key values
	Generator KeyGenerator `json:"-" yaml:"-"`
	// NotifyBefore is how long before rotation to send notification
	NotifyBefore time.Duration `json:"notify_before" yaml:"notify_before"`
	// NotifyCallback is called before rotation (optional)
	NotifyCallback func(job *RotationJob, rotateAt time.Time) `json:"-" yaml:"-"`
	// Enabled controls whether the job runs
	Enabled bool `json:"enabled" yaml:"enabled"`
}

// KeyGenerator creates new key values
type KeyGenerator interface {
	Generate() (string, error)
}

// Scheduler manages scheduled key rotations
type Scheduler struct {
	cron     *cron.Cron
	jobs     map[string]*RotationJob
	entryIDs map[string]cron.EntryID
	mu       sync.RWMutex
	running  bool
}

// New creates a new scheduler
func New() *Scheduler {
	return &Scheduler{
		cron:     cron.New(),
		jobs:     make(map[string]*RotationJob),
		entryIDs: make(map[string]cron.EntryID),
	}
}

// AddJob adds a rotation job to the scheduler
func (s *Scheduler) AddJob(job *RotationJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[job.Name]; exists {
		return fmt.Errorf("job %q already exists", job.Name)
	}

	// Validate cron expression
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err := parser.Parse(job.Schedule)
	if err != nil {
		return fmt.Errorf("invalid cron expression %q: %w", job.Schedule, err)
	}

	s.jobs[job.Name] = job

	if s.running && job.Enabled {
		return s.scheduleJob(job)
	}

	return nil
}

// RemoveJob removes a job from the scheduler
func (s *Scheduler) RemoveJob(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[name]; !exists {
		return fmt.Errorf("job %q not found", name)
	}

	if entryID, ok := s.entryIDs[name]; ok {
		s.cron.Remove(entryID)
		delete(s.entryIDs, name)
	}

	delete(s.jobs, name)
	return nil
}

// EnableJob enables a job
func (s *Scheduler) EnableJob(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[name]
	if !exists {
		return fmt.Errorf("job %q not found", name)
	}

	job.Enabled = true

	if s.running {
		return s.scheduleJob(job)
	}

	return nil
}

// DisableJob disables a job
func (s *Scheduler) DisableJob(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[name]
	if !exists {
		return fmt.Errorf("job %q not found", name)
	}

	job.Enabled = false

	if entryID, ok := s.entryIDs[name]; ok {
		s.cron.Remove(entryID)
		delete(s.entryIDs, name)
	}

	return nil
}

// Start starts the scheduler
func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("scheduler already running")
	}

	// Schedule all enabled jobs
	for _, job := range s.jobs {
		if job.Enabled {
			if err := s.scheduleJob(job); err != nil {
				return err
			}
		}
	}

	s.cron.Start()
	s.running = true
	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.running = false
	return s.cron.Stop()
}

// scheduleJob adds a job to the cron scheduler (must hold lock)
func (s *Scheduler) scheduleJob(job *RotationJob) error {
	// Remove existing entry if any
	if entryID, ok := s.entryIDs[job.Name]; ok {
		s.cron.Remove(entryID)
	}

	entryID, err := s.cron.AddFunc(job.Schedule, func() {
		s.executeJob(job)
	})
	if err != nil {
		return fmt.Errorf("failed to schedule job %q: %w", job.Name, err)
	}

	s.entryIDs[job.Name] = entryID

	// Schedule notification if configured
	if job.NotifyBefore > 0 && job.NotifyCallback != nil {
		s.scheduleNotification(job)
	}

	return nil
}

// scheduleNotification schedules a pre-rotation notification
func (s *Scheduler) scheduleNotification(job *RotationJob) {
	entry := s.cron.Entry(s.entryIDs[job.Name])
	if entry.ID == 0 {
		return
	}

	nextRun := entry.Next
	notifyAt := nextRun.Add(-job.NotifyBefore)

	if notifyAt.After(time.Now()) {
		go func() {
			time.Sleep(time.Until(notifyAt))
			job.NotifyCallback(job, nextRun)
		}()
	}
}

// executeJob runs a rotation job
func (s *Scheduler) executeJob(job *RotationJob) {
	startedAt := time.Now()
	
	// Generate new key value
	newValue, err := job.Generator.Generate()
	if err != nil {
		audit.LogRotation(&audit.RotationEntry{
			KeyName:      job.KeyName,
			StartedAt:    startedAt,
			InitiatedBy:  "scheduler:" + job.Name,
			Status:       "failed",
			ErrorMessage: err.Error(),
		})
		return
	}

	// Execute rotation
	coord := rotation.NewCoordinator("scheduler:" + job.Name)
	tx, err := coord.Execute(job.KeyName, newValue, job.Locations)

	// Log result
	entry := &audit.RotationEntry{
		KeyName:     job.KeyName,
		StartedAt:   startedAt,
		InitiatedBy: "scheduler:" + job.Name,
	}

	if err != nil {
		entry.Status = "failed"
		entry.ErrorMessage = err.Error()
	} else {
		entry.Status = "success"
		entry.LocationsUpdated = len(tx.Locations)
	}

	audit.LogRotation(entry)
}

// ListJobs returns all registered jobs
func (s *Scheduler) ListJobs() []*RotationJob {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]*RotationJob, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

// GetJob returns a job by name
func (s *Scheduler) GetJob(name string) (*RotationJob, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, ok := s.jobs[name]
	return job, ok
}

// NextRun returns when a job will next run
func (s *Scheduler) NextRun(name string) (time.Time, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entryID, ok := s.entryIDs[name]
	if !ok {
		return time.Time{}, fmt.Errorf("job %q not scheduled", name)
	}

	entry := s.cron.Entry(entryID)
	return entry.Next, nil
}
