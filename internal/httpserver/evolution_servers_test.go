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

type fakeHTTPEvolutionServerStore struct {
	createParams repository.CreateEvolutionServerParams
	items        []repository.EvolutionServer
}

func (s *fakeHTTPEvolutionServerStore) Create(ctx context.Context, params repository.CreateEvolutionServerParams) (repository.EvolutionServer, error) {
	s.createParams = params
	return repository.EvolutionServer{
		ID:                "server-id",
		Name:              params.Name,
		BaseURL:           params.BaseURL,
		APIKeySecretName:  params.APIKeySecretName,
		Enabled:           params.Enabled,
		Weight:            params.Weight,
		MaxConcurrentJobs: params.MaxConcurrentJobs,
		HealthStatus:      "unknown",
		Metadata:          params.Metadata,
	}, nil
}

func (s *fakeHTTPEvolutionServerStore) List(ctx context.Context) ([]repository.EvolutionServer, error) {
	return s.items, nil
}

func TestCreateEvolutionServerRoute(t *testing.T) {
	store := &fakeHTTPEvolutionServerStore{}
	server := NewServer(ServerConfig{EvolutionStore: store})
	body := []byte(`{
		"name": "evo-01",
		"baseUrl": "https://evo.example.com",
		"apiKeySecretName": "EVOLUTION_EVO_01_API_KEY",
		"enabled": true,
		"weight": 2,
		"maxConcurrentJobs": 7,
		"testRunId": "test-run",
		"metadata": {"region":"br"}
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/evolution-servers", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if store.createParams.Metadata["testRunId"] != "test-run" {
		t.Fatalf("testRunId = %v", store.createParams.Metadata["testRunId"])
	}
	if store.createParams.Weight != 2 {
		t.Fatalf("Weight = %d", store.createParams.Weight)
	}

	var response evolutionServerResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.ID != "server-id" {
		t.Fatalf("ID = %q", response.ID)
	}
	if response.APIKeySecretName != "EVOLUTION_EVO_01_API_KEY" {
		t.Fatalf("APIKeySecretName = %q", response.APIKeySecretName)
	}
}

func TestCreateEvolutionServerRouteRequiresBaseURL(t *testing.T) {
	server := NewServer(ServerConfig{EvolutionStore: &fakeHTTPEvolutionServerStore{}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/evolution-servers", bytes.NewReader([]byte(`{
		"name": "evo-01",
		"apiKeySecretName": "EVOLUTION_EVO_01_API_KEY"
	}`)))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestCreateEvolutionServerRouteRequiresSecretName(t *testing.T) {
	server := NewServer(ServerConfig{EvolutionStore: &fakeHTTPEvolutionServerStore{}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/evolution-servers", bytes.NewReader([]byte(`{
		"name": "evo-01",
		"baseUrl": "https://evo.example.com"
	}`)))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestListEvolutionServersRoute(t *testing.T) {
	server := NewServer(ServerConfig{EvolutionStore: &fakeHTTPEvolutionServerStore{items: []repository.EvolutionServer{
		{
			ID:                "server-id",
			Name:              "evo-01",
			BaseURL:           "https://evo.example.com",
			APIKeySecretName:  "EVOLUTION_EVO_01_API_KEY",
			Enabled:           true,
			Weight:            2,
			MaxConcurrentJobs: 7,
			HealthStatus:      "healthy",
			Metadata:          map[string]any{"region": "br"},
		},
	}}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/evolution-servers", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	var response listEvolutionServersResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(response.Items) != 1 {
		t.Fatalf("items len = %d", len(response.Items))
	}
	if response.Items[0].HealthStatus != "healthy" {
		t.Fatalf("HealthStatus = %q", response.Items[0].HealthStatus)
	}
}
