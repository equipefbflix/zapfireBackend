//go:build integration

package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"aquecedor-evolution/backend/internal/config"
	"aquecedor-evolution/backend/internal/conversationloop"
	"aquecedor-evolution/backend/internal/database"
	"aquecedor-evolution/backend/internal/evolution"
	"aquecedor-evolution/backend/internal/evolutionsync"
	"aquecedor-evolution/backend/internal/instance"
	"aquecedor-evolution/backend/internal/planner"
	"aquecedor-evolution/backend/internal/queue"
	"aquecedor-evolution/backend/internal/repository"
	"aquecedor-evolution/backend/internal/runner"
	schedulerpkg "aquecedor-evolution/backend/internal/scheduler"
	"aquecedor-evolution/backend/internal/warmingscore"
	"aquecedor-evolution/backend/internal/worker"
	"aquecedor-evolution/backend/internal/workerapp"
)

type reactiveLoopDelivery struct {
	delivery amqp.Delivery
}

func (d reactiveLoopDelivery) Body() []byte { return d.delivery.Body }
func (d reactiveLoopDelivery) Ack() error   { return d.delivery.Ack(false) }
func (d reactiveLoopDelivery) Nack(requeue bool) error {
	return d.delivery.Nack(false, requeue)
}
func (d reactiveLoopDelivery) Attempt() int { return 1 }

