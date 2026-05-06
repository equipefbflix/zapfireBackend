package scheduler

import (
	"context"
	"testing"
	"time"

	"aquecedor-evolution/backend/internal/queue"
	"aquecedor-evolution/backend/internal/repository"
)

type fakeDueJobStore struct {
	now   time.Time
	limit int
	jobs  []repository.WarmingJob
}

func (s *fakeDueJobStore) ListDuePending(ctx context.Context, now time.Time, limit int) ([]repository.WarmingJob, error) {
	s.now = now
	s.limit = limit
	return s.jobs, nil
}

type fakeWarmingJobPublisher struct {
	messages []queue.WarmingJobDueMessage
}

func (p *fakeWarmingJobPublisher) PublishWarmingJobDue(ctx context.Context, msg queue.WarmingJobDueMessage) error {
	p.messages = append(p.messages, msg)
	return nil
}

func TestWarmingJobSchedulerPublishesDueJobs(t *testing.T) {
	now := time.Date(2026, 5, 4, 15, 0, 0, 0, time.UTC)
	store := &fakeDueJobStore{jobs: []repository.WarmingJob{
		{ID: "job-id", Metadata: map[string]any{"testRunId": "test-run"}},
	}}
	publisher := &fakeWarmingJobPublisher{}
	scheduler := NewWarmingJobScheduler(store, publisher, 25)

	published, err := scheduler.PublishDue(context.Background(), now)
	if err != nil {
		t.Fatalf("PublishDue() error = %v", err)
	}

	if published != 1 {
		t.Fatalf("published = %d", published)
	}
	if store.limit != 25 {
		t.Fatalf("limit = %d", store.limit)
	}
	if len(publisher.messages) != 1 {
		t.Fatalf("messages len = %d", len(publisher.messages))
	}
	if publisher.messages[0].JobID != "job-id" {
		t.Fatalf("JobID = %q", publisher.messages[0].JobID)
	}
	if publisher.messages[0].TestRunID != "test-run" {
		t.Fatalf("TestRunID = %q", publisher.messages[0].TestRunID)
	}
}
