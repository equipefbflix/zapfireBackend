package scheduler

import (
	"context"
	"time"

	"aquecedor-evolution/backend/internal/queue"
	"aquecedor-evolution/backend/internal/repository"
)

type DueJobStore interface {
	ListDuePending(ctx context.Context, now time.Time, limit int) ([]repository.WarmingJob, error)
}

type WarmingJobPublisher interface {
	PublishWarmingJobDue(ctx context.Context, msg queue.WarmingJobDueMessage) error
}

type WarmingJobScheduler struct {
	jobs      DueJobStore
	publisher WarmingJobPublisher
	limit     int
}

func NewWarmingJobScheduler(jobs DueJobStore, publisher WarmingJobPublisher, limit int) WarmingJobScheduler {
	if limit <= 0 {
		limit = 50
	}
	return WarmingJobScheduler{
		jobs:      jobs,
		publisher: publisher,
		limit:     limit,
	}
}

func (s WarmingJobScheduler) PublishDue(ctx context.Context, now time.Time) (int, error) {
	jobs, err := s.jobs.ListDuePending(ctx, now, s.limit)
	if err != nil {
		return 0, err
	}

	for _, job := range jobs {
		if err := s.publisher.PublishWarmingJobDue(ctx, queue.NewWarmingJobDueMessage(job.ID, testRunID(job.Metadata), now)); err != nil {
			return 0, err
		}
	}

	return len(jobs), nil
}

func testRunID(metadata map[string]any) string {
	value, ok := metadata["testRunId"]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return text
}
