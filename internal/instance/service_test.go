package instance

import (
	"context"
	"errors"
	"strings"
	"testing"

	"aquecedor-evolution/backend/internal/evolution"
	"aquecedor-evolution/backend/internal/repository"
)

type fakeEvolutionServerStore struct {
	servers []repository.EvolutionServer
}

func (s fakeEvolutionServerStore) ListEnabled(ctx context.Context) ([]repository.EvolutionServer, error) {
	return s.servers, nil
}

func (s fakeEvolutionServerStore) GetByID(ctx context.Context, id string) (repository.EvolutionServer, error) {
	for _, srv := range s.servers {
		if srv.ID == id {
			return srv, nil
		}
	}
	return repository.EvolutionServer{}, nil
}

type fakeProxyStore struct {
	proxies      []repository.Proxy
	upsertParams []repository.CreateProxyParams
	createParams []repository.CreateProxyParams
}

func (s fakeProxyStore) ListEnabled(ctx context.Context) ([]repository.Proxy, error) {
	return s.proxies, nil
}

func (s *fakeProxyStore) Upsert(ctx context.Context, params repository.CreateProxyParams) (repository.Proxy, error) {
	s.upsertParams = append(s.upsertParams, params)
	proxyID := "upserted-proxy-id"
	return repository.Proxy{
		ID:                 proxyID,
		Name:               params.Name,
		Host:               params.Host,
		Port:               params.Port,
		Protocol:           params.Protocol,
		Username:           params.Username,
		PasswordSecretName: params.PasswordSecretName,
		Enabled:            params.Enabled,
		Metadata:           params.Metadata,
	}, nil
}

func (s *fakeProxyStore) Create(ctx context.Context, params repository.CreateProxyParams) (repository.Proxy, error) {
	s.createParams = append(s.createParams, params)
	proxyID := "created-proxy-id"
	return repository.Proxy{
		ID:                 proxyID,
		Name:               params.Name,
		Host:               params.Host,
		Port:               params.Port,
		Protocol:           params.Protocol,
		Username:           params.Username,
		PasswordSecretName: params.PasswordSecretName,
		Enabled:            params.Enabled,
		Metadata:           params.Metadata,
	}, nil
}

type fakeFBFlixProxyProvider struct {
	proxy repository.CreateProxyParams
	err   error
}

func (p *fakeFBFlixProxyProvider) AcquireProxy(ctx context.Context, params FBFlixAcquireParams) (repository.CreateProxyParams, error) {
	if p.err != nil {
		return repository.CreateProxyParams{}, p.err
	}
	return p.proxy, nil
}

type fakeInstanceStore struct {
	params repository.CreateInstanceParams
	item   repository.Instance
	items  []repository.Instance
}

func (s *fakeInstanceStore) Create(ctx context.Context, params repository.CreateInstanceParams) (repository.Instance, error) {
	s.params = params
	return repository.Instance{
		ID:                "instance-id",
		PhoneNumberID:     params.PhoneNumberID,
		EvolutionServerID: params.EvolutionServerID,
		ProxyID:           params.ProxyID,
		InstanceName:      params.InstanceName,
		Status:            params.Status,
		Metadata:          params.Metadata,
	}, nil
}

func (s *fakeInstanceStore) GetOpenByPhoneNumberID(ctx context.Context, phoneNumberID string) (repository.Instance, error) {
	return repository.Instance{
		ID:                "instance-id",
		PhoneNumberID:     phoneNumberID,
		EvolutionServerID: "server-id",
		InstanceName:      "instance-name",
		Status:            "open",
	}, nil
}

func (s *fakeInstanceStore) GetByID(ctx context.Context, id string) (repository.Instance, error) {
	if s.item.ID != "" {
		return s.item, nil
	}
	return repository.Instance{
		ID:                id,
		PhoneNumberID:     "phone-id",
		EvolutionServerID: "server-id",
		InstanceName:      "instance-name",
		Status:            "open",
	}, nil
}

func (s *fakeInstanceStore) List(ctx context.Context) ([]repository.Instance, error) {
	if len(s.items) > 0 {
		return s.items, nil
	}
	return []repository.Instance{}, nil
}

