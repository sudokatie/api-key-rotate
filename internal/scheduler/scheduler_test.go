package scheduler

import (
	"testing"
	"time"
)

func TestNewScheduler(t *testing.T) {
	s := New()
	if s == nil {
		t.Fatal("expected non-nil scheduler")
	}
	if s.jobs == nil {
		t.Error("expected jobs map to be initialized")
	}
	if s.entryIDs == nil {
		t.Error("expected entryIDs map to be initialized")
	}
}

func TestAddJob(t *testing.T) {
	s := New()

	job := &RotationJob{
		Name:      "test-job",
		KeyName:   "API_KEY",
		Schedule:  "0 0 * * *", // daily at midnight
		Generator: NewStaticGenerator("test-value"),
		Enabled:   true,
	}

	err := s.AddJob(job)
	if err != nil {
		t.Fatalf("AddJob failed: %v", err)
	}

	// Verify job was added
	if _, ok := s.jobs["test-job"]; !ok {
		t.Error("job not found in jobs map")
	}
}

func TestAddJobInvalidCron(t *testing.T) {
	s := New()

	job := &RotationJob{
		Name:      "test-job",
		KeyName:   "API_KEY",
		Schedule:  "invalid cron",
		Generator: NewStaticGenerator("test-value"),
		Enabled:   true,
	}

	err := s.AddJob(job)
	if err == nil {
		t.Error("expected error for invalid cron expression")
	}
}

func TestAddJobDuplicate(t *testing.T) {
	s := New()

	job := &RotationJob{
		Name:      "test-job",
		KeyName:   "API_KEY",
		Schedule:  "0 0 * * *",
		Generator: NewStaticGenerator("test-value"),
		Enabled:   true,
	}

	_ = s.AddJob(job)
	err := s.AddJob(job)
	if err == nil {
		t.Error("expected error for duplicate job")
	}
}

func TestRemoveJob(t *testing.T) {
	s := New()

	job := &RotationJob{
		Name:      "test-job",
		KeyName:   "API_KEY",
		Schedule:  "0 0 * * *",
		Generator: NewStaticGenerator("test-value"),
		Enabled:   true,
	}

	_ = s.AddJob(job)
	err := s.RemoveJob("test-job")
	if err != nil {
		t.Fatalf("RemoveJob failed: %v", err)
	}

	if _, ok := s.jobs["test-job"]; ok {
		t.Error("job still exists after removal")
	}
}

func TestRemoveJobNotFound(t *testing.T) {
	s := New()

	err := s.RemoveJob("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent job")
	}
}

func TestEnableDisableJob(t *testing.T) {
	s := New()

	job := &RotationJob{
		Name:      "test-job",
		KeyName:   "API_KEY",
		Schedule:  "0 0 * * *",
		Generator: NewStaticGenerator("test-value"),
		Enabled:   false,
	}

	_ = s.AddJob(job)

	err := s.EnableJob("test-job")
	if err != nil {
		t.Fatalf("EnableJob failed: %v", err)
	}
	if !s.jobs["test-job"].Enabled {
		t.Error("job should be enabled")
	}

	err = s.DisableJob("test-job")
	if err != nil {
		t.Fatalf("DisableJob failed: %v", err)
	}
	if s.jobs["test-job"].Enabled {
		t.Error("job should be disabled")
	}
}

func TestListJobs(t *testing.T) {
	s := New()

	job1 := &RotationJob{
		Name:      "job1",
		KeyName:   "KEY1",
		Schedule:  "0 0 * * *",
		Generator: NewStaticGenerator("v1"),
		Enabled:   true,
	}
	job2 := &RotationJob{
		Name:      "job2",
		KeyName:   "KEY2",
		Schedule:  "0 12 * * *",
		Generator: NewStaticGenerator("v2"),
		Enabled:   true,
	}

	_ = s.AddJob(job1)
	_ = s.AddJob(job2)

	jobs := s.ListJobs()
	if len(jobs) != 2 {
		t.Errorf("expected 2 jobs, got %d", len(jobs))
	}
}

func TestGetJob(t *testing.T) {
	s := New()

	job := &RotationJob{
		Name:      "test-job",
		KeyName:   "API_KEY",
		Schedule:  "0 0 * * *",
		Generator: NewStaticGenerator("test-value"),
		Enabled:   true,
	}

	_ = s.AddJob(job)

	found, ok := s.GetJob("test-job")
	if !ok {
		t.Fatal("job not found")
	}
	if found.Name != "test-job" {
		t.Errorf("expected job name 'test-job', got %q", found.Name)
	}

	_, ok = s.GetJob("nonexistent")
	if ok {
		t.Error("expected nonexistent job to not be found")
	}
}

func TestStartStop(t *testing.T) {
	s := New()

	job := &RotationJob{
		Name:      "test-job",
		KeyName:   "API_KEY",
		Schedule:  "0 0 * * *",
		Generator: NewStaticGenerator("test-value"),
		Enabled:   true,
	}

	_ = s.AddJob(job)

	err := s.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !s.running {
		t.Error("scheduler should be running")
	}

	// Should fail to start again
	err = s.Start()
	if err == nil {
		t.Error("expected error when starting already running scheduler")
	}

	ctx := s.Stop()
	<-ctx.Done()

	if s.running {
		t.Error("scheduler should not be running")
	}
}

func TestNextRun(t *testing.T) {
	s := New()

	job := &RotationJob{
		Name:      "test-job",
		KeyName:   "API_KEY",
		Schedule:  "* * * * *", // every minute
		Generator: NewStaticGenerator("test-value"),
		Enabled:   true,
	}

	_ = s.AddJob(job)
	_ = s.Start()
	defer s.Stop()

	nextRun, err := s.NextRun("test-job")
	if err != nil {
		t.Fatalf("NextRun failed: %v", err)
	}

	// Should be within the next minute
	if nextRun.Before(time.Now()) || nextRun.After(time.Now().Add(time.Minute)) {
		t.Errorf("unexpected next run time: %v", nextRun)
	}
}

func TestNextRunNotScheduled(t *testing.T) {
	s := New()

	_, err := s.NextRun("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent job")
	}
}
