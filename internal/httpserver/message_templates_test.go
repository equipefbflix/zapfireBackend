package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"aquecedor-evolution/backend/internal/repository"
)

type fakeHTTPMessageTemplateStore struct {
	createParams repository.CreateMessageTemplateParams
	updateParams repository.UpdateMessageTemplateParams
	items        []repository.MessageTemplate
}

func (s *fakeHTTPMessageTemplateStore) Create(ctx context.Context, params repository.CreateMessageTemplateParams) (repository.MessageTemplate, error) {
	s.createParams = params
	return repository.MessageTemplate{
		ID:              "template-id",
		Category:        params.Category,
		Title:           params.Title,
		Body:            params.Body,
		Weight:          params.Weight,
		Enabled:         params.Enabled,
		MinWarmingScore: params.MinWarmingScore,
		MaxWarmingScore: params.MaxWarmingScore,
		Metadata:        params.Metadata,
	}, nil
}

func (s *fakeHTTPMessageTemplateStore) List(ctx context.Context) ([]repository.MessageTemplate, error) {
	return s.items, nil
}

func (s *fakeHTTPMessageTemplateStore) Update(ctx context.Context, id string, params repository.UpdateMessageTemplateParams) (repository.MessageTemplate, error) {
	s.updateParams = params
	return repository.MessageTemplate{
		ID:              id,
		Category:        valueOr(params.Category, "casual"),
		Title:           valueOr(params.Title, "template atualizado"),
		Body:            valueOr(params.Body, "body atualizado"),
		Weight:          intOr(params.Weight, 1),
		Enabled:         boolOr(params.Enabled, true),
		MinWarmingScore: floatOr(params.MinWarmingScore, 0),
		MaxWarmingScore: floatOr(params.MaxWarmingScore, 100),
		Metadata:        params.Metadata,
	}, nil
}

func TestCreateMessageTemplateRoute(t *testing.T) {
	store := &fakeHTTPMessageTemplateStore{}
	server := NewServer(ServerConfig{MessageTemplates: store})
	body := []byte(`{
		"category": "casual",
		"title": "bom dia simples",
		"body": "Bom dia, tudo certo por ai?",
		"weight": 10,
		"enabled": true,
		"minWarmingScore": 0,
		"maxWarmingScore": 40,
		"testRunId": "test-run",
		"metadata": {"tone":"friendly"}
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/message-templates", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if store.createParams.Metadata["testRunId"] != "test-run" {
		t.Fatalf("testRunId = %v", store.createParams.Metadata["testRunId"])
	}
	if store.createParams.Weight != 10 {
		t.Fatalf("Weight = %d", store.createParams.Weight)
	}

	var response messageTemplateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.ID != "template-id" {
		t.Fatalf("ID = %q", response.ID)
	}
	if response.MaxWarmingScore != 40 {
		t.Fatalf("MaxWarmingScore = %f", response.MaxWarmingScore)
	}
}

func TestCreateMessageTemplateRouteRequiresBody(t *testing.T) {
	server := NewServer(ServerConfig{MessageTemplates: &fakeHTTPMessageTemplateStore{}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/message-templates", bytes.NewReader([]byte(`{
		"category": "casual",
		"title": "missing body"
	}`)))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestCreateMessageTemplateRouteRequiresCategory(t *testing.T) {
	server := NewServer(ServerConfig{MessageTemplates: &fakeHTTPMessageTemplateStore{}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/message-templates", bytes.NewReader([]byte(`{
		"title": "bom dia simples",
		"body": "Bom dia, tudo certo por ai?"
	}`)))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestListMessageTemplatesRoute(t *testing.T) {
	server := NewServer(ServerConfig{MessageTemplates: &fakeHTTPMessageTemplateStore{items: []repository.MessageTemplate{
		{
			ID:              "template-id",
			Category:        "casual",
			Title:           "bom dia simples",
			Body:            "Bom dia, tudo certo por ai?",
			Weight:          10,
			Enabled:         true,
			MinWarmingScore: 0,
			MaxWarmingScore: 40,
			Metadata:        map[string]any{"tone": "friendly"},
		},
	}}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/message-templates", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	var response listMessageTemplatesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(response.Items) != 1 {
		t.Fatalf("items len = %d", len(response.Items))
	}
	if response.Items[0].Category != "casual" {
		t.Fatalf("Category = %q", response.Items[0].Category)
	}
}

func TestUpdateMessageTemplateRoute(t *testing.T) {
	store := &fakeHTTPMessageTemplateStore{}
	server := NewServer(ServerConfig{MessageTemplates: store})
	body := []byte(`{
		"title": "template atualizado",
		"body": "novo body",
		"weight": 3,
		"enabled": false
	}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/message-templates/template-id", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if store.updateParams.Title == nil || *store.updateParams.Title != "template atualizado" {
		t.Fatalf("Title = %v", store.updateParams.Title)
	}
}

func valueOr(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return *value
}

func intOr(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}

func boolOr(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func floatOr(value *float64, fallback float64) float64 {
	if value == nil {
		return fallback
	}
	return *value
}
