package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"aquecedor-evolution/backend/internal/instance"
	"aquecedor-evolution/backend/internal/repository"
)

type fakeInstanceCreator struct {
	params instance.CreateParams
	err    error
}

func (c *fakeInstanceCreator) Create(ctx context.Context, params instance.CreateParams) (repository.Instance, error) {
	c.params = params
	if c.err != nil {
		return repository.Instance{}, c.err
	}
	proxyID := "proxy-id"
	return repository.Instance{
		ID:                "instance-id",
		PhoneNumberID:     params.PhoneNumberID,
		EvolutionServerID: "server-id",
		ProxyID:           &proxyID,
		InstanceName:      params.InstanceName,
		Status:            "created",
	}, nil
}

func (c *fakeInstanceCreator) Restart(ctx context.Context, phoneNumberID string) error {
	return c.err
}

func TestCreateInstanceRoute(t *testing.T) {
	creator := &fakeInstanceCreator{}
	server := NewServer(ServerConfig{
		InstanceCreator: creator,
	})
	body := []byte(`{
		"phoneNumberId": "phone-id",
		"phoneE164": "5511999999999",
		"instanceName": "chip_5511999999999",
		"testRunId": "test-run"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if creator.params.PhoneNumberID != "phone-id" {
		t.Fatalf("PhoneNumberID = %q", creator.params.PhoneNumberID)
	}
	if creator.params.TestRunID != "test-run" {
		t.Fatalf("TestRunID = %q", creator.params.TestRunID)
	}

	var response createInstanceResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.ID != "instance-id" {
		t.Fatalf("ID = %q", response.ID)
	}
	if response.ProxyID == nil || *response.ProxyID != "proxy-id" {
		t.Fatalf("ProxyID = %v", response.ProxyID)
	}
}

func TestCreateInstanceRouteRequiresFields(t *testing.T) {
	server := NewServer(ServerConfig{
		InstanceCreator: &fakeInstanceCreator{},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances", bytes.NewReader([]byte(`{"phoneNumberId":"phone-id"}`)))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestCreateInstanceRouteMapsNoEvolutionServer(t *testing.T) {
	server := NewServer(ServerConfig{
		InstanceCreator: &fakeInstanceCreator{err: instance.ErrNoEvolutionServer},
	})
	body := []byte(`{
		"phoneNumberId": "phone-id",
		"phoneE164": "5511999999999",
		"instanceName": "chip_5511999999999"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestCreateInstanceRouteMapsUnexpectedError(t *testing.T) {
	server := NewServer(ServerConfig{
		InstanceCreator: &fakeInstanceCreator{err: errors.New("boom")},
	})
	body := []byte(`{
		"phoneNumberId": "phone-id",
		"phoneE164": "5511999999999",
		"instanceName": "chip_5511999999999"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
}
