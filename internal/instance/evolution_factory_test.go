package instance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

func TestEvolutionClientFactoryPassesWebhookURLToConnectClient(t *testing.T) {
	t.Setenv("EVOLUTION_API_KEY", "secret")

	var payload map[string]any
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch callCount {
		case 1:
			if r.URL.Path != "/instance/connect" {
				t.Fatalf("path = %s", r.URL.Path)
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"message":"success","data":{"jid":"5511888888888@s.whatsapp.net","webhookUrl":"https://pairing.ngrok-free.app/api/v1/webhooks/evolution","eventString":"messages.upsert,connection.update"}}`))
		case 2:
			if r.URL.Path != "/instance/qr" {
				t.Fatalf("path = %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"message":"success","data":{"qrcode":"data:image/png;base64,AAA","code":"2@ABCDEF"}}`))
		default:
			t.Fatalf("unexpected call %d: %s %s", callCount, r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	factory := EvolutionClientFactory{
		SecretResolver: EnvSecretResolver{},
		Timeout:        time.Second,
		WebhookURL:     "https://pairing.ngrok-free.app/api/v1/webhooks/evolution",
	}
	creator := factory.New(repository.EvolutionServer{
		BaseURL:          server.URL,
		APIKeySecretName: "EVOLUTION_API_KEY",
	})

	if _, err := creator.ConnectInstance(context.Background(), "chip_1", ""); err != nil {
		t.Fatalf("ConnectInstance() error = %v", err)
	}
	if payload["webhookUrl"] != "https://pairing.ngrok-free.app/api/v1/webhooks/evolution" {
		t.Fatalf("webhookUrl = %#v", payload["webhookUrl"])
	}
}
