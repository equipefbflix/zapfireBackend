package queue

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

type fakeTopologyChannel struct {
	exchanges []ExchangeSpec
	queues    []QueueSpec
	bindings  []BindingSpec
}

func (f *fakeTopologyChannel) DeclareExchange(ctx context.Context, exchange ExchangeSpec) error {
	f.exchanges = append(f.exchanges, exchange)
	return nil
}

func (f *fakeTopologyChannel) DeclareQueue(ctx context.Context, queue QueueSpec) error {
	f.queues = append(f.queues, queue)
	return nil
}

func (f *fakeTopologyChannel) BindQueue(ctx context.Context, binding BindingSpec) error {
	f.bindings = append(f.bindings, binding)
	return nil
}

func TestDeclareTopology(t *testing.T) {
	channel := &fakeTopologyChannel{}
	topology := DefaultTopology(TopologyConfig{
		Exchange:             "aquecedor.events",
		WarmingJobsQueue:     "aquecedor.warming.jobs",
		EvolutionEventsQueue: "aquecedor.evolution.events",
		DeadLetterQueue:      "aquecedor.dead_letter",
	})

	if err := DeclareTopology(context.Background(), channel, topology); err != nil {
		t.Fatalf("DeclareTopology() error = %v", err)
	}

	if len(channel.exchanges) != 1 {
		t.Fatalf("exchanges len = %d", len(channel.exchanges))
	}
	if len(channel.queues) != 3 {
		t.Fatalf("queues len = %d", len(channel.queues))
	}
	if len(channel.bindings) != 3 {
		t.Fatalf("bindings len = %d", len(channel.bindings))
	}
	if channel.bindings[0].RoutingKey != RoutingKeyWarmingJobDue {
		t.Fatalf("first routing key = %q", channel.bindings[0].RoutingKey)
	}
}

type fakePublisher struct {
	exchange   string
	routingKey string
	body       []byte
	content    PublishContent
}

func (f *fakePublisher) Publish(ctx context.Context, exchange, routingKey string, content PublishContent) error {
	f.exchange = exchange
	f.routingKey = routingKey
	f.body = append([]byte(nil), content.Body...)
	f.content = content
	return nil
}

func TestPublisherPublishesWarmingJobDue(t *testing.T) {
	fake := &fakePublisher{}
	publisher := NewPublisher(fake, "aquecedor.events")
	msg := NewWarmingJobDueMessage("job-id", "test-run", time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC))

	if err := publisher.PublishWarmingJobDue(context.Background(), msg); err != nil {
		t.Fatalf("PublishWarmingJobDue() error = %v", err)
	}

	if fake.exchange != "aquecedor.events" {
		t.Fatalf("exchange = %q", fake.exchange)
	}
	if fake.routingKey != RoutingKeyWarmingJobDue {
		t.Fatalf("routingKey = %q", fake.routingKey)
	}
	if fake.content.ContentType != "application/json" {
		t.Fatalf("content type = %q", fake.content.ContentType)
	}
	if !fake.content.Persistent {
		t.Fatal("persistent = false")
	}

	var decoded WarmingJobDueMessage
	if err := json.Unmarshal(fake.body, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded.JobID != "job-id" {
		t.Fatalf("job id = %q", decoded.JobID)
	}
}

func TestPublisherPublishesEvolutionEventReceived(t *testing.T) {
	fake := &fakePublisher{}
	publisher := NewPublisher(fake, "aquecedor.events")
	msg := NewEvolutionEventReceivedMessage("event-id", "chip_5511999999999", "MESSAGES_UPSERT", "test-run", time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC))

	if err := publisher.PublishEvolutionEventReceived(context.Background(), msg); err != nil {
		t.Fatalf("PublishEvolutionEventReceived() error = %v", err)
	}

	if fake.routingKey != RoutingKeyEvolutionEventReceived {
		t.Fatalf("routingKey = %q", fake.routingKey)
	}

	var decoded EvolutionEventReceivedMessage
	if err := json.Unmarshal(fake.body, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded.EventID != "event-id" {
		t.Fatalf("event id = %q", decoded.EventID)
	}
}
