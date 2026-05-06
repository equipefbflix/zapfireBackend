package queue

import "time"

const (
	MessageTypeWarmingJobDue          = "warming.job.due"
	MessageTypeEvolutionEventReceived = "evolution.event.received"
)

type WarmingJobDueMessage struct {
	Type        string    `json:"type"`
	Version     int       `json:"version"`
	JobID       string    `json:"jobId"`
	TestRunID   string    `json:"testRunId,omitempty"`
	PublishedAt time.Time `json:"publishedAt"`
}

type EvolutionEventReceivedMessage struct {
	Type         string    `json:"type"`
	Version      int       `json:"version"`
	EventID      string    `json:"eventId"`
	InstanceName string    `json:"instanceName"`
	EventType    string    `json:"eventType"`
	TestRunID    string    `json:"testRunId,omitempty"`
	PublishedAt  time.Time `json:"publishedAt"`
}

func NewWarmingJobDueMessage(jobID, testRunID string, publishedAt time.Time) WarmingJobDueMessage {
	return WarmingJobDueMessage{
		Type:        MessageTypeWarmingJobDue,
		Version:     1,
		JobID:       jobID,
		TestRunID:   testRunID,
		PublishedAt: publishedAt.UTC(),
	}
}

func NewEvolutionEventReceivedMessage(eventID, instanceName, eventType, testRunID string, publishedAt time.Time) EvolutionEventReceivedMessage {
	return EvolutionEventReceivedMessage{
		Type:         MessageTypeEvolutionEventReceived,
		Version:      1,
		EventID:      eventID,
		InstanceName: instanceName,
		EventType:    eventType,
		TestRunID:    testRunID,
		PublishedAt:  publishedAt.UTC(),
	}
}
