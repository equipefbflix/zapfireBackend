package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"aquecedor-evolution/backend/internal/observability"
)

type fakeHTTPStaleCleanup struct {
	affected int64
}

func (s fakeHTTPStaleCleanup) Cleanup(ctx context.Context) (int64, error) {
	return s.affected, nil
}

func TestMetricsRoute(t *testing.T) {
	server := NewServer(ServerConfig{
		Observability: fakeHTTPObservability{snapshot: observability.Snapshot{
			JobStatusCounts: map[string]int{"pending": 2},
			ExecutionFailures: 1,
			EvolutionEvents: 5,
		}},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var response observability.Snapshot
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.JobStatusCounts["pending"] != 2 {
		t.Fatalf("pending = %d", response.JobStatusCounts["pending"])
	}
}

func TestStaleCleanupRoute(t *testing.T) {
	server := NewServer(ServerConfig{
		StaleJobCleanup: fakeHTTPStaleCleanup{affected: 3},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/warming-jobs/stale-cleanup", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var response StaleCleanupResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.Affected != 3 {
		t.Fatalf("affected = %d", response.Affected)
	}
}
