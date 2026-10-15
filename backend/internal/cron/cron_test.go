package cron

import (
	"context"
	"testing"
	"time"
)

func TestTickRunsOncePerMinuteAndFiresFutureOneShot(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s.Register("backup_site", func(context.Context, Job) (string, error) {
		return "ok", nil
	})
	minute, err := s.Create(Job{Name: "minute", Type: "backup_site", Cron: "* * * * *", Payload: map[string]string{}})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 14, 3, 15, 5, 0, time.UTC)
	s.tick(now)
	if err := s.Reload(); err != nil {
		t.Fatal(err)
	}
	s.tick(now.Add(20 * time.Second))
	s.tick(now.Add(40 * time.Second))
	if got := runCount(t, s, minute.ID); got != 1 {
		t.Fatalf("runs in one minute = %d", got)
	}
	s.tick(now.Add(time.Minute))
	if got := runCount(t, s, minute.ID); got != 2 {
		t.Fatalf("runs after the next minute = %d", got)
	}

	wall := time.Now().UTC().Truncate(time.Second)
	future := wall.Add(2 * time.Hour)
	later, err := s.Create(Job{Name: "later", Type: "backup_site", Once: true, RunAt: future.Format(time.RFC3339), Payload: map[string]string{}})
	if err != nil {
		t.Fatal(err)
	}
	s.tick(future.Add(-time.Minute))
	if got := runCount(t, s, later.ID); got != 0 {
		t.Fatalf("future one-shot fired early: %d", got)
	}
	s.tick(future)
	s.tick(future.Add(20 * time.Second))
	if got := runCount(t, s, later.ID); got != 1 {
		t.Fatalf("future one-shot runs = %d", got)
	}

	past := wall.Add(-2 * time.Hour)
	missed, err := s.Create(Job{Name: "old", Type: "backup_site", Once: true, RunAt: past.Format(time.RFC3339), Payload: map[string]string{}})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Reload(); err != nil {
		t.Fatal(err)
	}
	s.tick(time.Now())
	job := findJob(t, s, missed.ID)
	if !job.Missed || len(job.Runs) != 0 {
		t.Fatalf("missed one-shot replayed: %+v", job)
	}
	if got := runCount(t, s, later.ID); got != 1 {
		t.Fatalf("fired one-shot changed after reload: %d", got)
	}
}

func findJob(t *testing.T, s *Scheduler, id string) Job {
	t.Helper()
	for _, job := range s.List() {
		if job.ID == id {
			return job
		}
	}
	t.Fatalf("missing job %s", id)
	return Job{}
}

func runCount(t *testing.T, s *Scheduler, id string) int {
	t.Helper()
	return len(findJob(t, s, id).Runs)
}
