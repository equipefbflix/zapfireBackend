package httpserver

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"aquecedor-evolution/backend/internal/repository"
)

type fakeWebhookSyncService struct {
	event repository.EvolutionEvent
}

func (s *fakeWebhookSyncService) Sync(ctx context.Context, event repository.EvolutionEvent) error {
	s.event = event
	return nil
}

func TestEvolutionWebhookRouteTriggersSync(t *testing.T) {
	store := &fakeHTTPEvolutionEventStore{}
	syncer := &fakeWebhookSyncService{}
	server := NewServer(ServerConfig{
		EvolutionEvents: store,
		EvolutionSync:   syncer,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/evolution", bytes.NewReader([]byte(`{"event":"CONNECTION_UPDATE","instance":"chip-sp-01","data":{"state":"open"}}`)))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if syncer.event.ID != "event-id" {
		t.Fatalf("event id = %q", syncer.event.ID)
	}
	if syncer.event.EventType != "CONNECTION_UPDATE" {
		t.Fatalf("event type = %q", syncer.event.EventType)
	}
}

func TestEvolutionWebhookLegacyRouteTriggersSync(t *testing.T) {
	store := &fakeHTTPEvolutionEventStore{}
	syncer := &fakeWebhookSyncService{}
	server := NewServer(ServerConfig{
		EvolutionEvents: store,
		EvolutionSync:   syncer,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/webhook/evolution", bytes.NewReader([]byte(`{"event":"CONNECTION_UPDATE","instance":"chip-sp-01","data":{"state":"open"}}`)))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if syncer.event.ID != "event-id" {
		t.Fatalf("event id = %q", syncer.event.ID)
	}
	if syncer.event.EventType != "CONNECTION_UPDATE" {
		t.Fatalf("event type = %q", syncer.event.EventType)
	}
}
