package queue

import (
	"context"
	"encoding/json"
	"fmt"
)

type WarmingJobDueHandler interface {
	HandleDue(ctx context.Context, msg WarmingJobDueMessage) error
}

type WarmingJobDueConsumer struct {
	handler WarmingJobDueHandler
}

func NewWarmingJobDueConsumer(handler WarmingJobDueHandler) WarmingJobDueConsumer {
	return WarmingJobDueConsumer{handler: handler}
}

func (c WarmingJobDueConsumer) Handle(ctx context.Context, body []byte) error {
	var msg WarmingJobDueMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return fmt.Errorf("decode warming job message: %w", err)
	}
	if msg.Type != MessageTypeWarmingJobDue {
		return fmt.Errorf("invalid message type %q", msg.Type)
	}
	if msg.Version != 1 {
		return fmt.Errorf("invalid message version %d", msg.Version)
	}
	if msg.JobID == "" {
		return fmt.Errorf("jobId is required")
	}
	return c.handler.HandleDue(ctx, msg)
}