func TestReactiveLoopRealE2E(t *testing.T) {
	if os.Getenv("ENABLE_REAL_TESTS") != "true" {
		t.Skip("set ENABLE_REAL_TESTS=true to run real database integration tests")
	}

	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@127.0.0.1:5672/"
	}

	apiKey := firstEnv("EVOLUTION_TEST_API_KEY", "AUTHENTICATION_API_KEY")
	if apiKey == "" {
		t.Fatal("EVOLUTION_TEST_API_KEY or AUTHENTICATION_API_KEY is required")
	}

	testRunID := os.Getenv("TEST_RUN_ID")
	if testRunID == "" {
		testRunID = "reactive-loop-e2e-" + time.Now().UTC().Format("20060102T150405")
	}

	dbConfig, err := config.LoadDatabaseConfig()
	if err != nil {
		t.Fatalf("LoadDatabaseConfig() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	pool, err := database.Open(ctx, dbConfig)
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	defer pool.Close()

	broker, err := queue.DialRabbitMQ(rabbitURL)
	if err != nil {
		t.Fatalf("DialRabbitMQ() error = %v", err)
	}
	defer broker.Close()

	fixtureA := firstEnv("REACTIVE_FIXTURE_A_NAME")
	if fixtureA == "" {
		fixtureA = "reactive_a_go_20260515"
	}
	fixtureB := firstEnv("REACTIVE_FIXTURE_B_NAME")
	if fixtureB == "" {
		fixtureB = "reactive_b_go_20260515"
	}
	phoneAE164 := firstEnv("REACTIVE_FIXTURE_A_PHONE")
	if phoneAE164 == "" {
		phoneAE164 = "5519989411105"
	}
	phoneBE164 := firstEnv("REACTIVE_FIXTURE_B_PHONE")
	if phoneBE164 == "" {
		phoneBE164 = "5519995081355"
	}

	topology := queue.DefaultTopology(queue.TopologyConfig{
		Exchange:             "aquecedor.test." + testRunID + ".events",
		WarmingJobsQueue:     "aquecedor.test." + testRunID + ".warming.jobs",
		EvolutionEventsQueue: "aquecedor.test." + testRunID + ".evolution.events",
		DeadLetterQueue:      "aquecedor.test." + testRunID + ".dead_letter",
	})
	if err := queue.DeclareTopology(ctx, broker, topology); err != nil {
		t.Fatalf("DeclareTopology() error = %v", err)
	}
	if err := broker.SetPrefetch(1); err != nil {
		t.Fatalf("SetPrefetch() error = %v", err)
	}

	exec := repository.NewPgxExecutor(pool)
	phones := repository.NewPhoneNumberRepository(exec)
	servers := repository.NewEvolutionServerRepository(exec)
	instances := repository.NewInstanceRepository(exec)
	scripts := repository.NewConversationScriptRepository(exec)
	steps := repository.NewConversationStepRepository(exec)
	jobs := repository.NewWarmingJobRepository(exec)
	logs := repository.NewExecutionLogRepository(exec)
	events := repository.NewEvolutionEventRepository(exec)
	scriptName := "reactive-loop-e2e-" + testRunID

	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()

		_, _ = exec.Exec(cleanupCtx, `
delete from public.execution_logs
where warming_job_id in (
  select id from public.warming_jobs where metadata ->> 'testRunId' = $1
)`, testRunID)
		_, _ = exec.Exec(cleanupCtx, `
delete from public.evolution_events
where payload ->> 'testRunId' = $1
`, testRunID)
		_, _ = jobs.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = scripts.DeleteByName(cleanupCtx, scriptName)
	}()

	phoneA, err := phones.FindByE164(ctx, phoneAE164)
	if err != nil {
		t.Fatalf("phones.FindByE164(%s) error = %v", phoneAE164, err)
	}
	if phoneA == nil {
		t.Fatalf("managed phone A not found for %s", phoneAE164)
	}
	phoneB, err := phones.FindByE164(ctx, phoneBE164)
	if err != nil {
		t.Fatalf("phones.FindByE164(%s) error = %v", phoneBE164, err)
	}
	if phoneB == nil {
		t.Fatalf("managed phone B not found for %s", phoneBE164)
	}

	instanceA, err := instances.GetByInstanceName(ctx, fixtureA)
	if err != nil {
		t.Fatalf("GetByInstanceName(%s) error = %v", fixtureA, err)
	}
	instanceB, err := instances.GetByInstanceName(ctx, fixtureB)
	if err != nil {
		t.Fatalf("GetByInstanceName(%s) error = %v", fixtureB, err)
	}

	serverA, err := servers.GetByID(ctx, instanceA.EvolutionServerID)
	if err != nil {
		t.Fatalf("servers.GetByID(%s) error = %v", instanceA.EvolutionServerID, err)
	}
	serverB, err := servers.GetByID(ctx, instanceB.EvolutionServerID)
	if err != nil {
		t.Fatalf("servers.GetByID(%s) error = %v", instanceB.EvolutionServerID, err)
	}
	requireReactiveFixtureOpen(t, ctx, serverA, instanceA)
	requireReactiveFixtureOpen(t, ctx, serverB, instanceB)

	script, err := scripts.Create(ctx, repository.CreateConversationScriptParams{
		Name:            scriptName,
		Category:        "reactive",
		Enabled:         true,
		Weight:          999999,
		MinWarmingScore: 0,
		MaxWarmingScore: 100,
	})
	if err != nil {
		t.Fatalf("scripts.Create() error = %v", err)
	}
	if _, err := steps.Create(ctx, repository.CreateConversationStepParams{
		ScriptID:        script.ID,
		StepOrder:       1,
		SenderRole:      "a",
		ActionType:      "send_typing",
		Payload:         map[string]any{"number": "{{phoneB}}", "delay": 900},
		MinDelaySeconds: 0,
		MaxDelaySeconds: 0,
	}); err != nil {
		t.Fatalf("steps.Create() error = %v", err)
	}

	scoreService := warmingscore.NewService(config.LoadWarmingConfig(), logs, phones)
	plannerService := planner.NewService(config.PlannerConfig{
		MinDelaySeconds:                  0,
		MaxDelaySeconds:                  0,
		PairCooldownMinutes:              30,
		InboundTriggerCooldownSeconds:    0,
		WindowStartHour:                  0,
		WindowEndHour:                    23,
		MaxRunningJobsPerPair:            1,
		MaxRunningJobsPerEvolutionServer: 5,
	}, scripts, jobs, nil)
	inboundLoop := conversationloop.NewService(config.PlannerConfig{InboundTriggerCooldownSeconds: 0}, instances, phones, plannerService, jobs, jobs, nil)
	syncService := evolutionsync.NewService(instances, logs, events, scoreService, inboundLoop)

	server := NewServer(ServerConfig{
		App:             config.AppConfig{},
		EvolutionEvents: events,
		EvolutionSync:   syncService,
	})

	payload := map[string]any{
		"testRunId": testRunID,
		"event":     "MESSAGES_UPSERT",
		"instance":  fixtureA,
		"data": map[string]any{
			"key": map[string]any{
				"id":        "msg-" + testRunID,
				"fromMe":    false,
				"remoteJid": phoneBE164 + "@s.whatsapp.net",
			},
			"messageType": "conversation",
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/evolution", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}

	var jobID string
	row := exec.QueryRow(ctx, `
select id::text
from public.warming_jobs
where metadata ->> 'testRunId' = $1
order by created_at desc
limit 1
`, testRunID)
	if err := row.Scan(&jobID); err != nil {
		t.Fatalf("expected warming job for testRunId %s: %v", testRunID, err)
	}

	job, err := jobs.GetByID(ctx, jobID)
	if err != nil {
		t.Fatalf("jobs.GetByID() error = %v", err)
	}
	if job.Metadata["autoReactive"] != true {
		t.Fatalf("autoReactive = %#v", job.Metadata["autoReactive"])
	}

	publisher := queue.NewPublisher(broker, topology.Exchange.Name)
	scheduler := schedulerpkg.NewWarmingJobScheduler(jobs, publisher, 10)
	published, err := scheduler.PublishDue(ctx, time.Now().UTC())
	if err != nil {
		t.Fatalf("PublishDue() error = %v", err)
	}
	if published < 1 {
		t.Fatalf("published = %d", published)
	}

	deliveries, err := broker.Consume(topology.Queues[0].Name)
	if err != nil {
		t.Fatalf("Consume() error = %v", err)
	}

	t.Setenv("AUTHENTICATION_API_KEY", apiKey)
	t.Setenv("EVOLUTION_TEST_API_KEY", apiKey)
	executors := runner.NewInstanceExecutorFactory(servers, instance.EnvSecretResolver{}, runner.DefaultStepClientFactory{})
	jobRunner := runner.NewWarmingJobRunner(jobs, steps, instances, executors, logs, nil)
	jobWorker := worker.NewWarmingJobWorker(jobRunner)
	consumer := queue.NewWarmingJobDueConsumer(jobWorker)

	var delivery amqp.Delivery
	foundDelivery := false
	select {
	case first := <-deliveries:
		delivery = first
	case <-time.After(20 * time.Second):
		t.Fatal("timeout waiting reactive rabbitmq delivery")
	}

	deadline := time.Now().Add(20 * time.Second)
deliveryLoop:
	for {
		var dueMsg queue.WarmingJobDueMessage
		if err := json.Unmarshal(delivery.Body, &dueMsg); err != nil {
			t.Fatalf("Unmarshal(delivery) error = %v", err)
		}
		if dueMsg.JobID == job.ID {
			foundDelivery = true
			break
		}
		if err := delivery.Nack(false, true); err != nil {
			t.Fatalf("Nack(requeue) error = %v", err)
		}
		if time.Now().After(deadline) {
			break
		}
		select {
		case delivery = <-deliveries:
		case <-time.After(time.Until(deadline)):
			break deliveryLoop
		}
	}
	if !foundDelivery {
		t.Fatalf("did not receive queue delivery for job %s", job.ID)
	}

	if err := workerapp.ProcessDelivery(ctx, consumer, reactiveLoopDelivery{delivery: delivery}, 3); err != nil {
		t.Fatalf("ProcessDelivery() error = %v", err)
	}

	var updatedJob repository.WarmingJob
	waitUntil := time.Now().Add(5 * time.Second)
	for {
		updatedJob, err = jobs.GetByID(ctx, job.ID)
		if err != nil {
			t.Fatalf("jobs.GetByID() after worker error = %v", err)
		}
		if updatedJob.Status == "success" {
			break
		}
		if time.Now().After(waitUntil) {
			t.Fatalf("updatedJob.Status = %q error=%q", updatedJob.Status, updatedJob.Error)
		}
		time.Sleep(200 * time.Millisecond)
	}

	logItems, err := logs.List(ctx)
	if err != nil {
		t.Fatalf("logs.List() error = %v", err)
	}

	found := false
	for _, item := range logItems {
		if item.WarmingJobID != nil && *item.WarmingJobID == job.ID && item.Status == "success" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("success log for reactive processed job was not found")
	}
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return ""
}

func requireReactiveFixtureOpen(t *testing.T, ctx context.Context, server repository.EvolutionServer, inst repository.Instance) {
	t.Helper()

	apiKey := resolveInstanceAPIKey(inst.InstanceAPIKeySecretName)
	if apiKey == "" {
		t.Fatalf("instance %s has no resolvable api key secret", inst.InstanceName)
	}

	client := evolution.NewClient(evolution.Config{
		BaseURL: server.BaseURL,
		APIKey:  apiKey,
		Timeout: 30 * time.Second,
	})

	state, err := client.ConnectionState(ctx, inst.InstanceName)
	if err == nil && state.Instance.State == "open" && state.Instance.LoggedIn && state.Instance.JID != "" {
		return
	}

	connectResponse, connectErr := client.ConnectInstance(ctx, inst.InstanceName, "")
	if connectErr != nil {
		t.Fatalf("fixture %s is not open and connect failed: stateErr=%v connectErr=%v", inst.InstanceName, err, connectErr)
	}

	t.Fatalf(
		"fixture %s is not connected in evolution-go; pairingCode=%q code=%q stateErr=%v",
		inst.InstanceName,
		connectResponse.PairingCode,
		connectResponse.Code,
		err,
	)
}

func resolveInstanceAPIKey(secretName *string) string {
	if secretName == nil {
		return ""
	}
	if value, ok := instanceLiteralSecret(*secretName); ok {
		return value
	}
	return os.Getenv(*secretName)
}

func instanceLiteralSecret(secretName string) (string, bool) {
	const prefix = "literal:"
	if len(secretName) <= len(prefix) || secretName[:len(prefix)] != prefix {
		return "", false
	}
	return secretName[len(prefix):], true
}
