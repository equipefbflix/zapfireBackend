package runner

import (
	"context"
	"time"

	"aquecedor-evolution/backend/internal/evolution"
	"aquecedor-evolution/backend/internal/executor"
	"aquecedor-evolution/backend/internal/repository"
)

type EvolutionServerStore interface {
	GetByID(ctx context.Context, id string) (repository.EvolutionServer, error)
}

type SecretResolver interface {
	Resolve(secretName string) string
}

type StepClientFactory interface {
	New(server repository.EvolutionServer, apiKey string) executor.EvolutionStepClient
}

type InstanceStepExecutorFactory interface {
	ForInstance(ctx context.Context, instance repository.Instance) (*executor.StepExecutor, error)
}

type instanceExecutorFactory struct {
	servers       EvolutionServerStore
	secrets       SecretResolver
	clientFactory StepClientFactory
}

func NewInstanceExecutorFactory(servers EvolutionServerStore, secrets SecretResolver, clientFactory StepClientFactory) InstanceStepExecutorFactory {
	return instanceExecutorFactory{
		servers:       servers,
		secrets:       secrets,
		clientFactory: clientFactory,
	}
}

func (f instanceExecutorFactory) ForInstance(ctx context.Context, instance repository.Instance) (*executor.StepExecutor, error) {
	server, err := f.servers.GetByID(ctx, instance.EvolutionServerID)
	if err != nil {
		return nil, err
	}

	apiKey := ""
	if f.secrets != nil {
		apiKey = f.secrets.Resolve(server.APIKeySecretName)
	}

	client := f.clientFactory.New(server, apiKey)
	stepExecutor := executor.NewStepExecutor(client)
	return &stepExecutor, nil
}

type DefaultStepClientFactory struct {
	Timeout time.Duration
}

func (f DefaultStepClientFactory) New(server repository.EvolutionServer, apiKey string) executor.EvolutionStepClient {
	return evolution.NewClient(evolution.Config{
		BaseURL: server.BaseURL,
		APIKey:  apiKey,
		Timeout: f.Timeout,
	})
}
