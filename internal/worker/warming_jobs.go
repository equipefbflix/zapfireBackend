package worker

import (
	"context"

	"aquecedor-evolution/backend/internal/queue"
)

type WarmingJobRunner interface {
	Run(ctx context.Context, jobID string) (int, error)
}

type WarmingJobWorker struct {
	runner WarmingJobRunner
}

func NewWarmingJobWorker(runner WarmingJobRunner) WarmingJobWorker {
	return WarmingJobWorker{
		runner: runner,
	}
}

func (w WarmingJobWorker) HandleDue(ctx context.Context, msg queue.WarmingJobDueMessage) error {
	_, err := w.runner.Run(ctx, msg.JobID)
	return err
}
