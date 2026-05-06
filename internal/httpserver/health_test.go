package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"aquecedor-evolution/backend/internal/config"
	"aquecedor-evolution/backend/internal/observability"
)

type fakeHTTPObservability struct {
	snapshot observability.Snapshot
}

func (s fakeHTTPObservability) Snapshot(ctx context.Context) (observability.Snapshot, error) {
	return s.snapshot, nil
}

func TestHealthRoutes(t *testing.T) {
	server := NewServer(ServerConfig{
		App: config.AppConfig{
			Env:       "test",
			Port:      "0",
			PublicURL: "http://localhost",
		},
		EvolutionServers: []config.EvolutionServerConfig{
			{Name: "evo1", BaseURL: "https://evo1.example.com", Enabled: true},
		},
		Observability: fakeHTTPObservability{snapshot: observability.Snapshot{
			JobStatusCounts: map[string]int{"running": 1},
		}},
	})

	for _, path := range []string{"/health", "/api/v1/health"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()

			server.Handler().ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d", rec.Code)
			}

			var response HealthResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			if response.Status != "ok" {
				t.Fatalf("status = %q", response.Status)
			}
			if response.AppEnv != "test" {
				t.Fatalf("app env = %q", response.AppEnv)
			}
			if len(response.EvolutionServers) != 1 {
				t.Fatalf("evolution servers len = %d", len(response.EvolutionServers))
			}
			if response.EvolutionServers[0].Name != "evo1" {
				t.Fatalf("evolution server name = %q", response.EvolutionServers[0].Name)
			}
			if response.Supabase.Status != "healthy" {
				t.Fatalf("supabase status = %q", response.Supabase.Status)
			}
			if response.Metrics == nil || response.Metrics.JobStatusCounts["running"] != 1 {
				t.Fatalf("metrics = %+v", response.Metrics)
			}
		})
	}
}

func TestHealthRouteDegradedWhenStaleJobsExist(t *testing.T) {
	server := NewServer(ServerConfig{
		App: config.AppConfig{Env: "test", Port: "0", PublicURL: "http://localhost"},
		Observability: fakeHTTPObservability{snapshot: observability.Snapshot{
			StaleRunningJobs: 1,
		}},
	})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var response HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.Status != "degraded" {
		t.Fatalf("status = %q", response.Status)
	}
}

func TestNotFound(t *testing.T) {
	server := NewServer(ServerConfig{})
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}
