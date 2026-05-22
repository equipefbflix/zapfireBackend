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

func TestInstanceRepositoryCreateOnboardingWithoutPhoneNumberRealDatabase(t *testing.T) {
	if os.Getenv("ENABLE_REAL_TESTS") != "true" {
		t.Skip("set ENABLE_REAL_TESTS=true to run real database integration tests")
	}

	testRunID := os.Getenv("TEST_RUN_ID")
	if testRunID == "" {
		testRunID = "instance-onboarding-repository-" + time.Now().UTC().Format("20060102T150405")
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
	servers := NewEvolutionServerRepository(executor)
	instances := NewInstanceRepository(executor)

	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		_, _ = instances.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = servers.DeleteByTestRunID(cleanupCtx, testRunID)
	}()

	server, err := servers.Create(ctx, CreateEvolutionServerParams{
		Name:              "test_onboarding_evo_" + testRunID,
		BaseURL:           "https://evo.example.com",
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
		PhoneNumberID:     "",
		EvolutionServerID: server.ID,
		InstanceName:      "test_onboarding_instance_" + testRunID,
		Status:            "created",
		Metadata: map[string]any{
			"testRunId": testRunID,
			"origin":    "wizard-onboarding",
		},
	})
	if err != nil {
		t.Fatalf("instances.Create() without phone number error = %v", err)
	}
	if instance.ID == "" {
		t.Fatal("instance ID is empty")
	}
}
