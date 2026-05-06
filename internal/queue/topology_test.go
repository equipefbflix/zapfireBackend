package queue

import "testing"

func TestDefaultTopology(t *testing.T) {
	topology := DefaultTopology(TopologyConfig{
		Exchange:             "aquecedor.events",
		WarmingJobsQueue:     "aquecedor.warming.jobs",
		EvolutionEventsQueue: "aquecedor.evolution.events",
		DeadLetterQueue:      "aquecedor.dead_letter",
	})

	if topology.Exchange.Name != "aquecedor.events" {
		t.Fatalf("exchange name = %q", topology.Exchange.Name)
	}
	if topology.Exchange.Kind != "direct" {
		t.Fatalf("exchange kind = %q", topology.Exchange.Kind)
	}
	if !topology.Exchange.Durable {
		t.Fatal("exchange durable = false")
	}
	if len(topology.Queues) != 3 {
		t.Fatalf("queues len = %d", len(topology.Queues))
	}
	if len(topology.Bindings) != 3 {
		t.Fatalf("bindings len = %d", len(topology.Bindings))
	}

	var warmingQueue QueueSpec
	for _, queue := range topology.Queues {
		if queue.Name == "aquecedor.warming.jobs" {
			warmingQueue = queue
			break
		}
	}
	if warmingQueue.Name == "" {
		t.Fatal("warming queue not found")
	}
	if warmingQueue.DeadLetter {
		t.Fatal("warming queue should not be marked as dead letter")
	}
	if warmingQueue.DeadLetterExchange != "aquecedor.events" {
		t.Fatalf("warming dead-letter exchange = %q", warmingQueue.DeadLetterExchange)
	}
	if warmingQueue.DeadLetterRoutingKey != "dead_letter" {
		t.Fatalf("warming dead-letter routing key = %q", warmingQueue.DeadLetterRoutingKey)
	}

	wantRouting := map[string]string{
		"aquecedor.warming.jobs":     "warming.job.due",
		"aquecedor.evolution.events": "evolution.event.received",
		"aquecedor.dead_letter":      "dead_letter",
	}
	for _, binding := range topology.Bindings {
		if wantRouting[binding.Queue] != binding.RoutingKey {
			t.Fatalf("binding for queue %q = %q", binding.Queue, binding.RoutingKey)
		}
	}
}
