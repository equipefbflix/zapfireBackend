package queue

import (
	"encoding/json"
	"testing"
	"time"
)

func TestWarmingJobDueMessageJSON(t *testing.T) {
	publishedAt := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	msg := WarmingJobDueMessage{
		Type:        MessageTypeWarmingJobDue,
		Version:     1,
		JobID:       "job-id",
		TestRunID:   "test-run",
		PublishedAt: publishedAt,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if decoded["type"] != "warming.job.due" {
		t.Fatalf("type = %v", decoded["type"])
	}
	if decoded["version"].(float64) != 1 {
		t.Fatalf("version = %v", decoded["version"])
	}
	if decoded["jobId"] != "job-id" {
		t.Fatalf("jobId = %v", decoded["jobId"])
	}
	if decoded["testRunId"] != "test-run" {
		t.Fatalf("testRunId = %v", decoded["testRunId"])
	}
	if decoded["publishedAt"] != "2026-04-29T12:00:00Z" {
		t.Fatalf("publishedAt = %v", decoded["publishedAt"])
	}
}

func TestEvolutionEventReceivedMessageJSON(t *testing.T) {
	publishedAt := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	msg := EvolutionEventReceivedMessage{
		Type:         MessageTypeEvolutionEventReceived,
		Version:      1,
		EventID:      "event-id",
		InstanceName: "chip_5511999999999",
		EventType:    "MESSAGES_UPSERT",
		TestRunID:    "test-run",
		PublishedAt:  publishedAt,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if decoded["type"] != "evolution.event.received" {
		t.Fatalf("type = %v", decoded["type"])
	}
	if decoded["eventId"] != "event-id" {
		t.Fatalf("eventId = %v", decoded["eventId"])
	}
	if decoded["instanceName"] != "chip_5511999999999" {
		t.Fatalf("instanceName = %v", decoded["instanceName"])
	}
	if decoded["eventType"] != "MESSAGES_UPSERT" {
		t.Fatalf("eventType = %v", decoded["eventType"])
	}
}
