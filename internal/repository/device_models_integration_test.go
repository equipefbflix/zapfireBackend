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

func TestDeviceModelRepositoryCreateAndListRealDatabase(t *testing.T) {
	if os.Getenv("ENABLE_REAL_TESTS") != "true" {
		t.Skip("set ENABLE_REAL_TESTS=true to run real database integration tests")
	}

	testRunID := os.Getenv("TEST_RUN_ID")
	if testRunID == "" {
		testRunID = "device-models-" + time.Now().UTC().Format("20060102T150405")
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
	repo := NewDeviceModelRepository(executor)

	defer func() {
		_, _ = executor.Exec(context.Background(), `
delete from public.device_models
where metadata ->> 'testRunId' = $1
`, testRunID)
	}()

	created, err := repo.Create(ctx, CreateDeviceModelParams{
		Name:         "iPhone 15 " + testRunID,
		OS:           "ios",
		SystemLabel:  "iOS",
		VersionLabel: "17.4",
		ImageURL:     "https://example.com/iphone15.png",
		SortOrder:    7,
		Enabled:      true,
		Metadata:     map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID == "" {
		t.Fatal("created.ID is empty")
	}

	items, err := repo.ListEnabled(ctx)
	if err != nil {
		t.Fatalf("ListEnabled() error = %v", err)
	}

	found := false
	for _, item := range items {
		if item.ID == created.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("created model %q not found in enabled list", created.ID)
	}
}

