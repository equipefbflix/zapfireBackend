//go:build e2e

package instance

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"aquecedor-evolution/backend/internal/config"
	"aquecedor-evolution/backend/internal/database"
	"aquecedor-evolution/backend/internal/evolution"
	"aquecedor-evolution/backend/internal/repository"
)

type emptyProxyStore struct{}

func (emptyProxyStore) ListEnabled(ctx context.Context) ([]repository.Proxy, error) {
	return nil, nil
}

func TestServiceCreateRealEvolutionInstanceE2E(t *testing.T) {
	if os.Getenv("ENABLE_REAL_TESTS") != "true" {
		t.Skip("set ENABLE_REAL_TESTS=true to run real Evolution E2E tests")
	}

	baseURL := os.Getenv("EVOLUTION_TEST_BASE_URL")
	apiKey := os.Getenv("EVOLUTION_TEST_API_KEY")
	if baseURL == "" {
		t.Fatal("EVOLUTION_TEST_BASE_URL is required")
	}
	if apiKey == "" {
		t.Fatal("EVOLUTION_TEST_API_KEY is required")
	}

	testRunID := os.Getenv("TEST_RUN_ID")
	if testRunID == "" {
		testRunID = "instance-e2e-" + time.Now().UTC().Format("20060102T150405")
	}
	instanceName := "codex_e2e_" + time.Now().UTC().Format("20060102t150405")

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
	phones := repository.NewPhoneNumberRepository(executor)
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
		_, _ = phones.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = servers.DeleteByTestRunID(cleanupCtx, testRunID)
		_ = client.DeleteInstance(cleanupCtx, instanceName)
	}()

	phone, err := phones.Create(ctx, repository.CreatePhoneNumberParams{
		PhoneE164: "+5511999999999",
		Label:     "e2e_instance_phone_" + testRunID,
		Metadata:  map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("phones.Create() error = %v", err)
	}

	server, err := servers.Create(ctx, repository.CreateEvolutionServerParams{
		Name:              "000_e2e_" + testRunID,
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
		PhoneNumberID: phone.ID,
		PhoneE164:     "5511999999999",
		InstanceName:  instanceName,
		TestRunID:     testRunID,
	})
	if err != nil {
		t.Fatalf("service.Create() error = %v", err)
	}
	if created.ID == "" {
		t.Fatal("created.ID is empty")
	}
	if created.EvolutionServerID != server.ID {
		t.Fatalf("created.EvolutionServerID = %q", created.EvolutionServerID)
	}

	items, err := client.FetchInstances(ctx, instanceName)
	if err != nil {
		t.Fatalf("FetchInstances() error = %v", err)
	}
	if len(items) == 0 {
		t.Fatal("FetchInstances() returned zero items for created instance")
	}

	if err := client.DeleteInstance(ctx, instanceName); err != nil {
		t.Fatalf("DeleteInstance() error = %v", err)
	}

	time.Sleep(2 * time.Second)

	items, err = client.FetchInstances(ctx, instanceName)
	if err != nil {
		httpErr, ok := err.(*evolution.HTTPError)
		if !ok || httpErr.StatusCode != 404 {
			t.Fatalf("FetchInstances() after delete error = %v", err)
		}
		return
	}
	if len(items) != 0 {
		t.Fatalf("instance still exists after delete: %s", fmt.Sprintf("%+v", items))
	}
}
