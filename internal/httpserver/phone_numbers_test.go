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

type fakePhoneNumberStore struct {
	createParams repository.CreatePhoneNumberParams
	items        []repository.PhoneNumber
}

func (s *fakePhoneNumberStore) Create(ctx context.Context, params repository.CreatePhoneNumberParams) (repository.PhoneNumber, error) {
	s.createParams = params
	return repository.PhoneNumber{
		ID:           "phone-id",
		PhoneE164:    params.PhoneE164,
		Label:        params.Label,
		Status:       "new",
		WarmingScore: 0,
		Metadata:     params.Metadata,
	}, nil
}

func (s *fakePhoneNumberStore) List(ctx context.Context) ([]repository.PhoneNumber, error) {
	return s.items, nil
}

func (s *fakePhoneNumberStore) Update(ctx context.Context, id string, params repository.UpdatePhoneNumberParams) (repository.PhoneNumber, error) {
	return repository.PhoneNumber{
		ID:        id,
		PhoneE164: "5511999999999",
		Label:     *params.Label,
		Status:    *params.Status,
		Metadata:  params.Metadata,
	}, nil
}

func (s *fakePhoneNumberStore) Delete(ctx context.Context, id string) error {
	return nil
}

func (s *fakePhoneNumberStore) GetByID(ctx context.Context, id string) (repository.PhoneNumber, error) {
	return repository.PhoneNumber{
		ID:               id,
		PhoneE164:        "5511999999999",
		Label:            "chip",
		Status:           "warming",
		WarmingScore:     42,
		ConnectionStatus: "open",
		Metadata:         map[string]any{},
	}, nil
}

func (s *fakePhoneNumberStore) GetDailyMessageCount(ctx context.Context, phoneNumberID string) (int, error) {
	return 0, nil
}

func TestCreatePhoneNumberRoute(t *testing.T) {
	store := &fakePhoneNumberStore{}
	server := NewServer(ServerConfig{PhoneNumbers: store})
	body := []byte(`{
		"phoneE164": "5511999999999",
		"label": "chip-sp-01",
		"testRunId": "test-run",
		"metadata": {"carrier":"vivo"}
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/phone-numbers", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if store.createParams.Metadata["testRunId"] != "test-run" {
		t.Fatalf("testRunId = %v", store.createParams.Metadata["testRunId"])
	}

	var response phoneNumberResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.ID != "phone-id" {
		t.Fatalf("ID = %q", response.ID)
	}
}

func TestCreatePhoneNumberRouteRequiresPhone(t *testing.T) {
	server := NewServer(ServerConfig{PhoneNumbers: &fakePhoneNumberStore{}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/phone-numbers", bytes.NewReader([]byte(`{"label":"missing"}`)))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestListPhoneNumbersRoute(t *testing.T) {
	server := NewServer(ServerConfig{PhoneNumbers: &fakePhoneNumberStore{items: []repository.PhoneNumber{
		{ID: "phone-id", PhoneE164: "5511999999999", Label: "chip", Status: "new", Metadata: map[string]any{}},
	}}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/phone-numbers", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	var response listPhoneNumbersResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(response.Items) != 1 {
		t.Fatalf("items len = %d", len(response.Items))
	}
}

func TestUpdatePhoneNumberRoute(t *testing.T) {
	store := &fakePhoneNumberStore{}
	server := NewServer(ServerConfig{PhoneNumbers: store})
	body := []byte(`{
		"label": "updated-chip",
		"status": "active"
	}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/phone-numbers/phone-id", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}

	var response phoneNumberResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.Label != "updated-chip" {
		t.Fatalf("Label = %q", response.Label)
	}
}

func TestDeletePhoneNumberRoute(t *testing.T) {
	store := &fakePhoneNumberStore{}
	server := NewServer(ServerConfig{PhoneNumbers: store})
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/phone-numbers/phone-id", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestRestartPhoneNumberRoute(t *testing.T) {
	creator := &fakeInstanceCreator{}
	server := NewServer(ServerConfig{
		PhoneNumbers:    &fakePhoneNumberStore{},
		InstanceCreator: creator,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/phone-numbers/phone-id/restart", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestGetDailyLimitRoute(t *testing.T) {
	server := NewServer(ServerConfig{
		PhoneNumbers:        &fakePhoneNumberStore{},
		DailyLimitPerNumber: 30,
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/phone-numbers/phone-id/daily-limit", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}

	var response dailyLimitResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.PhoneNumberID != "phone-id" {
		t.Fatalf("PhoneNumberID = %q", response.PhoneNumberID)
	}
	if response.DailyLimit != 30 {
		t.Fatalf("DailyLimit = %d", response.DailyLimit)
	}
}

func TestGetDailyLimitRouteDefaultsTo30(t *testing.T) {
	server := NewServer(ServerConfig{
		PhoneNumbers: &fakePhoneNumberStore{},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/phone-numbers/phone-id/daily-limit", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}

	var response dailyLimitResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.DailyLimit != 30 {
		t.Fatalf("DailyLimit = %d", response.DailyLimit)
	}
}

func TestGetDailyLimitRouteRequiresConfig(t *testing.T) {
	server := NewServer(ServerConfig{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/phone-numbers/phone-id/daily-limit", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
}
