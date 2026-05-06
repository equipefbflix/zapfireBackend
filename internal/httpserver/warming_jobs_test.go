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

type fakeHTTPWarmingJobStore struct {
	createParams repository.CreateWarmingJobParams
	items        []repository.WarmingJob
}

type fakeHTTPWarmingJobRunner struct {
	jobID    string
	executed int
	err      error
}

func (r *fakeHTTPWarmingJobRunner) Run(ctx context.Context, jobID string) (int, error) {
	r.jobID = jobID
	return r.executed, r.err
}

func (s *fakeHTTPWarmingJobStore) Create(ctx context.Context, params repository.CreateWarmingJobParams) (repository.WarmingJob, error) {
	s.createParams = params
	return repository.WarmingJob{
		ID:               "job-id",
		ScriptID:         params.ScriptID,
		PhoneAID:         params.PhoneAID,
		PhoneBID:         params.PhoneBID,
		Status:           "pending",
		ScheduledAt:      params.ScheduledAt,
		CurrentStepOrder: 0,
		Error:            "",
		Metadata:         params.Metadata,
	}, nil
}

func (s *fakeHTTPWarmingJobStore) List(ctx context.Context) ([]repository.WarmingJob, error) {
	return s.items, nil
}

func TestCreateWarmingJobRoute(t *testing.T) {
	store := &fakeHTTPWarmingJobStore{}
	server := NewServer(ServerConfig{WarmingJobs: store})
	body := []byte(`{
		"phoneAId": "phone-a-id",
		"phoneBId": "phone-b-id",
		"scriptId": "script-id",
		"scheduledAt": "2026-05-04T15:00:00Z",
		"testRunId": "test-run",
		"metadata": {"source":"manual"}
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/warming-jobs", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if store.createParams.Metadata["testRunId"] != "test-run" {
		t.Fatalf("testRunId = %v", store.createParams.Metadata["testRunId"])
	}
	if store.createParams.ScheduledAt.IsZero() {
		t.Fatal("ScheduledAt is zero")
	}

	var response warmingJobResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.ID != "job-id" {
		t.Fatalf("ID = %q", response.ID)
	}
}

func TestCreateWarmingJobRouteRequiresDifferentPhones(t *testing.T) {
	server := NewServer(ServerConfig{WarmingJobs: &fakeHTTPWarmingJobStore{}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/warming-jobs", bytes.NewReader([]byte(`{
		"phoneAId": "same-phone-id",
		"phoneBId": "same-phone-id",
		"scheduledAt": "2026-05-04T15:00:00Z"
	}`)))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestCreateWarmingJobRouteRequiresScheduledAt(t *testing.T) {
	server := NewServer(ServerConfig{WarmingJobs: &fakeHTTPWarmingJobStore{}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/warming-jobs", bytes.NewReader([]byte(`{
		"phoneAId": "phone-a-id",
		"phoneBId": "phone-b-id"
	}`)))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestListWarmingJobsRoute(t *testing.T) {
	scriptID := "script-id"
	server := NewServer(ServerConfig{WarmingJobs: &fakeHTTPWarmingJobStore{items: []repository.WarmingJob{
		{
			ID:               "job-id",
			ScriptID:         &scriptID,
			PhoneAID:         "phone-a-id",
			PhoneBID:         "phone-b-id",
			Status:           "pending",
			ScheduledAt:      time.Date(2026, 5, 4, 15, 0, 0, 0, time.UTC),
			CurrentStepOrder: 0,
			Error:            "",
			Metadata:         map[string]any{"source": "manual"},
		},
	}}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/warming-jobs", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	var response listWarmingJobsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(response.Items) != 1 {
		t.Fatalf("items len = %d", len(response.Items))
	}
	if response.Items[0].Status != "pending" {
		t.Fatalf("Status = %q", response.Items[0].Status)
	}
}

func TestRunWarmingJobNowRoute(t *testing.T) {
	runner := &fakeHTTPWarmingJobRunner{executed: 3}
	server := NewServer(ServerConfig{WarmingJobRunner: runner})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/warming-jobs/job-123/run-now", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if runner.jobID != "job-123" {
		t.Fatalf("jobID = %q", runner.jobID)
	}

	var response runWarmingJobNowResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.ExecutedSteps != 3 {
		t.Fatalf("ExecutedSteps = %d", response.ExecutedSteps)
	}
}
