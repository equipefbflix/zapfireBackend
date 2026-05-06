package observability

import (
	"context"
	"testing"
	"time"

	"aquecedor-evolution/backend/internal/config"
)

type fakeJobMetricsStore struct {
	counts map[string]int
	stale  int
}

func (s *fakeJobMetricsStore) CountByStatus(ctx context.Context) (map[string]int, error) {
	return s.counts, nil
}

func (s *fakeJobMetricsStore) CountStaleRunning(ctx context.Context, olderThan time.Time) (int, error) {
	return s.stale, nil
}

type fakeExecutionLogMetricsStore struct{ count int }

func (s *fakeExecutionLogMetricsStore) CountFailuresSince(ctx context.Context, since time.Time) (int, error) {
	return s.count, nil
}

type fakeEventMetricsStore struct{ count int }

func (s *fakeEventMetricsStore) CountSince(ctx context.Context, since time.Time) (int, error) {
	return s.count, nil
}

type fakeStaleCleaner struct {
	affected int64
	reason   string
}

func (s *fakeStaleCleaner) FailStaleRunning(ctx context.Context, olderThan time.Time, reason string) (int64, error) {
	s.reason = reason
	return s.affected, nil
}

func TestServiceSnapshot(t *testing.T) {
	service := NewService(
		config.ObservabilityConfig{
			LookbackWindow:    30 * time.Minute,
			StaleRunningAfter: 15 * time.Minute,
		},
		&fakeJobMetricsStore{counts: map[string]int{"pending": 2, "running": 1}, stale: 1},
		&fakeExecutionLogMetricsStore{count: 3},
		&fakeEventMetricsStore{count: 9},
	)

	snapshot, err := service.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	if snapshot.JobStatusCounts["pending"] != 2 {
		t.Fatalf("pending = %d", snapshot.JobStatusCounts["pending"])
	}
	if snapshot.StaleRunningJobs != 1 {
		t.Fatalf("stale = %d", snapshot.StaleRunningJobs)
	}
	if snapshot.ExecutionFailures != 3 {
		t.Fatalf("execution failures = %d", snapshot.ExecutionFailures)
	}
	if snapshot.EvolutionEvents != 9 {
		t.Fatalf("evolution events = %d", snapshot.EvolutionEvents)
	}
}

func TestStaleCleanupServiceCleanup(t *testing.T) {
	cleaner := &fakeStaleCleaner{affected: 4}
	service := NewStaleCleanupService(
		config.ObservabilityConfig{
			StaleRunningAfter: 20 * time.Minute,
			StaleCleanupReason: "cleanup test",
		},
		cleaner,
	)

	affected, err := service.Cleanup(context.Background())
	if err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	if affected != 4 {
		t.Fatalf("affected = %d", affected)
	}
	if cleaner.reason != "cleanup test" {
		t.Fatalf("reason = %q", cleaner.reason)
	}
}
