package instance

import (
	"context"
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

type fakeProxyStore struct {
	proxies []repository.Proxy
}

func (s fakeProxyStore) ListEnabled(ctx context.Context) ([]repository.Proxy, error) {
	return s.proxies, nil
}

type fakeInstanceStore struct {
	params repository.CreateInstanceParams
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

type fakeEvolutionCreator struct {
	request evolution.CreateInstanceRequest
}

func (c *fakeEvolutionCreator) CreateInstance(ctx context.Context, request evolution.CreateInstanceRequest) (evolution.CreateInstanceResponse, error) {
	c.request = request
	var response evolution.CreateInstanceResponse
	response.Instance.InstanceName = request.InstanceName
	response.Hash.APIKey = "instance-api-key"
	return response, nil
}

type fakeEvolutionFactory struct {
	creator *fakeEvolutionCreator
}

func (f fakeEvolutionFactory) New(server repository.EvolutionServer) EvolutionInstanceCreator {
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
		Proxies: fakeProxyStore{proxies: []repository.Proxy{
			{ID: proxyID, Name: "proxy1", Host: "proxy.example.com", Port: 8000, Protocol: "http", Username: &username, PasswordSecretName: &passwordSecret, Enabled: true, MaxInstances: &maxInstances},
		}},
		Instances:        instanceStore,
		EvolutionFactory: fakeEvolutionFactory{creator: creator},
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
}

func TestServiceCreateRequiresEnabledEvolutionServer(t *testing.T) {
	service := NewService(ServiceConfig{
		EvolutionServers: fakeEvolutionServerStore{},
		Proxies:          fakeProxyStore{},
		Instances:        &fakeInstanceStore{},
		EvolutionFactory: fakeEvolutionFactory{creator: &fakeEvolutionCreator{}},
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
