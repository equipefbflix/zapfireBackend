package observability

import (
	"context"
	"time"

	"aquecedor-evolution/backend/internal/config"
)

type JobMetricsStore interface {
	CountByStatus(ctx context.Context) (map[string]int, error)
	CountStaleRunning(ctx context.Context, olderThan time.Time) (int, error)
}

type ExecutionLogMetricsStore interface {
	CountFailuresSince(ctx context.Context, since time.Time) (int, error)
}

type EvolutionEventMetricsStore interface {
	CountSince(ctx context.Context, since time.Time) (int, error)
}

type Snapshot struct {
	LookbackWindowMinutes   int            `json:"lookbackWindowMinutes"`
	StaleRunningAfterMinutes int           `json:"staleRunningAfterMinutes"`
	JobStatusCounts         map[string]int `json:"jobStatusCounts"`
	StaleRunningJobs        int            `json:"staleRunningJobs"`
	ExecutionFailures       int            `json:"executionFailures"`
	EvolutionEvents         int            `json:"evolutionEvents"`
}

type Service struct {
	cfg    config.ObservabilityConfig
	jobs   JobMetricsStore
	logs   ExecutionLogMetricsStore
	events EvolutionEventMetricsStore
}

func NewService(cfg config.ObservabilityConfig, jobs JobMetricsStore, logs ExecutionLogMetricsStore, events EvolutionEventMetricsStore) Service {
	return Service{cfg: cfg, jobs: jobs, logs: logs, events: events}
}

func (s Service) Snapshot(ctx context.Context) (Snapshot, error) {
	since := time.Now().Add(-s.cfg.LookbackWindow)
	staleBefore := time.Now().Add(-s.cfg.StaleRunningAfter)

	jobCounts, err := s.jobs.CountByStatus(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	staleRunningJobs, err := s.jobs.CountStaleRunning(ctx, staleBefore)
	if err != nil {
		return Snapshot{}, err
	}
	executionFailures, err := s.logs.CountFailuresSince(ctx, since)
	if err != nil {
		return Snapshot{}, err
	}
	evolutionEvents, err := s.events.CountSince(ctx, since)
	if err != nil {
		return Snapshot{}, err
	}

	return Snapshot{
		LookbackWindowMinutes:    int(s.cfg.LookbackWindow / time.Minute),
		StaleRunningAfterMinutes: int(s.cfg.StaleRunningAfter / time.Minute),
		JobStatusCounts:          jobCounts,
		StaleRunningJobs:         staleRunningJobs,
		ExecutionFailures:        executionFailures,
		EvolutionEvents:          evolutionEvents,
	}, nil
}

type StaleJobCleaner interface {
	FailStaleRunning(ctx context.Context, olderThan time.Time, reason string) (int64, error)
}

type StaleCleanupService struct {
	cfg  config.ObservabilityConfig
	jobs StaleJobCleaner
}

func NewStaleCleanupService(cfg config.ObservabilityConfig, jobs StaleJobCleaner) StaleCleanupService {
	return StaleCleanupService{cfg: cfg, jobs: jobs}
}

func (s StaleCleanupService) Cleanup(ctx context.Context) (int64, error) {
	return s.jobs.FailStaleRunning(ctx, time.Now().Add(-s.cfg.StaleRunningAfter), s.cfg.StaleCleanupReason)
}
