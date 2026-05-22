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

type fakeHTTPProxyStore struct {
	createParams repository.CreateProxyParams
	items        []repository.Proxy
}

func (s *fakeHTTPProxyStore) Create(ctx context.Context, params repository.CreateProxyParams) (repository.Proxy, error) {
	s.createParams = params
	return repository.Proxy{
		ID:                 "proxy-id",
		Name:               params.Name,
		Host:               params.Host,
		Port:               params.Port,
		Protocol:           params.Protocol,
		Username:           params.Username,
		PasswordSecretName: params.PasswordSecretName,
		Enabled:            params.Enabled,
		MaxInstances:       params.MaxInstances,
		CurrentInstances:   0,
		Metadata:           params.Metadata,
	}, nil
}

func (s *fakeHTTPProxyStore) List(ctx context.Context) ([]repository.Proxy, error) {
	return s.items, nil
}

func TestCreateProxyRoute(t *testing.T) {
	store := &fakeHTTPProxyStore{}
	server := NewServer(ServerConfig{Proxies: store})
	body := []byte(`{
		"name": "proxy-sp-01",
		"host": "proxy.example.com",
		"port": 8000,
		"protocol": "http",
		"username": "proxy-user",
		"passwordSecretName": "PROXY_SP_01_PASSWORD",
		"enabled": true,
		"maxInstances": 20,
		"testRunId": "test-run",
		"metadata": {"provider":"datacenter"}
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/proxies", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if store.createParams.Metadata["testRunId"] != "test-run" {
		t.Fatalf("testRunId = %v", store.createParams.Metadata["testRunId"])
	}
	if store.createParams.PasswordSecretName == nil || *store.createParams.PasswordSecretName != "PROXY_SP_01_PASSWORD" {
		t.Fatalf("PasswordSecretName = %v", store.createParams.PasswordSecretName)
	}

	var response proxyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.ID != "proxy-id" {
		t.Fatalf("ID = %q", response.ID)
	}
	if response.PasswordSecretName == nil || *response.PasswordSecretName != "PROXY_SP_01_PASSWORD" {
		t.Fatalf("PasswordSecretName = %v", response.PasswordSecretName)
	}
}

func TestCreateProxyRouteRequiresHost(t *testing.T) {
	server := NewServer(ServerConfig{Proxies: &fakeHTTPProxyStore{}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/proxies", bytes.NewReader([]byte(`{
		"name": "proxy-sp-01",
		"port": 8000,
		"protocol": "http"
	}`)))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestCreateProxyRouteRequiresPort(t *testing.T) {
	server := NewServer(ServerConfig{Proxies: &fakeHTTPProxyStore{}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/proxies", bytes.NewReader([]byte(`{
		"name": "proxy-sp-01",
		"host": "proxy.example.com",
		"protocol": "http"
	}`)))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestListProxiesRoute(t *testing.T) {
	username := "proxy-user"
	passwordSecretName := "PROXY_SP_01_PASSWORD"
	maxInstances := 20
	server := NewServer(ServerConfig{Proxies: &fakeHTTPProxyStore{items: []repository.Proxy{
		{
			ID:                 "proxy-id",
			Name:               "proxy-sp-01",
			Host:               "proxy.example.com",
			Port:               8000,
			Protocol:           "http",
			Username:           &username,
			PasswordSecretName: &passwordSecretName,
			Enabled:            true,
			MaxInstances:       &maxInstances,
			CurrentInstances:   3,
			Metadata:           map[string]any{"provider": "datacenter"},
		},
	}}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/proxies", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestTestProxyRoute(t *testing.T) {
	server := NewServer(ServerConfig{})
	body := []byte(`{
		"host": "127.0.0.1",
		"port": 1080,
		"protocol": "socks5"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/proxies/test", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}

	var response testProxyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.Success {
		t.Fatalf("expected success=false for unreachable proxy")
	}
}

func TestTestProxyRouteDefaultsProtocol(t *testing.T) {
	server := NewServer(ServerConfig{})
	body := []byte(`{"host": "127.0.0.1", "port": 1080}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/proxies/test", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestTestProxyRouteRequiresHost(t *testing.T) {
	server := NewServer(ServerConfig{})
	body := []byte(`{"port": 1080}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/proxies/test", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestTestProxyRouteRequiresPort(t *testing.T) {
	server := NewServer(ServerConfig{})
	body := []byte(`{"host": "127.0.0.1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/proxies/test", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestTestProxyRouteRejectsInvalidJSON(t *testing.T) {
	server := NewServer(ServerConfig{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/proxies/test", bytes.NewReader([]byte(`{invalid`)))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}
