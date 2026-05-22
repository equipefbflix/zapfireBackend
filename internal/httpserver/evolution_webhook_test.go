package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"aquecedor-evolution/backend/internal/config"
	"aquecedor-evolution/backend/internal/repository"
)

type fakeHTTPEvolutionEventStore struct {
	createParams repository.CreateEvolutionEventParams
}

func (s *fakeHTTPEvolutionEventStore) Create(ctx context.Context, params repository.CreateEvolutionEventParams) (repository.EvolutionEvent, error) {
	s.createParams = params
	return repository.EvolutionEvent{
		ID:           "event-id",
		InstanceName: params.InstanceName,
		EventType:    params.EventType,
		Payload:      params.Payload,
		ReceivedAt:   time.Date(2026, 5, 4, 15, 0, 0, 0, time.UTC),
	}, nil
}

func TestEvolutionWebhookRoute(t *testing.T) {
	store := &fakeHTTPEvolutionEventStore{}
	server := NewServer(ServerConfig{
		App:             config.AppConfig{WebhookEvolutionSecret: "shared-secret"},
		EvolutionEvents: store,
	})
	body := []byte(`{"event":"messages.upsert","instance":"chip-sp-01","data":{"key":{"id":"message-id"}}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/evolution", bytes.NewReader(body))
	req.Header.Set("X-Webhook-Secret", "shared-secret")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if store.createParams.EventType != "MESSAGES_UPSERT" {
		t.Fatalf("EventType = %q", store.createParams.EventType)
	}
	if store.createParams.InstanceName != "chip-sp-01" {
		t.Fatalf("InstanceName = %q", store.createParams.InstanceName)
	}

	var response evolutionWebhookResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.ID != "event-id" {
		t.Fatalf("ID = %q", response.ID)
	}
}

func TestEvolutionWebhookRouteRejectsInvalidSecret(t *testing.T) {
	server := NewServer(ServerConfig{
		App:             config.AppConfig{WebhookEvolutionSecret: "shared-secret"},
		EvolutionEvents: &fakeHTTPEvolutionEventStore{},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/evolution", bytes.NewReader([]byte(`{"event":"messages.upsert"}`)))
	req.Header.Set("X-Webhook-Secret", "wrong")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestEvolutionWebhookRouteRequiresEventType(t *testing.T) {
	server := NewServer(ServerConfig{EvolutionEvents: &fakeHTTPEvolutionEventStore{}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/evolution", bytes.NewReader([]byte(`{"instance":"chip-sp-01"}`)))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestEvolutionWebhookRouteAcceptsByEventsSubpath(t *testing.T) {
	store := &fakeHTTPEvolutionEventStore{}
	server := NewServer(ServerConfig{
		EvolutionEvents: store,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/evolution/connection-update", bytes.NewReader([]byte(`{"instance":"chip-sp-01","data":{"state":"open"}}`)))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if store.createParams.EventType != "CONNECTION_UPDATE" {
		t.Fatalf("EventType = %q", store.createParams.EventType)
	}
}

func TestEvolutionWebhookRouteAcceptsLegacyPath(t *testing.T) {
	store := &fakeHTTPEvolutionEventStore{}
	server := NewServer(ServerConfig{
		EvolutionEvents: store,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/webhook/evolution", bytes.NewReader([]byte(`{"event":"CONNECTION_UPDATE","instance":"chip-sp-01","data":{"state":"open"}}`)))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if store.createParams.EventType != "CONNECTION_UPDATE" {
		t.Fatalf("EventType = %q", store.createParams.EventType)
	}
	if store.createParams.InstanceName != "chip-sp-01" {
		t.Fatalf("InstanceName = %q", store.createParams.InstanceName)
	}
}
