package worker

import (
	"context"
	"testing"
	"time"

	"aquecedor-evolution/backend/internal/queue"
)

type fakeWorkerRunner struct {
	jobID string
	count int
}

func (r *fakeWorkerRunner) Run(ctx context.Context, jobID string) (int, error) {
	r.jobID = jobID
	r.count = 2
	return r.count, nil
}

func TestWarmingJobWorkerHandlesDueMessage(t *testing.T) {
	runner := &fakeWorkerRunner{}
	worker := NewWarmingJobWorker(runner)

	err := worker.HandleDue(context.Background(), queue.NewWarmingJobDueMessage("job-id", "test-run", time.Date(2026, 5, 4, 15, 1, 0, 0, time.UTC)))
	if err != nil {
		t.Fatalf("HandleDue() error = %v", err)
	}

	if runner.jobID != "job-id" {
		t.Fatalf("job id = %q", runner.jobID)
	}
	if runner.count != 2 {
		t.Fatalf("count = %d", runner.count)
	}
}