func (s *fakeInstanceStore) UpdateClassification(ctx context.Context, id string, params repository.UpdateInstanceClassificationParams) (repository.Instance, error) {
	item := s.item
	if item.ID == "" {
		item = repository.Instance{
			ID:                id,
			PhoneNumberID:     "phone-id",
			EvolutionServerID: "server-id",
			InstanceName:      "instance-name",
			Status:            "open",
			Metadata:          map[string]any{},
		}
	}
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}
	item.Metadata["classification"] = params.Classification
	s.item = item
	return item, nil
}

func (s *fakeInstanceStore) UpdateConnectionStateByName(ctx context.Context, instanceName string, status string, lastError string) error {
	return nil
}

func (s *fakeInstanceStore) Delete(ctx context.Context, id string) error {
	return nil
}

type fakeEvolutionCreator struct {
	request  evolution.CreateInstanceRequest
	requests []evolution.CreateInstanceRequest
	errors   []error
}

func (c *fakeEvolutionCreator) CreateInstance(ctx context.Context, request evolution.CreateInstanceRequest) (evolution.CreateInstanceResponse, error) {
	c.request = request
	c.requests = append(c.requests, request)
	if len(c.errors) > 0 {
		err := c.errors[0]
		c.errors = c.errors[1:]
		if err != nil {
			return evolution.CreateInstanceResponse{}, err
		}
	}
	var response evolution.CreateInstanceResponse
	response.Instance.InstanceName = request.InstanceName
	response.Hash.APIKey = "instance-api-key"
	return response, nil
}

func (c *fakeEvolutionCreator) RestartInstance(ctx context.Context, instanceName string) error {
	return nil
}

func (c *fakeEvolutionCreator) DeleteInstance(ctx context.Context, instanceName string) error {
	return nil
}

func (c *fakeEvolutionCreator) ConnectInstance(ctx context.Context, instanceName, number string) (evolution.ConnectInstanceResponse, error) {
	return evolution.ConnectInstanceResponse{Code: "ok"}, nil
}

func (c *fakeEvolutionCreator) ConnectionState(ctx context.Context, instanceName string) (evolution.ConnectionStateResponse, error) {
	var response evolution.ConnectionStateResponse
	response.Instance.InstanceName = instanceName
	response.Instance.State = "open"
	return response, nil
}

type fakeEvolutionFactory struct {
	creator *fakeEvolutionCreator
	apiKey  string
}

func (f *fakeEvolutionFactory) New(server repository.EvolutionServer) EvolutionInstanceCreator {
	return f.creator
}

func (f *fakeEvolutionFactory) NewWithAPIKey(server repository.EvolutionServer, apiKey string) EvolutionInstanceCreator {
	f.apiKey = apiKey
	return f.creator
}

