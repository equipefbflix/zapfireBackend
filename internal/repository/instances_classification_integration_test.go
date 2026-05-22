//go:build integration

package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"aquecedor-evolution/backend/internal/config"
	"aquecedor-evolution/backend/internal/database"
)

func TestInstanceRepositoryUpdateClassificationRealDatabase(t *testing.T) {
	if os.Getenv("ENABLE_REAL_TESTS") != "true" {
		t.Skip("set ENABLE_REAL_TESTS=true to run real database integration tests")
	}

	testRunID := os.Getenv("TEST_RUN_ID")
	if testRunID == "" {
		testRunID = "instance-classification-" + time.Now().UTC().Format("20060102T150405")
	}

	dbConfig, err := config.LoadDatabaseConfig()
	if err != nil {
		t.Fatalf("LoadDatabaseConfig() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := database.Open(ctx, dbConfig)
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	defer pool.Close()

	executor := NewPgxExecutor(pool)
	phones := NewPhoneNumberRepository(executor)
	servers := NewEvolutionServerRepository(executor)
	instances := NewInstanceRepository(executor)

	defer func() {
		_, _ = executor.Exec(context.Background(), `delete from public.instances where metadata ->> 'testRunId' = $1`, testRunID)
		_, _ = executor.Exec(context.Background(), `delete from public.phone_numbers where metadata ->> 'testRunId' = $1`, testRunID)
		_, _ = executor.Exec(context.Background(), `delete from public.evolution_servers where metadata ->> 'testRunId' = $1`, testRunID)
	}()

	phone, err := phones.Create(ctx, CreatePhoneNumberParams{
		PhoneE164: "+550000009999",
		Label:     "instance classification " + testRunID,
		Metadata:  map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("phones.Create() error = %v", err)
	}

	server, err := servers.Create(ctx, CreateEvolutionServerParams{
		Name:              "instance-classification-" + testRunID,
		BaseURL:           "https://go.zaapfire.com.br",
		APIKeySecretName:  "EVOLUTION_TEST_API_KEY",
		Enabled:           true,
		Weight:            1,
		MaxConcurrentJobs: 5,
		Metadata:          map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("servers.Create() error = %v", err)
	}

	instance, err := instances.Create(ctx, CreateInstanceParams{
		PhoneNumberID:     phone.ID,
		EvolutionServerID: server.ID,
		InstanceName:      "instance_classification_" + testRunID,
		Status:            "created",
		Metadata:          map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("instances.Create() error = %v", err)
	}

	updated, err := instances.UpdateClassification(ctx, instance.ID, UpdateInstanceClassificationParams{
		Classification: "internal",
	})
	if err != nil {
		t.Fatalf("UpdateClassification() error = %v", err)
	}
	if updated.Metadata["classification"] != "internal" {
		t.Fatalf("updated.Metadata[classification] = %v", updated.Metadata["classification"])
	}
}
