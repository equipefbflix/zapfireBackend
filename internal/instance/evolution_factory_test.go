package instance

import (
	"testing"

	"aquecedor-evolution/backend/internal/repository"
)

func TestEnvSecretResolver(t *testing.T) {
	t.Setenv("EVOLUTION_API_KEY", "secret")

	resolver := EnvSecretResolver{}
	if got := resolver.Resolve("EVOLUTION_API_KEY"); got != "secret" {
		t.Fatalf("Resolve() = %q", got)
	}
}

func TestEvolutionClientFactoryCreatesClient(t *testing.T) {
	t.Setenv("EVOLUTION_API_KEY", "secret")

	factory := EvolutionClientFactory{
		SecretResolver: EnvSecretResolver{},
	}
	creator := factory.New(repository.EvolutionServer{
		BaseURL:          "https://evo.example.com",
		APIKeySecretName: "EVOLUTION_API_KEY",
	})

	if creator == nil {
		t.Fatal("creator = nil")
	}
}
