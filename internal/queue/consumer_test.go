package queue

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

type fakeWarmingJobDueHandler struct {
	message WarmingJobDueMessage
	called  bool
}

func (h *fakeWarmingJobDueHandler) HandleDue(ctx context.Context, msg WarmingJobDueMessage) error {
	h.called = true
	h.message = msg
	return nil
}

func TestWarmingJobDueConsumerHandlesValidMessage(t *testing.T) {
	handler := &fakeWarmingJobDueHandler{}
	consumer := NewWarmingJobDueConsumer(handler)
	body, err := json.Marshal(NewWarmingJobDueMessage("job-id", "test-run", time.Date(2026, 5, 4, 15, 0, 0, 0, time.UTC)))
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	if err := consumer.Handle(context.Background(), body); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if !handler.called {
		t.Fatal("handler called = false")
	}
	if handler.message.JobID != "job-id" {
		t.Fatalf("JobID = %q", handler.message.JobID)
	}
}

func TestWarmingJobDueConsumerRejectsWrongType(t *testing.T) {
	consumer := NewWarmingJobDueConsumer(&fakeWarmingJobDueHandler{})
	body := []byte(`{"type":"unknown","version":1,"jobId":"job-id","publishedAt":"2026-05-04T15:00:00Z"}`)

	if err := consumer.Handle(context.Background(), body); err == nil {
		t.Fatal("Handle() error = nil, want error")
	}
}

func TestWarmingJobDueConsumerRejectsMissingJobID(t *testing.T) {
	consumer := NewWarmingJobDueConsumer(&fakeWarmingJobDueHandler{})
	body := []byte(`{"type":"warming.job.due","version":1,"publishedAt":"2026-05-04T15:00:00Z"}`)

	if err := consumer.Handle(context.Background(), body); err == nil {
		t.Fatal("Handle() error = nil, want error")
	}
}
