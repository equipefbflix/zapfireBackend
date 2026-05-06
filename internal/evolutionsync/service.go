package evolutionsync

import (
	"context"
	"strings"

	"aquecedor-evolution/backend/internal/repository"
)

type InstanceStore interface {
	UpdateConnectionStateByName(ctx context.Context, instanceName string, status string, lastError string) error
	GetByInstanceName(ctx context.Context, instanceName string) (repository.Instance, error)
}

type ExecutionLogStore interface {
	UpdateStatusByMessageID(ctx context.Context, messageID string, remoteJID string, status string, responsePayload map[string]any, errorText string) error
	FindPhoneNumberIDByMessageID(ctx context.Context, messageID string) (string, error)
}

type EventStore interface {
	MarkProcessed(ctx context.Context, eventID string, processingError string) error
}

type ScoreStore interface {
	Recalculate(ctx context.Context, phoneNumberID string) (float64, string, error)
}

type InboundReactor interface {
	HandleInbound(ctx context.Context, event repository.EvolutionEvent) (*repository.WarmingJob, error)
}

type Service struct {
	instances InstanceStore
	logs      ExecutionLogStore
	events    EventStore
	score     ScoreStore
	reactor   InboundReactor
}

func NewService(instances InstanceStore, logs ExecutionLogStore, events EventStore, score ScoreStore, reactor InboundReactor) Service {
	return Service{
		instances: instances,
		logs:      logs,
		events:    events,
		score:     score,
		reactor:   reactor,
	}
}

func (s Service) Sync(ctx context.Context, event repository.EvolutionEvent) error {
	switch strings.ToUpper(event.EventType) {
	case "CONNECTION_UPDATE":
		state := firstString(
			nestedString(event.Payload, "data", "state"),
			nestedString(event.Payload, "data", "connection"),
			stringField(event.Payload, "state"),
		)
		lastError := firstString(
			nestedString(event.Payload, "data", "error"),
			stringField(event.Payload, "error"),
		)
		if state != "" && event.InstanceName != "" {
			if err := s.instances.UpdateConnectionStateByName(ctx, event.InstanceName, mapConnectionState(state), lastError); err != nil {
				_ = s.events.MarkProcessed(ctx, event.ID, err.Error())
				return err
			}
		}
	case "MESSAGES_UPDATE", "MESSAGES_UPSERT", "SEND_MESSAGE":
		messageID := firstString(
			nestedString(event.Payload, "data", "key", "id"),
			nestedString(event.Payload, "data", "message", "key", "id"),
			nestedString(event.Payload, "key", "id"),
		)
		remoteJID := firstString(
			nestedString(event.Payload, "data", "key", "remoteJid"),
			nestedString(event.Payload, "key", "remoteJid"),
		)
		status := mapMessageStatus(firstString(
			nestedString(event.Payload, "data", "status"),
			stringField(event.Payload, "status"),
		))
		errorText := firstString(
			nestedString(event.Payload, "data", "error"),
			stringField(event.Payload, "error"),
		)
		if messageID != "" {
			if err := s.logs.UpdateStatusByMessageID(ctx, messageID, remoteJID, status, event.Payload, errorText); err != nil {
				_ = s.events.MarkProcessed(ctx, event.ID, err.Error())
				return err
			}
			if s.score != nil {
				phoneNumberID, err := s.logs.FindPhoneNumberIDByMessageID(ctx, messageID)
				if err != nil {
					_ = s.events.MarkProcessed(ctx, event.ID, err.Error())
					return err
				}
				if phoneNumberID != "" {
					if _, _, err := s.score.Recalculate(ctx, phoneNumberID); err != nil {
						_ = s.events.MarkProcessed(ctx, event.ID, err.Error())
						return err
					}
				}
			}
			if s.reactor != nil && strings.ToUpper(event.EventType) == "MESSAGES_UPSERT" {
				if _, err := s.reactor.HandleInbound(ctx, event); err != nil {
					_ = s.events.MarkProcessed(ctx, event.ID, err.Error())
					return err
				}
			}
		}
	}

	return s.events.MarkProcessed(ctx, event.ID, "")
}

func mapConnectionState(state string) string {
	switch strings.ToLower(state) {
	case "open", "opened":
		return "open"
	case "close", "closed", "disconnected":
		return "close"
	case "connecting":
		return "connecting"
	case "failed", "error":
		return "failed"
	case "paused":
		return "paused"
	default:
		return strings.ToLower(state)
	}
}

func mapMessageStatus(status string) string {
	switch strings.ToLower(status) {
	case "error", "failed":
		return "failed"
	default:
		return "success"
	}
}

func nestedString(payload map[string]any, keys ...string) string {
	current := any(payload)
	for _, key := range keys {
		m, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current, ok = m[key]
		if !ok {
			return ""
		}
	}
	text, _ := current.(string)
	return text
}

func firstString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func stringField(payload map[string]any, key string) string {
	value, _ := payload[key].(string)
	return value
}
