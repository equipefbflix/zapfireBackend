//go:build integration

package evolutionsync

import (
	"context"
	"os"
	"testing"
	"time"

	"aquecedor-evolution/backend/internal/config"
	"aquecedor-evolution/backend/internal/database"
	"aquecedor-evolution/backend/internal/repository"
	"aquecedor-evolution/backend/internal/warmingscore"
)

func TestServiceSyncRecalculatesScoreRealDatabase(t *testing.T) {
	if os.Getenv("ENABLE_REAL_TESTS") != "true" {
		t.Skip("set ENABLE_REAL_TESTS=true to run real database integration tests")
	}

	testRunID := os.Getenv("TEST_RUN_ID")
	if testRunID == "" {
		testRunID = "evolutionsync-" + time.Now().UTC().Format("20060102T150405")
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
	events := repository.NewEvolutionEventRepository(executor)

	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		_, _ = executor.Exec(cleanupCtx, `
delete from public.execution_logs
where instance_id in (
  select id from public.instances where metadata ->> 'testRunId' = $1
)
`, testRunID)
		_, _ = executor.Exec(cleanupCtx, `
delete from public.evolution_events
where instance_name like $1
`, "sync-score-%"+testRunID+"%")
		_, _ = instances.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = phones.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = servers.DeleteByTestRunID(cleanupCtx, testRunID)
	}()

	phone, err := phones.Create(ctx, repository.CreatePhoneNumberParams{
		PhoneE164: "+550000000201",
		Label:     "sync_score_phone",
		Metadata:  map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("phones.Create() error = %v", err)
	}

	server, err := servers.Create(ctx, repository.CreateEvolutionServerParams{
		Name:              "sync-score-evo-" + testRunID,
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

	instanceName := "sync-score-" + testRunID
	instance, err := instances.Create(ctx, repository.CreateInstanceParams{
		PhoneNumberID:     phone.ID,
		EvolutionServerID: server.ID,
		InstanceName:      instanceName,
		Status:            "open",
		Metadata:          map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("instances.Create() error = %v", err)
	}

	actionText := "send_text"
	if _, err := logs.Create(ctx, repository.CreateExecutionLogParams{
		InstanceID:          &instance.ID,
		ActionType:          &actionText,
		Status:              "running",
		EvolutionMessageKey: map[string]any{"id": "message-id"},
	}); err != nil {
		t.Fatalf("logs.Create() error = %v", err)
	}

	event, err := events.Create(ctx, repository.CreateEvolutionEventParams{
		InstanceName: instanceName,
		EventType:    "MESSAGES_UPDATE",
		Payload: map[string]any{
			"data": map[string]any{
				"key": map[string]any{
					"id":        "message-id",
					"remoteJid": "5511999999999@s.whatsapp.net",
				},
				"status": "delivered",
			},
		},
	})
	if err != nil {
		t.Fatalf("events.Create() error = %v", err)
	}

	scoreService := warmingscore.NewService(config.WarmingConfig{
		MinScoreToMarkWarm:       1,
		ScoreMessageSuccess:      1.5,
		ScoreReplySuccess:        2.0,
		ScoreReactionSuccess:     0.5,
		ScoreDailyActiveBonus:    3.0,
		ScoreFailurePenalty:      2.0,
		ScoreDisconnectedPenalty: 5.0,
	}, logs, phones)
	service := NewService(instances, logs, events, scoreService, nil)

	if err := service.Sync(ctx, event); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	updatedPhone, err := phones.GetByID(ctx, phone.ID)
	if err != nil {
		t.Fatalf("phones.GetByID() error = %v", err)
	}
	if updatedPhone.WarmingScore <= 0 {
		t.Fatalf("warming score = %v", updatedPhone.WarmingScore)
	}
	if updatedPhone.Status != "warm" {
		t.Fatalf("phone status = %q", updatedPhone.Status)
	}

	var processedAt *time.Time
	if err := executor.QueryRow(ctx, `
select processed_at
from public.evolution_events
where id = $1
`, event.ID).Scan(&processedAt); err != nil {
		t.Fatalf("query processed_at error = %v", err)
	}
	if processedAt == nil {
		t.Fatal("processed_at is nil")
	}
}
