package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"aquecedor-evolution/backend/internal/repository"
)

type fakeHTTPExecutionLogStore struct {
	createParams repository.CreateExecutionLogParams
	items        []repository.ExecutionLog
}

func (s *fakeHTTPExecutionLogStore) Create(ctx context.Context, params repository.CreateExecutionLogParams) (repository.ExecutionLog, error) {
	s.createParams = params
	return repository.ExecutionLog{
		ID:                  "log-id",
		WarmingJobID:        params.WarmingJobID,
		InstanceID:          params.InstanceID,
		ActionType:          params.ActionType,
		Status:              params.Status,
		RequestPayload:      params.RequestPayload,
		ResponsePayload:     params.ResponsePayload,
		EvolutionMessageKey: params.EvolutionMessageKey,
		RemoteJID:           params.RemoteJID,
		Error:               params.Error,
		DurationMs:          params.DurationMs,
		CreatedAt:           time.Date(2026, 5, 4, 15, 0, 0, 0, time.UTC),
	}, nil
}

func (s *fakeHTTPExecutionLogStore) List(ctx context.Context) ([]repository.ExecutionLog, error) {
	return s.items, nil
}

func TestCreateExecutionLogRoute(t *testing.T) {
	store := &fakeHTTPExecutionLogStore{}
	server := NewServer(ServerConfig{ExecutionLogs: store})
	body := []byte(`{
		"warmingJobId": "job-id",
		"instanceId": "instance-id",
		"actionType": "send_text",
		"status": "success",
		"requestPayload": {"text":"Bom dia"},
		"responsePayload": {"messageId":"abc"},
		"evolutionMessageKey": {"id":"abc"},
		"remoteJid": "5511999999999@s.whatsapp.net",
		"durationMs": 120
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/execution-logs", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if store.createParams.RequestPayload["text"] != "Bom dia" {
		t.Fatalf("RequestPayload text = %v", store.createParams.RequestPayload["text"])
	}

	var response executionLogResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.ID != "log-id" {
		t.Fatalf("ID = %q", response.ID)
	}
}

func TestCreateExecutionLogRouteRequiresStatus(t *testing.T) {
	server := NewServer(ServerConfig{ExecutionLogs: &fakeHTTPExecutionLogStore{}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/execution-logs", bytes.NewReader([]byte(`{
		"actionType": "send_text"
	}`)))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestListExecutionLogsRoute(t *testing.T) {
	actionType := "send_presence"
	server := NewServer(ServerConfig{ExecutionLogs: &fakeHTTPExecutionLogStore{items: []repository.ExecutionLog{
		{
			ID:             "log-id",
			ActionType:     &actionType,
			Status:         "success",
			RequestPayload: map[string]any{"presence": "composing"},
			CreatedAt:      time.Date(2026, 5, 4, 15, 0, 0, 0, time.UTC),
		},
	}}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/execution-logs", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	var response listExecutionLogsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(response.Items) != 1 {
		t.Fatalf("items len = %d", len(response.Items))
	}
	if response.Items[0].ActionType == nil || *response.Items[0].ActionType != "send_presence" {
		t.Fatalf("ActionType = %v", response.Items[0].ActionType)
	}
}
