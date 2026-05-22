package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"aquecedor-evolution/backend/internal/evolution"
	"aquecedor-evolution/backend/internal/instance"
	"aquecedor-evolution/backend/internal/repository"
)

type fakeInstanceCreator struct {
	params                    instance.CreateParams
	err                       error
	item                      repository.Instance
	items                     []repository.Instance
	updateClassificationID    string
	updateClassificationValue string
}

type fakeInstanceOperationalSummaryStore struct {
	item instanceOperationalSummaryResponse
	err  error
}

func (s *fakeInstanceOperationalSummaryStore) GetOperationalSummary(ctx context.Context, id string) (instanceOperationalSummaryResponse, error) {
	if s.err != nil {
		return instanceOperationalSummaryResponse{}, s.err
	}
	if s.item.InstanceID == "" {
		s.item = instanceOperationalSummaryResponse{
			InstanceID:         id,
			PhoneNumberID:      "phone-id",
			InstanceName:       "chip_5511999999999",
			Status:             "open",
			ConnectionStatus:   "open",
			WarmingScore:       42,
			DailyMessageCount:  12,
			DailyLimit:         30,
			LastConnectionSync: "2026-05-21T10:00:00Z",
		}
	}
	return s.item, nil
}

func (c *fakeInstanceCreator) Create(ctx context.Context, params instance.CreateParams) (repository.Instance, error) {
	c.params = params
	if c.err != nil {
		return repository.Instance{}, c.err
	}
	proxyID := "proxy-id"
	metadata := map[string]any{}
	if params.Classification != "" {
		metadata["classification"] = params.Classification
	}
	return repository.Instance{
		ID:                "instance-id",
		PhoneNumberID:     params.PhoneNumberID,
		EvolutionServerID: "server-id",
		ProxyID:           &proxyID,
		InstanceName:      params.InstanceName,
		Status:            "created",
		Metadata:          metadata,
	}, nil
}

func (c *fakeInstanceCreator) Restart(ctx context.Context, phoneNumberID string) error {
	return c.err
}

func (c *fakeInstanceCreator) RestartByID(ctx context.Context, id string) error {
	return c.err
}

func (c *fakeInstanceCreator) DeleteByID(ctx context.Context, id string) error {
	return c.err
}

func (c *fakeInstanceCreator) List(ctx context.Context) ([]repository.Instance, error) {
	if c.err != nil {
		return nil, c.err
	}
	return c.items, nil
}

func (c *fakeInstanceCreator) GetByID(ctx context.Context, id string) (repository.Instance, error) {
	if c.err != nil {
		return repository.Instance{}, c.err
	}
	if c.item.ID != "" {
		return c.item, nil
	}
	return repository.Instance{ID: id, PhoneNumberID: "phone-id", EvolutionServerID: "server-id", InstanceName: "chip_5511999999999", Status: "open"}, nil
}

func (c *fakeInstanceCreator) Connect(ctx context.Context, id string) (evolution.ConnectInstanceResponse, error) {
	if c.err != nil {
		return evolution.ConnectInstanceResponse{}, c.err
	}
	return evolution.ConnectInstanceResponse{PairingCode: "PAIR-123", Code: "QR-TOKEN", Count: 1}, nil
}

func (c *fakeInstanceCreator) SyncState(ctx context.Context, id string) (repository.Instance, error) {
	if c.err != nil {
		return repository.Instance{}, c.err
	}
	if c.item.ID != "" {
		return c.item, nil
	}
	return repository.Instance{ID: id, PhoneNumberID: "phone-id", EvolutionServerID: "server-id", InstanceName: "chip_5511999999999", Status: "open"}, nil
}

func (c *fakeInstanceCreator) UpdateClassification(ctx context.Context, id string, classification string) (repository.Instance, error) {
	c.updateClassificationID = id
	c.updateClassificationValue = classification
	if c.err != nil {
		return repository.Instance{}, c.err
	}
	item := c.item
	if item.ID == "" {
		item = repository.Instance{ID: id, PhoneNumberID: "phone-id", EvolutionServerID: "server-id", InstanceName: "chip_5511999999999", Status: "open", Metadata: map[string]any{}}
	}
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}
	item.Metadata["classification"] = classification
	return item, nil
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
		"classification": "external",
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

