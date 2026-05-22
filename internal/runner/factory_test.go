package runner

import (
	"context"
	"strings"
	"testing"

	"aquecedor-evolution/backend/internal/evolution"
	"aquecedor-evolution/backend/internal/executor"
	"aquecedor-evolution/backend/internal/repository"
)

type fakeEvolutionServerStore struct {
	id     string
	server repository.EvolutionServer
}

func (s *fakeEvolutionServerStore) GetByID(ctx context.Context, id string) (repository.EvolutionServer, error) {
	s.id = id
	return s.server, nil
}

type fakeSecretResolver struct{}

func (fakeSecretResolver) Resolve(secretName string) string {
	if value, ok := strings.CutPrefix(secretName, "literal:"); ok {
		return value
	}
	return "resolved-" + secretName
}

type fakeStepClientFactory struct {
	server repository.EvolutionServer
	apiKey string
}

func (f *fakeStepClientFactory) New(server repository.EvolutionServer, apiKey string) executor.EvolutionStepClient {
	f.server = server
	f.apiKey = apiKey
	return &fakeEvolutionStepClientForFactory{}
}

type fakeEvolutionStepClientForFactory struct{}

func (c *fakeEvolutionStepClientForFactory) SendText(ctx context.Context, instanceName string, request evolution.SendTextRequest) (evolution.SendMessageResponse, error) {
	return evolution.SendMessageResponse{}, nil
}

func (c *fakeEvolutionStepClientForFactory) SendMedia(ctx context.Context, instanceName string, request evolution.SendMediaRequest) (evolution.SendMessageResponse, error) {
	return evolution.SendMessageResponse{}, nil
}

func (c *fakeEvolutionStepClientForFactory) SendWhatsAppAudio(ctx context.Context, instanceName string, request evolution.SendWhatsAppAudioRequest) (evolution.SendMessageResponse, error) {
	return evolution.SendMessageResponse{}, nil
}

func (c *fakeEvolutionStepClientForFactory) SendStatus(ctx context.Context, instanceName string, request evolution.SendStatusRequest) (evolution.SendMessageResponse, error) {
	return evolution.SendMessageResponse{}, nil
}

func (c *fakeEvolutionStepClientForFactory) SendSticker(ctx context.Context, instanceName string, request evolution.SendStickerRequest) error {
	return nil
}

func (c *fakeEvolutionStepClientForFactory) SendPresence(ctx context.Context, instanceName string, request evolution.SendPresenceRequest) error {
	return nil
}

func (c *fakeEvolutionStepClientForFactory) SendReaction(ctx context.Context, instanceName string, request evolution.SendReactionRequest) error {
	return nil
}

func TestInstanceExecutorFactoryForInstance(t *testing.T) {
	serverStore := &fakeEvolutionServerStore{server: repository.EvolutionServer{
		ID:               "server-id",
		Name:             "evo-01",
		BaseURL:          "https://evo.example.com",
		APIKeySecretName: "EVOLUTION_EVO_01_API_KEY",
	}}
	clientFactory := &fakeStepClientFactory{}
	factory := NewInstanceExecutorFactory(serverStore, fakeSecretResolver{}, clientFactory)

	stepExecutor, err := factory.ForInstance(context.Background(), repository.Instance{
		ID:                       "instance-id",
		EvolutionServerID:        "server-id",
		InstanceAPIKeySecretName: stringPtr("literal:instance-token"),
	})
	if err != nil {
		t.Fatalf("ForInstance() error = %v", err)
	}
	if serverStore.id != "server-id" {
		t.Fatalf("server id = %q", serverStore.id)
	}
	if clientFactory.apiKey != "instance-token" {
		t.Fatalf("api key = %q", clientFactory.apiKey)
	}
	if stepExecutor == nil {
		t.Fatal("stepExecutor = nil")
	}
}

func stringPtr(value string) *string {
	return &value
}
