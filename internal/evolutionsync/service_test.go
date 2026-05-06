package evolutionsync

import (
	"context"
	"testing"

	"aquecedor-evolution/backend/internal/repository"
)

type fakeInstanceStore struct {
	instanceName   string
	status         string
	lastError      string
	instanceByName repository.Instance
	lookupName     string
}

func (s *fakeInstanceStore) UpdateConnectionStateByName(ctx context.Context, instanceName string, status string, lastError string) error {
	s.instanceName = instanceName
	s.status = status
	s.lastError = lastError
	return nil
}

func (s *fakeInstanceStore) GetByInstanceName(ctx context.Context, instanceName string) (repository.Instance, error) {
	s.lookupName = instanceName
	return s.instanceByName, nil
}

type fakeExecutionLogStore struct {
	messageID       string
	remoteJID       string
	status          string
	responsePayload map[string]any
	errorText       string
	phoneNumberID   string
}

func (s *fakeExecutionLogStore) UpdateStatusByMessageID(ctx context.Context, messageID string, remoteJID string, status string, responsePayload map[string]any, errorText string) error {
	s.messageID = messageID
	s.remoteJID = remoteJID
	s.status = status
	s.responsePayload = responsePayload
	s.errorText = errorText
	return nil
}

func (s *fakeExecutionLogStore) FindPhoneNumberIDByMessageID(ctx context.Context, messageID string) (string, error) {
	return s.phoneNumberID, nil
}

type fakeEventStore struct {
	processedID   string
	processingErr string
}

func (s *fakeEventStore) MarkProcessed(ctx context.Context, eventID string, processingError string) error {
	s.processedID = eventID
	s.processingErr = processingError
	return nil
}

type fakeScoreStore struct {
	phoneID string
}

func (s *fakeScoreStore) Recalculate(ctx context.Context, phoneNumberID string) (float64, string, error) {
	s.phoneID = phoneNumberID
	return 10, "warming", nil
}

type fakeInboundReactor struct {
	event repository.EvolutionEvent
}

func (s *fakeInboundReactor) HandleInbound(ctx context.Context, event repository.EvolutionEvent) (*repository.WarmingJob, error) {
	s.event = event
	return &repository.WarmingJob{ID: "job-1"}, nil
}

func TestServiceSyncConnectionUpdate(t *testing.T) {
	instances := &fakeInstanceStore{}
	logs := &fakeExecutionLogStore{}
	events := &fakeEventStore{}
	score := &fakeScoreStore{}
	service := NewService(instances, logs, events, score, nil)

	event := repository.EvolutionEvent{
		ID:           "event-id",
		InstanceName: "chip-sp-01",
		EventType:    "CONNECTION_UPDATE",
		Payload: map[string]any{
			"data": map[string]any{
				"state": "open",
			},
		},
	}

	if err := service.Sync(context.Background(), event); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if instances.instanceName != "chip-sp-01" {
		t.Fatalf("instanceName = %q", instances.instanceName)
	}
	if instances.status != "open" {
		t.Fatalf("status = %q", instances.status)
	}
	if events.processedID != "event-id" {
		t.Fatalf("processedID = %q", events.processedID)
	}
	if logs.messageID != "" {
		t.Fatalf("unexpected log update = %q", logs.messageID)
	}
}

func TestServiceSyncMessageUpdate(t *testing.T) {
	instances := &fakeInstanceStore{instanceByName: repository.Instance{PhoneNumberID: "phone-id"}}
	logs := &fakeExecutionLogStore{phoneNumberID: "phone-id"}
	events := &fakeEventStore{}
	score := &fakeScoreStore{}
	service := NewService(instances, logs, events, score, nil)

	event := repository.EvolutionEvent{
		ID:           "event-id",
		InstanceName: "chip-sp-01",
		EventType:    "MESSAGES_UPDATE",
		Payload: map[string]any{
			"data": map[string]any{
				"key": map[string]any{
					"id":        "message-id",
					"remoteJid": "5511999999999@s.whatsapp.net",
				},
				"status": "delivered",
			},
		},
	}

	if err := service.Sync(context.Background(), event); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if logs.messageID != "message-id" {
		t.Fatalf("messageID = %q", logs.messageID)
	}
	if logs.remoteJID != "5511999999999@s.whatsapp.net" {
		t.Fatalf("remoteJID = %q", logs.remoteJID)
	}
	if logs.status != "success" {
		t.Fatalf("status = %q", logs.status)
	}
	if events.processedID != "event-id" {
		t.Fatalf("processedID = %q", events.processedID)
	}
	if score.phoneID != "phone-id" {
		t.Fatalf("score phoneID = %q", score.phoneID)
	}
}

func TestServiceSyncMessageUpsertCallsInboundReactor(t *testing.T) {
	instances := &fakeInstanceStore{instanceByName: repository.Instance{PhoneNumberID: "phone-id"}}
	logs := &fakeExecutionLogStore{}
	events := &fakeEventStore{}
	score := &fakeScoreStore{}
	reactor := &fakeInboundReactor{}
	service := NewService(instances, logs, events, score, reactor)

	event := repository.EvolutionEvent{
		ID:           "event-id",
		InstanceName: "chip-sp-01",
		EventType:    "MESSAGES_UPSERT",
		Payload: map[string]any{
			"data": map[string]any{
				"key": map[string]any{
					"id":        "message-id",
					"fromMe":    false,
					"remoteJid": "5511999999999@s.whatsapp.net",
				},
			},
		},
	}

	if err := service.Sync(context.Background(), event); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if reactor.event.ID != "event-id" {
		t.Fatalf("reactor event ID = %q", reactor.event.ID)
	}
}

func TestServiceSyncMessageFailure(t *testing.T) {
	events := &fakeEventStore{}
	score := &fakeScoreStore{}
	service := NewService(&fakeInstanceStore{instanceByName: repository.Instance{PhoneNumberID: "phone-id"}}, &fakeExecutionLogStore{phoneNumberID: "phone-id"}, events, score, nil)

	event := repository.EvolutionEvent{
		ID:           "event-id",
		InstanceName: "chip-sp-01",
		EventType:    "MESSAGES_UPDATE",
		Payload: map[string]any{
			"data": map[string]any{
				"key": map[string]any{
					"id": "message-id",
				},
				"status": "ERROR",
				"error":  "delivery failed",
			},
		},
	}

	if err := service.Sync(context.Background(), event); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if events.processingErr != "" {
		t.Fatalf("processingErr = %q", events.processingErr)
	}
	if score.phoneID != "phone-id" {
		t.Fatalf("score phoneID = %q", score.phoneID)
	}
}