func TestCreateInstanceRouteWithManualProxy(t *testing.T) {
	creator := &fakeInstanceCreator{}
	server := NewServer(ServerConfig{
		InstanceCreator: creator,
	})
	body := []byte(`{
		"instanceName": "chip_5511999999999",
		"classification": "external",
		"manualProxy": {
			"host": "manual.proxy.local",
			"port": 9000,
			"protocol": "http",
			"username": "proxy-user",
			"password": "proxy-pass"
		}
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if creator.params.ManualProxy == nil {
		t.Fatal("ManualProxy = nil")
	}
	if creator.params.ManualProxy.Host != "manual.proxy.local" {
		t.Fatalf("ManualProxy.Host = %q", creator.params.ManualProxy.Host)
	}
	if creator.params.ManualProxy.Port != 9000 {
		t.Fatalf("ManualProxy.Port = %d", creator.params.ManualProxy.Port)
	}
	if creator.params.ManualProxy.Protocol != "http" {
		t.Fatalf("ManualProxy.Protocol = %q", creator.params.ManualProxy.Protocol)
	}
	if creator.params.ManualProxy.Username == nil || *creator.params.ManualProxy.Username != "proxy-user" {
		t.Fatalf("ManualProxy.Username = %v", creator.params.ManualProxy.Username)
	}
	if creator.params.ManualProxy.Password == nil || *creator.params.ManualProxy.Password != "proxy-pass" {
		t.Fatalf("ManualProxy.Password = %v", creator.params.ManualProxy.Password)
	}
}

func TestGetInstanceOperationalSummaryRoute(t *testing.T) {
	server := NewServer(ServerConfig{
		InstanceCreator:            &fakeInstanceCreator{},
		InstanceOperationalSummary: &fakeInstanceOperationalSummaryStore{},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances/instance-1/operational-summary", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var response instanceOperationalSummaryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if response.WarmingScore != 42 {
		t.Fatalf("WarmingScore = %v", response.WarmingScore)
	}
	if response.DailyLimit != 30 {
		t.Fatalf("DailyLimit = %d", response.DailyLimit)
	}
}

func TestUpdateInstanceClassificationRoute(t *testing.T) {
	server := NewServer(ServerConfig{
		InstanceCreator: &fakeInstanceCreator{},
	})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/instances/instance-1/classification", bytes.NewReader([]byte(`{"classification":"internal"}`)))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
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

func TestCreateInstanceRouteRejectsInvalidManualProxy(t *testing.T) {
	server := NewServer(ServerConfig{
		InstanceCreator: &fakeInstanceCreator{},
	})
	body := []byte(`{
		"instanceName": "chip_5511999999999",
		"manualProxy": {
			"host": "manual.proxy.local",
			"port": 0,
			"protocol": "ftp"
		}
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestListInstancesRoute(t *testing.T) {
	server := NewServer(ServerConfig{
		InstanceCreator: &fakeInstanceCreator{items: []repository.Instance{
			{
				ID:                  "instance-1",
				PhoneNumberID:       "phone-1",
				EvolutionServerID:   "server-1",
				InstanceName:        "chip_5511999999999",
				Status:              "open",
				EvolutionInstanceID: stringPtr("evo-instance-1"),
				Metadata:            map[string]any{"source": "test"},
			},
		}},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}

	var response struct {
		Items []createInstanceResponse `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(response.Items) != 1 {
		t.Fatalf("items len = %d", len(response.Items))
	}
	if response.Items[0].InstanceName != "chip_5511999999999" {
		t.Fatalf("InstanceName = %q", response.Items[0].InstanceName)
	}
}

func TestGetInstanceRoute(t *testing.T) {
	server := NewServer(ServerConfig{
		InstanceCreator: &fakeInstanceCreator{item: repository.Instance{
			ID:                "instance-1",
			PhoneNumberID:     "phone-1",
			EvolutionServerID: "server-1",
			InstanceName:      "chip_5511999999999",
			Status:            "open",
		}},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances/instance-1", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestConnectInstanceRoute(t *testing.T) {
	server := NewServer(ServerConfig{
		InstanceCreator: &fakeInstanceCreator{},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances/instance-1/connect", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestSyncInstanceStateRoute(t *testing.T) {
	server := NewServer(ServerConfig{
		InstanceCreator: &fakeInstanceCreator{item: repository.Instance{
			ID:                "instance-1",
			PhoneNumberID:     "phone-1",
			EvolutionServerID: "server-1",
			InstanceName:      "chip_5511999999999",
			Status:            "open",
		}},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances/instance-1/sync-state", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestRestartInstanceRoute(t *testing.T) {
	server := NewServer(ServerConfig{
		InstanceCreator: &fakeInstanceCreator{},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances/instance-1/restart", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteInstanceRoute(t *testing.T) {
	server := NewServer(ServerConfig{
		InstanceCreator: &fakeInstanceCreator{},
	})
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/instances/instance-1", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteInstanceRouteMapsError(t *testing.T) {
	server := NewServer(ServerConfig{
		InstanceCreator: &fakeInstanceCreator{err: errors.New("boom")},
	})
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/instances/instance-1", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestDeleteInstanceRouteRequiresConfig(t *testing.T) {
	server := NewServer(ServerConfig{})
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/instances/instance-1", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
}

func stringPtr(value string) *string {
	return &value
}
