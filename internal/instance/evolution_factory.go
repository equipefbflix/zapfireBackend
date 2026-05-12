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
}

func literalSecret(secretName string) (string, bool) {
	value, ok := strings.CutPrefix(secretName, "literal:")
	return value, ok
}

func (f EvolutionClientFactory) New(server repository.EvolutionServer) EvolutionInstanceCreator {
	var apiKey string
	if f.SecretResolver != nil {
		apiKey = f.SecretResolver.Resolve(server.APIKeySecretName)
	}
	return evolution.NewClient(evolution.Config{
		BaseURL: server.BaseURL,
		APIKey:  apiKey,
		Timeout: f.Timeout,
	})
}
