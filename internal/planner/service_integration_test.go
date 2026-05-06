//go:build integration

package planner

import (
	"context"
	"os"
	"testing"
	"time"

	"aquecedor-evolution/backend/internal/config"
	"aquecedor-evolution/backend/internal/database"
	"aquecedor-evolution/backend/internal/repository"
)

func TestServicePlanRealDatabaseSkipsRecentScript(t *testing.T) {
	if os.Getenv("ENABLE_REAL_TESTS") != "true" {
		t.Skip("set ENABLE_REAL_TESTS=true to run real database integration tests")
	}

	testRunID := os.Getenv("TEST_RUN_ID")
	if testRunID == "" {
		testRunID = "planner-" + time.Now().UTC().Format("20060102T150405")
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

	executor := repository.NewPgxExecutor(pool)
	phones := repository.NewPhoneNumberRepository(executor)
	scripts := repository.NewConversationScriptRepository(executor)
	jobs := repository.NewWarmingJobRepository(executor)

	scriptNameA := "planner_a_" + testRunID
	scriptNameB := "planner_b_" + testRunID

	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		_, _ = jobs.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = phones.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = scripts.DeleteByName(cleanupCtx, scriptNameA)
		_, _ = scripts.DeleteByName(cleanupCtx, scriptNameB)
	}()

	phoneA, err := phones.Create(ctx, repository.CreatePhoneNumberParams{
		PhoneE164: "+550000000301",
		Label:     "planner_phone_a",
		Metadata:  map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("phones.Create(phoneA) error = %v", err)
	}

	phoneB, err := phones.Create(ctx, repository.CreatePhoneNumberParams{
		PhoneE164: "+550000000302",
		Label:     "planner_phone_b",
		Metadata:  map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("phones.Create(phoneB) error = %v", err)
	}

	scriptA, err := scripts.Create(ctx, repository.CreateConversationScriptParams{
		Name:            scriptNameA,
		Category:        "casual",
		Enabled:         true,
		Weight:          10,
		MinWarmingScore: 0,
		MaxWarmingScore: 100,
	})
	if err != nil {
		t.Fatalf("scripts.Create(scriptA) error = %v", err)
	}

	scriptB, err := scripts.Create(ctx, repository.CreateConversationScriptParams{
		Name:            scriptNameB,
		Category:        "casual",
		Enabled:         true,
		Weight:          5,
		MinWarmingScore: 0,
		MaxWarmingScore: 100,
	})
	if err != nil {
		t.Fatalf("scripts.Create(scriptB) error = %v", err)
	}

	if _, err := jobs.Create(ctx, repository.CreateWarmingJobParams{
		ScriptID:    &scriptA.ID,
		PhoneAID:    phoneA.ID,
		PhoneBID:    phoneB.ID,
		ScheduledAt: time.Now().UTC().Add(-10 * time.Minute),
		Metadata:    map[string]any{"testRunId": testRunID},
	}); err != nil {
		t.Fatalf("jobs.Create() error = %v", err)
	}

	service := NewService(config.PlannerConfig{
		MinDelaySeconds:     20,
		MaxDelaySeconds:     20,
		PairCooldownMinutes: 30,
		WindowStartHour:     8,
		WindowEndHour:       22,
	}, scripts, jobs, func() time.Time {
		return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC)
	})

	plan, err := service.Plan(ctx, Params{
		PhoneAID:  phoneA.ID,
		PhoneBID:  phoneB.ID,
		PairScore: 20,
		Category:  "casual",
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if plan.Script.ID != scriptB.ID {
		t.Fatalf("script id = %q want %q", plan.Script.ID, scriptB.ID)
	}
}
