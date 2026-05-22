//go:build e2e

package instance

import (
	"context"
	"os"
	"testing"
	"time"

	"aquecedor-evolution/backend/internal/config"
	"aquecedor-evolution/backend/internal/database"
	"aquecedor-evolution/backend/internal/evolution"
	"aquecedor-evolution/backend/internal/repository"
)

func TestServiceCreateRealEvolutionOnboardingInstanceWithoutPhoneE2E(t *testing.T) {
	if os.Getenv("ENABLE_REAL_TESTS") != "true" {
		t.Skip("set ENABLE_REAL_TESTS=true to run real Evolution E2E tests")
	}

	baseURL := firstEnv("EVOLUTION_TEST_BASE_URL", "SERVER_URL")
	apiKey := firstEnv("EVOLUTION_TEST_API_KEY", "AUTHENTICATION_API_KEY")
	if baseURL == "" {
		t.Fatal("EVOLUTION_TEST_BASE_URL or SERVER_URL is required")
	}
	if apiKey == "" {
		t.Fatal("EVOLUTION_TEST_API_KEY or AUTHENTICATION_API_KEY is required")
	}

	testRunID := os.Getenv("TEST_RUN_ID")
	if testRunID == "" {
		testRunID = "instance-onboarding-e2e-" + time.Now().UTC().Format("20060102T150405")
	}
	instanceName := "codex_onboarding_" + time.Now().UTC().Format("20060102t150405")

	dbConfig, err := config.LoadDatabaseConfig()
	if err != nil {
		t.Fatalf("LoadDatabaseConfig() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := database.Open(ctx, dbConfig)
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	defer pool.Close()

	executor := repository.NewPgxExecutor(pool)
	servers := repository.NewEvolutionServerRepository(executor)
	instances := repository.NewInstanceRepository(executor)

	client := evolution.NewClient(evolution.Config{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Timeout: 60 * time.Second,
	})

	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cleanupCancel()
		_, _ = instances.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = servers.DeleteByTestRunID(cleanupCtx, testRunID)
		_ = client.DeleteInstance(cleanupCtx, instanceName)
	}()

	_, err = servers.Create(ctx, repository.CreateEvolutionServerParams{
		Name:              "000_onboarding_e2e_" + testRunID,
		BaseURL:           baseURL,
		APIKeySecretName:  "EVOLUTION_TEST_API_KEY",
		Enabled:           true,
		Weight:            1,
		MaxConcurrentJobs: 1,
		Metadata:          map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("servers.Create() error = %v", err)
	}

	t.Setenv("EVOLUTION_TEST_API_KEY", apiKey)
	service := NewService(ServiceConfig{
		EvolutionServers: servers,
		Proxies:          emptyProxyStore{},
		Instances:        instances,
		EvolutionFactory: EvolutionClientFactory{
			SecretResolver: EnvSecretResolver{},
			Timeout:        60 * time.Second,
		},
		SecretResolver: EnvSecretResolver{},
	})

	created, err := service.Create(ctx, CreateParams{
		InstanceName: instanceName,
		TestRunID:    testRunID,
	})
	if err != nil {
		t.Fatalf("service.Create() without phone number error = %v", err)
	}
	if created.ID == "" {
		t.Fatal("created ID is empty")
	}

	items, err := client.FetchInstances(ctx, instanceName)
	if err != nil {
		t.Fatalf("FetchInstances() error = %v", err)
	}
	if len(items) == 0 {
		t.Fatal("FetchInstances() returned zero items for created onboarding instance")
	}

	connectResponse, err := service.Connect(ctx, created.ID)
	if err != nil {
		t.Fatalf("service.Connect() error = %v", err)
	}
	if connectResponse.PairingCode == "" && connectResponse.Code == "" {
		t.Fatalf("connect response = %#v, want pairing payload", connectResponse)
	}
}
