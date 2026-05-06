//go:build integration

package warmingscore

import (
	"context"
	"os"
	"testing"
	"time"

	"aquecedor-evolution/backend/internal/config"
	"aquecedor-evolution/backend/internal/database"
	"aquecedor-evolution/backend/internal/repository"
)

func TestServiceRecalculateRealDatabase(t *testing.T) {
	if os.Getenv("ENABLE_REAL_TESTS") != "true" {
		t.Skip("set ENABLE_REAL_TESTS=true to run real database integration tests")
	}

	testRunID := os.Getenv("TEST_RUN_ID")
	if testRunID == "" {
		testRunID = "warming-score-" + time.Now().UTC().Format("20060102T150405")
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
	servers := repository.NewEvolutionServerRepository(executor)
	instances := repository.NewInstanceRepository(executor)
	logs := repository.NewExecutionLogRepository(executor)

	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		_, _ = executor.Exec(cleanupCtx, `
delete from public.execution_logs
where instance_id in (
  select id from public.instances where metadata ->> 'testRunId' = $1
)
`, testRunID)
		_, _ = instances.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = phones.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = servers.DeleteByTestRunID(cleanupCtx, testRunID)
	}()

	phone, err := phones.Create(ctx, repository.CreatePhoneNumberParams{
		PhoneE164: "+550000000101",
		Label:     "warming_score_phone",
		Metadata:  map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("phones.Create() error = %v", err)
	}

	server, err := servers.Create(ctx, repository.CreateEvolutionServerParams{
		Name:              "warming-score-evo-" + testRunID,
		BaseURL:           "https://evo.example.com",
		APIKeySecretName:  "EVOLUTION_TEST_API_KEY",
		Enabled:           true,
		Weight:            1,
		MaxConcurrentJobs: 1,
		Metadata:          map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("servers.Create() error = %v", err)
	}

	instance, err := instances.Create(ctx, repository.CreateInstanceParams{
		PhoneNumberID:     phone.ID,
		EvolutionServerID: server.ID,
		InstanceName:      "warming_score_instance_" + testRunID,
		Status:            "open",
		Metadata:          map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("instances.Create() error = %v", err)
	}

	actionText := "send_text"
	actionReply := "send_reply"
	actionReaction := "send_reaction"
	for _, item := range []repository.CreateExecutionLogParams{
		{InstanceID: &instance.ID, ActionType: &actionText, Status: "success"},
		{InstanceID: &instance.ID, ActionType: &actionReply, Status: "success"},
		{InstanceID: &instance.ID, ActionType: &actionReaction, Status: "success"},
		{InstanceID: &instance.ID, ActionType: &actionText, Status: "failed"},
	} {
		if _, err := logs.Create(ctx, item); err != nil {
			t.Fatalf("logs.Create() error = %v", err)
		}
	}

	service := NewService(config.WarmingConfig{
		MinScoreToMarkWarm:       4,
		ScoreMessageSuccess:      1.5,
		ScoreReplySuccess:        2.0,
		ScoreReactionSuccess:     0.5,
		ScoreDailyActiveBonus:    3.0,
		ScoreFailurePenalty:      2.0,
		ScoreDisconnectedPenalty: 5.0,
	}, logs, phones)

	score, status, err := service.Recalculate(ctx, phone.ID)
	if err != nil {
		t.Fatalf("Recalculate() error = %v", err)
	}
	if status != "warm" {
		t.Fatalf("status = %q", status)
	}
	if score <= 4 {
		t.Fatalf("score = %v", score)
	}

	updated, err := phones.GetByID(ctx, phone.ID)
	if err != nil {
		t.Fatalf("phones.GetByID() error = %v", err)
	}
	if updated.Status != "warm" {
		t.Fatalf("updated status = %q", updated.Status)
	}
	if updated.WarmingScore != score {
		t.Fatalf("updated score = %v want %v", updated.WarmingScore, score)
	}
}
