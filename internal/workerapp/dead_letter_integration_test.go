//go:build integration

package workerapp

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"aquecedor-evolution/backend/internal/queue"
)

type integrationDelivery struct {
	body    []byte
	attempt int
	nackFn  func(requeue bool) error
	ackFn   func() error
}

func (d integrationDelivery) Body() []byte      { return d.body }
func (d integrationDelivery) Attempt() int      { return d.attempt }
func (d integrationDelivery) Ack() error        { return d.ackFn() }
func (d integrationDelivery) Nack(requeue bool) error { return d.nackFn(requeue) }

type errorHandler struct{}

func (errorHandler) Handle(ctx context.Context, body []byte) error {
	return errors.New("boom")
}

func TestProcessDeliveryDeadLettersWhenRetriesExceededRealRabbit(t *testing.T) {
	if os.Getenv("ENABLE_REAL_TESTS") != "true" {
		t.Skip("set ENABLE_REAL_TESTS=true to run real RabbitMQ integration tests")
	}
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		t.Fatal("RABBITMQ_URL is required")
	}

	testRunID := os.Getenv("TEST_RUN_ID")
	if testRunID == "" {
		testRunID = "dead-letter-" + time.Now().UTC().Format("20060102T150405")
	}

	exchange := "aquecedor.test." + testRunID + ".events"
	warmingQueue := "aquecedor.test." + testRunID + ".warming.jobs"
	eventsQueue := "aquecedor.test." + testRunID + ".evolution.events"
	deadLetterQueue := "aquecedor.test." + testRunID + ".dead_letter"

	broker, err := queue.DialRabbitMQ(url)
	if err != nil {
		t.Fatalf("DialRabbitMQ() error = %v", err)
	}
	defer broker.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	topology := queue.DefaultTopology(queue.TopologyConfig{
		Exchange:             exchange,
		WarmingJobsQueue:     warmingQueue,
		EvolutionEventsQueue: eventsQueue,
		DeadLetterQueue:      deadLetterQueue,
	})
	if err := queue.DeclareTopology(ctx, broker, topology); err != nil {
		t.Fatalf("DeclareTopology() error = %v", err)
	}

	msg := queue.NewWarmingJobDueMessage("job-id", testRunID, time.Now().UTC())
	publisher := queue.NewPublisher(broker, exchange)
	if err := publisher.PublishWarmingJobDue(ctx, msg); err != nil {
		t.Fatalf("PublishWarmingJobDue() error = %v", err)
	}

	deliveries, err := broker.Consume(warmingQueue)
	if err != nil {
		t.Fatalf("Consume(warmingQueue) error = %v", err)
	}

	select {
	case delivery := <-deliveries:
		err := ProcessDelivery(ctx, errorHandler{}, integrationDelivery{
			body:    delivery.Body,
			attempt: 4,
			ackFn:   func() error { return delivery.Ack(false) },
			nackFn:  func(requeue bool) error { return delivery.Nack(false, requeue) },
		}, 3)
		if err == nil {
			t.Fatal("ProcessDelivery() error = nil")
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for warming queue delivery")
	}

	deadDeliveries, err := broker.Consume(deadLetterQueue)
	if err != nil {
		t.Fatalf("Consume(deadLetterQueue) error = %v", err)
	}

	select {
	case dead := <-deadDeliveries:
		if len(dead.Body) == 0 {
			t.Fatal("dead-letter body is empty")
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for dead-letter delivery")
	}
}