func TestServiceCreateWithProxy(t *testing.T) {
	proxyID := "proxy-id"
	username := "proxy-user"
	passwordSecret := "PROXY_PASSWORD"
	maxInstances := 10
	instanceStore := &fakeInstanceStore{}
	creator := &fakeEvolutionCreator{}
	service := NewService(ServiceConfig{
		EvolutionServers: fakeEvolutionServerStore{servers: []repository.EvolutionServer{
			{ID: "server-id", Name: "evo1", BaseURL: "https://evo.example.com", APIKeySecretName: "EVOLUTION_API_KEY", Enabled: true},
		}},
		Proxies: &fakeProxyStore{proxies: []repository.Proxy{
			{ID: proxyID, Name: "proxy1", Host: "proxy.example.com", Port: 8000, Protocol: "http", Username: &username, PasswordSecretName: &passwordSecret, Enabled: true, MaxInstances: &maxInstances},
		}},
		Instances:        instanceStore,
		EvolutionFactory: &fakeEvolutionFactory{creator: creator},
		SecretResolver: StaticSecretResolver{
			"EVOLUTION_API_KEY": "evolution-secret",
			"PROXY_PASSWORD":    "proxy-secret",
		},
		WebhookURL: "https://backend.example.com/api/v1/webhooks/evolution",
	})

	created, err := service.Create(context.Background(), CreateParams{
		PhoneNumberID: "phone-id",
		PhoneE164:     "5511999999999",
		InstanceName:  "chip_5511999999999",
		TestRunID:     "test-run",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if created.ID != "instance-id" {
		t.Fatalf("created ID = %q", created.ID)
	}
	if creator.request.ProxyHost != "proxy.example.com" {
		t.Fatalf("ProxyHost = %q", creator.request.ProxyHost)
	}
	if creator.request.ProxyPort != "8000" {
		t.Fatalf("ProxyPort = %q", creator.request.ProxyPort)
	}
	if creator.request.ProxyPassword != "proxy-secret" {
		t.Fatalf("ProxyPassword = %q", creator.request.ProxyPassword)
	}
	if creator.request.Webhook == nil {
		t.Fatal("Webhook = nil")
	}
	if creator.request.Webhook.URL != "https://backend.example.com/api/v1/webhooks/evolution" {
		t.Fatalf("Webhook.URL = %q", creator.request.Webhook.URL)
	}
	if !creator.request.Webhook.ByEvents {
		t.Fatal("Webhook.ByEvents = false")
	}
	if len(creator.request.Webhook.Events) != 4 {
		t.Fatalf("Webhook.Events len = %d", len(creator.request.Webhook.Events))
	}
	if instanceStore.params.EvolutionServerID != "server-id" {
		t.Fatalf("EvolutionServerID = %q", instanceStore.params.EvolutionServerID)
	}
	if instanceStore.params.ProxyID == nil || *instanceStore.params.ProxyID != "proxy-id" {
		t.Fatalf("ProxyID = %v", instanceStore.params.ProxyID)
	}
	if instanceStore.params.Metadata["testRunId"] != "test-run" {
		t.Fatalf("testRunId = %v", instanceStore.params.Metadata["testRunId"])
	}
	if creator.request.Token == "" {
		t.Fatal("Token = empty")
	}
	if instanceStore.params.InstanceAPIKeySecretName == nil {
		t.Fatal("InstanceAPIKeySecretName = nil")
	}
}

func TestServiceCreateWithLiteralProxyPassword(t *testing.T) {
	proxyID := "proxy-id"
	username := "proxy-user"
	passwordSecret := "literal:proxy-secret"
	instanceStore := &fakeInstanceStore{}
	creator := &fakeEvolutionCreator{}
	service := NewService(ServiceConfig{
		EvolutionServers: fakeEvolutionServerStore{servers: []repository.EvolutionServer{
			{ID: "server-id", Name: "evo1", BaseURL: "https://evo.example.com", APIKeySecretName: "EVOLUTION_API_KEY", Enabled: true},
		}},
		Proxies: &fakeProxyStore{proxies: []repository.Proxy{
			{ID: proxyID, Name: "proxy1", Host: "proxy.example.com", Port: 8000, Protocol: "http", Username: &username, PasswordSecretName: &passwordSecret, Enabled: true},
		}},
		Instances:        instanceStore,
		EvolutionFactory: &fakeEvolutionFactory{creator: creator},
		SecretResolver: StaticSecretResolver{
			"EVOLUTION_API_KEY": "evolution-secret",
		},
	})

	_, err := service.Create(context.Background(), CreateParams{
		PhoneNumberID: "phone-id",
		PhoneE164:     "5511999999999",
		InstanceName:  "chip_5511999999999",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if creator.request.ProxyPassword != "proxy-secret" {
		t.Fatalf("ProxyPassword = %q", creator.request.ProxyPassword)
	}
}

func TestServiceCreateRequiresEnabledEvolutionServer(t *testing.T) {
	service := NewService(ServiceConfig{
		EvolutionServers: fakeEvolutionServerStore{},
		Proxies:          &fakeProxyStore{},
		Instances:        &fakeInstanceStore{},
		EvolutionFactory: &fakeEvolutionFactory{creator: &fakeEvolutionCreator{}},
		SecretResolver:   StaticSecretResolver{},
	})

	if _, err := service.Create(context.Background(), CreateParams{
		PhoneNumberID: "phone-id",
		PhoneE164:     "5511999999999",
		InstanceName:  "chip_5511999999999",
	}); err == nil {
		t.Fatal("Create() error = nil, want error")
	}
}

func TestServiceCreateWithManualProxyPersistsAndUsesExactProxy(t *testing.T) {
	instanceStore := &fakeInstanceStore{}
	creator := &fakeEvolutionCreator{}
	service := NewService(ServiceConfig{
		EvolutionServers: fakeEvolutionServerStore{servers: []repository.EvolutionServer{
			{ID: "server-id", Name: "evo1", BaseURL: "https://evo.example.com", APIKeySecretName: "EVOLUTION_API_KEY", Enabled: true},
		}},
		Proxies:          &fakeProxyStore{},
		Instances:        instanceStore,
		EvolutionFactory: &fakeEvolutionFactory{creator: creator},
		SecretResolver: StaticSecretResolver{
			"EVOLUTION_API_KEY": "evolution-secret",
		},
	})

	created, err := service.Create(context.Background(), CreateParams{
		PhoneNumberID: "phone-id",
		PhoneE164:     "5511999999999",
		InstanceName:  "chip_5511999999999",
		ManualProxy: &ManualProxyInput{
			Host:     "manual.proxy.local",
			Port:     9000,
			Protocol: "http",
			Username: stringPtr("proxy-user"),
			Password: stringPtr("proxy-pass"),
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if created.ProxyID == nil {
		t.Fatal("created.ProxyID = nil")
	}
	if creator.request.ProxyHost != "manual.proxy.local" {
		t.Fatalf("ProxyHost = %q", creator.request.ProxyHost)
	}
	if creator.request.ProxyPort != "9000" {
		t.Fatalf("ProxyPort = %q", creator.request.ProxyPort)
	}
	if creator.request.ProxyUsername != "proxy-user" {
		t.Fatalf("ProxyUsername = %q", creator.request.ProxyUsername)
	}
	if creator.request.ProxyPassword != "proxy-pass" {
		t.Fatalf("ProxyPassword = %q", creator.request.ProxyPassword)
	}
}

func TestServiceCreateWithoutManualProxyAcquiresProxyFromFBFlix(t *testing.T) {
	instanceStore := &fakeInstanceStore{}
	creator := &fakeEvolutionCreator{}
	provider := &fakeFBFlixProxyProvider{
		proxy: repository.CreateProxyParams{
			Name:               "FBFlix auto",
			Host:               "fbflix.proxy.local",
			Port:               8080,
			Protocol:           "http",
			Username:           stringPtr("fbflix-user"),
			PasswordSecretName: stringPtr("literal:fbflix-pass"),
			Enabled:            true,
			Metadata:           map[string]any{"source": "fbflix"},
		},
	}
	service := NewService(ServiceConfig{
		EvolutionServers: fakeEvolutionServerStore{servers: []repository.EvolutionServer{
			{ID: "server-id", Name: "evo1", BaseURL: "https://evo.example.com", APIKeySecretName: "EVOLUTION_API_KEY", Enabled: true},
		}},
		Proxies:             &fakeProxyStore{},
		Instances:           instanceStore,
		EvolutionFactory:    &fakeEvolutionFactory{creator: creator},
		SecretResolver:      StaticSecretResolver{"EVOLUTION_API_KEY": "evolution-secret"},
		FBFlixProxyProvider: provider,
	})

	created, err := service.Create(context.Background(), CreateParams{
		PhoneNumberID: "phone-id",
		PhoneE164:     "5511999999999",
		InstanceName:  "chip_5511999999999",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if created.ProxyID == nil {
		t.Fatal("created.ProxyID = nil")
	}
	if creator.request.ProxyHost != "fbflix.proxy.local" {
		t.Fatalf("ProxyHost = %q", creator.request.ProxyHost)
	}
	if creator.request.ProxyPassword != "fbflix-pass" {
		t.Fatalf("ProxyPassword = %q", creator.request.ProxyPassword)
	}
}

func TestServiceCreateWithoutManualProxyReturnsErrorWhenFBFlixFails(t *testing.T) {
	instanceStore := &fakeInstanceStore{}
	creator := &fakeEvolutionCreator{}
	provider := &fakeFBFlixProxyProvider{
		err: errors.New("provider unavailable"),
	}
	service := NewService(ServiceConfig{
		EvolutionServers: fakeEvolutionServerStore{servers: []repository.EvolutionServer{
			{ID: "server-id", Name: "evo1", BaseURL: "https://evo.example.com", APIKeySecretName: "EVOLUTION_API_KEY", Enabled: true},
		}},
		Proxies:             &fakeProxyStore{},
		Instances:           instanceStore,
		EvolutionFactory:    &fakeEvolutionFactory{creator: creator},
		SecretResolver:      StaticSecretResolver{"EVOLUTION_API_KEY": "evolution-secret"},
		FBFlixProxyProvider: provider,
	})

	_, err := service.Create(context.Background(), CreateParams{
		PhoneNumberID: "phone-id",
		PhoneE164:     "5511999999999",
		InstanceName:  "chip_5511999999999",
	})
	if err == nil {
		t.Fatal("Create() error = nil")
	}
	if !strings.Contains(err.Error(), "acquire fbflix proxy") {
		t.Fatalf("error = %v", err)
	}
	if len(creator.requests) != 0 {
		t.Fatalf("CreateInstance requests = %d, want 0", len(creator.requests))
	}
}

func TestServiceConnectReturnsPairingPayload(t *testing.T) {
	factory := &fakeEvolutionFactory{creator: &fakeEvolutionCreator{}}
	service := NewService(ServiceConfig{
		EvolutionServers: fakeEvolutionServerStore{servers: []repository.EvolutionServer{
			{ID: "server-id", Name: "evo1", BaseURL: "https://evo.example.com", APIKeySecretName: "EVOLUTION_API_KEY", Enabled: true},
		}},
		Proxies: &fakeProxyStore{},
		Instances: &fakeInstanceStore{item: repository.Instance{
			ID:                       "instance-id",
			EvolutionServerID:        "server-id",
			InstanceName:             "instance-name",
			InstanceAPIKeySecretName: stringPtr("literal:instance-token"),
		}},
		EvolutionFactory: factory,
		SecretResolver:   StaticSecretResolver{},
	})

	response, err := service.Connect(context.Background(), "instance-id")
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if response.Code != "ok" {
		t.Fatalf("response.Code = %q", response.Code)
	}
	if factory.apiKey != "instance-token" {
		t.Fatalf("api key = %q", factory.apiKey)
	}
}

func TestServiceCreateRetriesWithVariantNameWhenEvolutionInstanceAlreadyExists(t *testing.T) {
	instanceStore := &fakeInstanceStore{}
	creator := &fakeEvolutionCreator{
		errors: []error{
			&evolution.HTTPError{StatusCode: 500, Body: `{"error":"instance already exists"}`},
			nil,
		},
	}
	service := NewService(ServiceConfig{
		EvolutionServers: fakeEvolutionServerStore{servers: []repository.EvolutionServer{
			{ID: "server-id", Name: "evo1", BaseURL: "https://evo.example.com", APIKeySecretName: "EVOLUTION_API_KEY", Enabled: true},
		}},
		Proxies:          &fakeProxyStore{},
		Instances:        instanceStore,
		EvolutionFactory: &fakeEvolutionFactory{creator: creator},
		SecretResolver: StaticSecretResolver{
			"EVOLUTION_API_KEY": "evolution-secret",
		},
	})

	created, err := service.Create(context.Background(), CreateParams{
		InstanceName: "teste",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if len(creator.requests) != 2 {
		t.Fatalf("CreateInstance() calls = %d", len(creator.requests))
	}
	if !strings.HasPrefix(creator.requests[0].InstanceName, "teste_") {
		t.Fatalf("first instance name should have timestamp suffix, got %q", creator.requests[0].InstanceName)
	}
	if creator.requests[1].InstanceName == creator.requests[0].InstanceName {
		t.Fatalf("second instance name should differ, got %q", creator.requests[1].InstanceName)
	}
	if !strings.HasPrefix(creator.requests[1].InstanceName, creator.requests[0].InstanceName+"_") {
		t.Fatalf("second instance name = %q", creator.requests[1].InstanceName)
	}
	if created.InstanceName != creator.requests[1].InstanceName {
		t.Fatalf("created instance name = %q, want %q", created.InstanceName, creator.requests[1].InstanceName)
	}
}
