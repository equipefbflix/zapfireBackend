package instance

import (
	"os"
	"strings"
	"time"

	"aquecedor-evolution/backend/internal/evolution"
	"aquecedor-evolution/backend/internal/repository"
)

type EnvSecretResolver struct{}

func (EnvSecretResolver) Resolve(secretName string) string {
	if value, ok := literalSecret(secretName); ok {
		return value
	}
	return os.Getenv(secretName)
}

type EvolutionClientFactory struct {
	SecretResolver SecretResolver
	Timeout        time.Duration
	WebhookURL     string
}

func literalSecret(secretName string) (string, bool) {
	value, ok := strings.CutPrefix(secretName, "literal:")
	return value, ok
}

func (f EvolutionClientFactory) New(server repository.EvolutionServer) EvolutionInstanceCreator {
	return f.NewWithAPIKey(server, "")
}

func (f EvolutionClientFactory) NewWithAPIKey(server repository.EvolutionServer, apiKey string) EvolutionInstanceCreator {
	resolvedAPIKey := apiKey
	if strings.TrimSpace(resolvedAPIKey) == "" && f.SecretResolver != nil {
		resolvedAPIKey = f.SecretResolver.Resolve(server.APIKeySecretName)
	}
	return evolution.NewClient(evolution.Config{
		BaseURL:    server.BaseURL,
		APIKey:     resolvedAPIKey,
		Timeout:    f.Timeout,
		WebhookURL: f.WebhookURL,
	})
}
