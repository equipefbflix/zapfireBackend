//go:build integration

package conversationloop

import (
	"context"
	"os"
	"testing"
	"time"

	"aquecedor-evolution/backend/internal/config"
	"aquecedor-evolution/backend/internal/database"
	"aquecedor-evolution/backend/internal/planner"
	"aquecedor-evolution/backend/internal/repository"
)

func TestServiceHandleInboundRealDatabase(t *testing.T) {
	if os.Getenv("ENABLE_REAL_TESTS") != "true" {
		t.Skip("set ENABLE_REAL_TESTS=true to run real database integration tests")
	}

	testRunID := os.Getenv("TEST_RUN_ID")
	if testRunID == "" {
		testRunID = "conversationloop-" + time.Now().UTC().Format("20060102T150405")
	}
	suffix := time.Now().UTC().Format("150405")
	phoneAE164 := "550000" + suffix
	phoneBE164 := "551000" + suffix

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
	scripts := repository.NewConversationScriptRepository(executor)
	steps := repository.NewConversationStepRepository(executor)
	jobs := repository.NewWarmingJobRepository(executor)

	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		_, _ = executor.Exec(cleanupCtx, `
delete from public.conversation_steps
where script_id in (
  select id from public.conversation_scripts where name like $1
)
`, "reactive-loop-%"+testRunID+"%")
		_, _ = executor.Exec(cleanupCtx, `
delete from public.warming_jobs
where metadata ->> 'testRunId' = $1
`, testRunID)
		_, _ = scripts.DeleteByName(cleanupCtx, "reactive-loop-"+testRunID)
		_, _ = instances.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = phones.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = servers.DeleteByTestRunID(cleanupCtx, testRunID)
	}()

	phoneA, err := phones.Create(ctx, repository.CreatePhoneNumberParams{
		PhoneE164: phoneAE164,
		Label:     "reactive_a",
		Metadata:  map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("phones.Create(A) error = %v", err)
	}
	phoneB, err := phones.Create(ctx, repository.CreatePhoneNumberParams{
		PhoneE164: phoneBE164,
		Label:     "reactive_b",
		Metadata:  map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("phones.Create(B) error = %v", err)
	}

	if err := phones.UpdateWarmingState(ctx, phoneA.ID, 12, "warming"); err != nil {
		t.Fatalf("phones.UpdateWarmingState(A) error = %v", err)
	}
	if err := phones.UpdateWarmingState(ctx, phoneB.ID, 18, "warming"); err != nil {
		t.Fatalf("phones.UpdateWarmingState(B) error = %v", err)
	}

	server, err := servers.Create(ctx, repository.CreateEvolutionServerParams{
		Name:              "reactive-loop-evo-" + testRunID,
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

	if _, err := instances.Create(ctx, repository.CreateInstanceParams{
		PhoneNumberID:     phoneA.ID,
		EvolutionServerID: server.ID,
		InstanceName:      "reactive-loop-a-" + testRunID,
		Status:            "open",
		Metadata:          map[string]any{"testRunId": testRunID},
	}); err != nil {
		t.Fatalf("instances.Create(A) error = %v", err)
	}
	if _, err := instances.Create(ctx, repository.CreateInstanceParams{
		PhoneNumberID:     phoneB.ID,
		EvolutionServerID: server.ID,
		InstanceName:      "reactive-loop-b-" + testRunID,
		Status:            "open",
		Metadata:          map[string]any{"testRunId": testRunID},
	}); err != nil {
		t.Fatalf("instances.Create(B) error = %v", err)
	}

	script, err := scripts.Create(ctx, repository.CreateConversationScriptParams{
		Name:            "reactive-loop-" + testRunID,
		Category:        "reactive",
		Enabled:         true,
		Weight:          10,
		MinWarmingScore: 11.5,
		MaxWarmingScore: 12.5,
	})
	if err != nil {
		t.Fatalf("scripts.Create() error = %v", err)
	}
	if _, err := steps.Create(ctx, repository.CreateConversationStepParams{
		ScriptID:        script.ID,
		StepOrder:       1,
		SenderRole:      "a",
		ActionType:      "send_typing",
		Payload:         map[string]any{"number": phoneBE164, "delay": 1000},
		MinDelaySeconds: 1,
		MaxDelaySeconds: 2,
	}); err != nil {
		t.Fatalf("steps.Create() error = %v", err)
	}

	plannerService := planner.NewService(config.PlannerConfig{
		MinDelaySeconds:               10,
		MaxDelaySeconds:               20,
		PairCooldownMinutes:           30,
		InboundTriggerCooldownSeconds: 90,
		WindowStartHour:               0,
		WindowEndHour:                 23,
	}, scripts, jobs, func() time.Time { return time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC) })
	service := NewService(config.PlannerConfig{InboundTriggerCooldownSeconds: 90}, instances, phones, plannerService, jobs, jobs, func() time.Time {
		return time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	})

	created, err := service.HandleInbound(ctx, repository.EvolutionEvent{
		ID:           "event-" + testRunID,
		InstanceName: "reactive-loop-a-" + testRunID,
		EventType:    "MESSAGES_UPSERT",
		Payload: map[string]any{
			"data": map[string]any{
				"key": map[string]any{
					"id":        "message-" + testRunID,
					"fromMe":    false,
					"remoteJid": phoneBE164 + "@s.whatsapp.net",
				},
				"messageType": "conversation",
			},
		},
	})
	if err != nil {
		t.Fatalf("HandleInbound() error = %v", err)
	}
	if created == nil {
		t.Fatal("created job is nil")
	}
	if created.PhoneAID != phoneA.ID {
		t.Fatalf("PhoneAID = %q", created.PhoneAID)
	}
	if created.PhoneBID != phoneB.ID {
		t.Fatalf("PhoneBID = %q", created.PhoneBID)
	}
	if created.ScriptID == nil || *created.ScriptID == "" {
		t.Fatalf("ScriptID = %v", created.ScriptID)
	}
	if created.Metadata["autoReactive"] != true {
		t.Fatalf("autoReactive = %#v", created.Metadata["autoReactive"])
	}
}
