package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"aquecedor-evolution/backend/internal/config"
	"aquecedor-evolution/backend/internal/repository"
)

type fakeDeviceModelStore struct {
	createParams repository.CreateDeviceModelParams
	updateParams repository.UpdateDeviceModelParams
	err          error
	items        []repository.DeviceModel
	item         repository.DeviceModel
	includeDisabled bool
}

func (f *fakeDeviceModelStore) Create(ctx context.Context, params repository.CreateDeviceModelParams) (repository.DeviceModel, error) {
	f.createParams = params
	if f.err != nil {
		return repository.DeviceModel{}, f.err
	}
	return repository.DeviceModel{
		ID:           "model-1",
		Name:         params.Name,
		OS:           params.OS,
		SystemLabel:  params.SystemLabel,
		VersionLabel: params.VersionLabel,
		ImageURL:     params.ImageURL,
		SortOrder:    params.SortOrder,
		Enabled:      params.Enabled,
		Metadata:     params.Metadata,
	}, nil
}

func (f *fakeDeviceModelStore) List(ctx context.Context, includeDisabled bool) ([]repository.DeviceModel, error) {
	f.includeDisabled = includeDisabled
	if f.err != nil {
		return nil, f.err
	}
	return f.items, nil
}

func (f *fakeDeviceModelStore) Update(ctx context.Context, id string, params repository.UpdateDeviceModelParams) (repository.DeviceModel, error) {
	f.updateParams = params
	if f.err != nil {
		return repository.DeviceModel{}, f.err
	}
	return repository.DeviceModel{
		ID:           id,
		Name:         deviceValueOr(params.Name, "iPhone 15"),
		OS:           deviceValueOr(params.OS, "ios"),
		SystemLabel:  deviceValueOr(params.SystemLabel, "iOS"),
		VersionLabel: deviceValueOr(params.VersionLabel, "17.4"),
		ImageURL:     deviceValueOr(params.ImageURL, "https://example.com/iphone15.png"),
		SortOrder:    deviceIntValueOr(params.SortOrder, 1),
		Enabled:      deviceBoolValueOr(params.Enabled, true),
		Metadata:     params.Metadata,
	}, nil
}

func TestDeviceModelCreateRoute(t *testing.T) {
	store := &fakeDeviceModelStore{}
	server := NewServer(ServerConfig{DeviceModels: store})

	body := []byte(`{
		"name":"iPhone 15",
		"os":"ios",
		"systemLabel":"iOS",
		"versionLabel":"17.4",
		"imageUrl":"https://example.com/iphone15.png",
		"sortOrder":1,
		"enabled":true,
		"technicalProfile":{
			"brand":"Apple",
			"model":"iPhone15,2",
			"buildId":"21E236",
			"resolution":"1179 x 2556 px",
			"dpi":"460 ppi",
			"cpu":"A16 Bionic",
			"ram":"6 GB",
			"fingerprintTemplate":"Apple/iPhone15,2/iPhone15,2:17.4/21E236/1:user/release-keys"
		},
		"metadata":{"color":"black"}
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/device-models", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if store.createParams.Name != "iPhone 15" {
		t.Fatalf("store.createParams.Name = %q", store.createParams.Name)
	}
	technicalProfile, ok := store.createParams.Metadata["technicalProfile"].(map[string]any)
	if !ok {
		t.Fatalf("technicalProfile metadata missing: %#v", store.createParams.Metadata)
	}
	if technicalProfile["cpu"] != "A16 Bionic" {
		t.Fatalf("technicalProfile.cpu = %#v", technicalProfile["cpu"])
	}
}

func TestDeviceModelListRoute(t *testing.T) {
	store := &fakeDeviceModelStore{items: []repository.DeviceModel{
		{
			ID:           "model-1",
			Name:         "iPhone 15",
			OS:           "ios",
			SystemLabel:  "iOS",
			VersionLabel: "17.4",
			ImageURL:     "https://example.com/iphone15.png",
			SortOrder:    1,
			Enabled:      true,
		},
	}}
	server := NewServer(ServerConfig{DeviceModels: store})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/device-models", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var response struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(response.Items) != 1 {
		t.Fatalf("len(response.Items) = %d", len(response.Items))
	}
}

func TestDeviceModelUpdateRoute(t *testing.T) {
	store := &fakeDeviceModelStore{}
	server := NewServer(ServerConfig{DeviceModels: store})

	body := []byte(`{
		"enabled":false,
		"sortOrder":4,
		"technicalProfile":{
			"brand":"Google",
			"model":"oriole",
			"resolution":"1080 x 2400 px"
		}
	}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/device-models/model-1", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if store.updateParams.Enabled == nil || *store.updateParams.Enabled != false {
		t.Fatalf("store.updateParams.Enabled = %v", store.updateParams.Enabled)
	}
	if store.updateParams.SortOrder == nil || *store.updateParams.SortOrder != 4 {
		t.Fatalf("store.updateParams.SortOrder = %v", store.updateParams.SortOrder)
	}
	technicalProfile, ok := store.updateParams.Metadata["technicalProfile"].(map[string]any)
	if !ok {
		t.Fatalf("technicalProfile metadata missing: %#v", store.updateParams.Metadata)
	}
	if technicalProfile["brand"] != "Google" {
		t.Fatalf("technicalProfile.brand = %#v", technicalProfile["brand"])
	}
}

func TestDeviceModelListRouteMapsErrors(t *testing.T) {
	server := NewServer(ServerConfig{DeviceModels: &fakeDeviceModelStore{err: errors.New("boom")}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/device-models", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestDeviceModelListRouteIsPublicForWizard(t *testing.T) {
	store := &fakeDeviceModelStore{items: []repository.DeviceModel{{
		ID: "model-1",
		Name: "iPhone 13 Pro",
		OS: "ios",
		Enabled: true,
	}}}
	server := NewServer(ServerConfig{
		App: config.AppConfig{AuthEnabled: true},
		AuthVerifier: fakeHTTPAuthVerifier{},
		DeviceModels: store,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/device-models", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if store.includeDisabled {
		t.Fatal("includeDisabled = true, want false for public wizard access")
	}
}

func TestDeviceModelListRouteIgnoresIncludeDisabledWithoutAuth(t *testing.T) {
	store := &fakeDeviceModelStore{items: []repository.DeviceModel{}}
	server := NewServer(ServerConfig{
		App: config.AppConfig{AuthEnabled: true},
		AuthVerifier: fakeHTTPAuthVerifier{},
		DeviceModels: store,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/device-models?includeDisabled=true", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if store.includeDisabled {
		t.Fatal("includeDisabled = true, want false when request is unauthenticated")
	}
}

func deviceValueOr(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return *value
}

func deviceIntValueOr(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}

func deviceBoolValueOr(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}
