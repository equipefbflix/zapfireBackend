//go:build integration

package queue

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestRabbitMQPublishConsumeReal(t *testing.T) {
	if os.Getenv("ENABLE_REAL_TESTS") != "true" {
		t.Skip("set ENABLE_REAL_TESTS=true to run real RabbitMQ integration tests")
	}

	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		t.Fatal("RABBITMQ_URL is required")
	}

	testRunID := os.Getenv("TEST_RUN_ID")
	if testRunID == "" {
		testRunID = "manual-" + time.Now().UTC().Format("20060102T150405")
	}

	exchange := "aquecedor.test." + testRunID + ".events"
	warmingQueue := "aquecedor.test." + testRunID + ".warming.jobs"
	eventsQueue := "aquecedor.test." + testRunID + ".evolution.events"
	deadLetterQueue := "aquecedor.test." + testRunID + ".dead_letter"

	broker, err := DialRabbitMQ(url)
	if err != nil {
		t.Fatalf("DialRabbitMQ() error = %v", err)
	}
	defer broker.Close()

	defer func() {
		_, _ = broker.channel.QueueDelete(warmingQueue, false, false, false)
		_, _ = broker.channel.QueueDelete(eventsQueue, false, false, false)
		_, _ = broker.channel.QueueDelete(deadLetterQueue, false, false, false)
		_ = broker.channel.ExchangeDelete(exchange, false, false)
	}()

	topology := DefaultTopology(TopologyConfig{
		Exchange:             exchange,
		WarmingJobsQueue:     warmingQueue,
		EvolutionEventsQueue: eventsQueue,
		DeadLetterQueue:      deadLetterQueue,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := DeclareTopology(ctx, broker, topology); err != nil {
		t.Fatalf("DeclareTopology() error = %v", err)
	}

	deliveries, err := broker.channel.Consume(
		warmingQueue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		t.Fatalf("Consume() error = %v", err)
	}

	publisher := NewPublisher(broker, exchange)
	msg := NewWarmingJobDueMessage("test-job-id", testRunID, time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC))
	if err := publisher.PublishWarmingJobDue(ctx, msg); err != nil {
		t.Fatalf("PublishWarmingJobDue() error = %v", err)
	}

	select {
	case delivery := <-deliveries:
		var decoded WarmingJobDueMessage
		if err := json.Unmarshal(delivery.Body, &decoded); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}
		if decoded.JobID != "test-job-id" {
			t.Fatalf("job id = %q", decoded.JobID)
		}
		if decoded.TestRunID != testRunID {
			t.Fatalf("testRunId = %q", decoded.TestRunID)
		}
		if err := delivery.Ack(false); err != nil {
			t.Fatalf("Ack() error = %v", err)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for RabbitMQ delivery")
	}
}

