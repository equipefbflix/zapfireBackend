//go:build integration

package workerapp

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"aquecedor-evolution/backend/internal/config"
	"aquecedor-evolution/backend/internal/database"
	"aquecedor-evolution/backend/internal/evolution"
	"aquecedor-evolution/backend/internal/executor"
	"aquecedor-evolution/backend/internal/queue"
	"aquecedor-evolution/backend/internal/repository"
	"aquecedor-evolution/backend/internal/runner"
	"aquecedor-evolution/backend/internal/worker"
)

type fakeIntegrationDelivery struct {
	body        []byte
	acked       bool
	nacked      bool
	nackRequeue bool
}

func (d *fakeIntegrationDelivery) Body() []byte { return d.body }
func (d *fakeIntegrationDelivery) Ack() error   { d.acked = true; return nil }
func (d *fakeIntegrationDelivery) Nack(requeue bool) error {
	d.nacked = true
	d.nackRequeue = requeue
	return nil
}
func (d *fakeIntegrationDelivery) Attempt() int { return 1 }

type integrationSecretResolver struct{}

func (integrationSecretResolver) Resolve(secretName string) string {
	return "integration-" + secretName
}

type integrationStepClientFactory struct{}

func (integrationStepClientFactory) New(server repository.EvolutionServer, apiKey string) executor.EvolutionStepClient {
	return integrationStepClient{}
}

type integrationStepClient struct{}

func (integrationStepClient) SendText(ctx context.Context, instanceName string, request evolution.SendTextRequest) (evolution.SendMessageResponse, error) {
	return evolution.SendMessageResponse{Key: evolution.MessageKey{ID: "integration-message-id"}}, nil
}

func (integrationStepClient) SendMedia(ctx context.Context, instanceName string, request evolution.SendMediaRequest) (evolution.SendMessageResponse, error) {
	return evolution.SendMessageResponse{Key: evolution.MessageKey{ID: "integration-media-id"}}, nil
}

func (integrationStepClient) SendWhatsAppAudio(ctx context.Context, instanceName string, request evolution.SendWhatsAppAudioRequest) (evolution.SendMessageResponse, error) {
	return evolution.SendMessageResponse{Key: evolution.MessageKey{ID: "integration-audio-id"}}, nil
}

func (integrationStepClient) SendStatus(ctx context.Context, instanceName string, request evolution.SendStatusRequest) (evolution.SendMessageResponse, error) {
	return evolution.SendMessageResponse{AcceptedAsync: true}, nil
}

func (integrationStepClient) SendSticker(ctx context.Context, instanceName string, request evolution.SendStickerRequest) error {
	return nil
}

func (integrationStepClient) SendPresence(ctx context.Context, instanceName string, request evolution.SendPresenceRequest) error {
	return nil
}

func (integrationStepClient) SendReaction(ctx context.Context, instanceName string, request evolution.SendReactionRequest) error {
	return nil
}

func TestWorkerLocalFlowRealDatabase(t *testing.T) {
	if os.Getenv("ENABLE_REAL_TESTS") != "true" {
		t.Skip("set ENABLE_REAL_TESTS=true to run real database integration tests")
	}

	testRunID := os.Getenv("TEST_RUN_ID")
	if testRunID == "" {
		testRunID = "worker-local-" + time.Now().UTC().Format("20060102T150405")
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

	exec := repository.NewPgxExecutor(pool)
	phones := repository.NewPhoneNumberRepository(exec)
	servers := repository.NewEvolutionServerRepository(exec)
	instances := repository.NewInstanceRepository(exec)
	scripts := repository.NewConversationScriptRepository(exec)
	steps := repository.NewConversationStepRepository(exec)
	jobs := repository.NewWarmingJobRepository(exec)
	logs := repository.NewExecutionLogRepository(exec)

	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		_, _ = exec.Exec(cleanupCtx, `
delete from public.execution_logs
where warming_job_id in (
  select id from public.warming_jobs where metadata ->> 'testRunId' = $1
)`, testRunID)
		_, _ = jobs.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = exec.Exec(cleanupCtx, `
delete from public.conversation_scripts
where name = $1
`, "worker-local-script-"+testRunID)
		_, _ = instances.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = phones.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = servers.DeleteByTestRunID(cleanupCtx, testRunID)
	}()

	server, err := servers.Create(ctx, repository.CreateEvolutionServerParams{
		Name:              "worker-local-evo-" + testRunID,
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

	phoneA, err := phones.Create(ctx, repository.CreatePhoneNumberParams{
		PhoneE164: "+550000000101",
		Label:     "worker local phone a",
		Metadata:  map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("phones.Create(phoneA) error = %v", err)
	}
	phoneB, err := phones.Create(ctx, repository.CreatePhoneNumberParams{
		PhoneE164: "+550000000102",
		Label:     "worker local phone b",
		Metadata:  map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("phones.Create(phoneB) error = %v", err)
	}

	instanceA, err := instances.Create(ctx, repository.CreateInstanceParams{
		PhoneNumberID:     phoneA.ID,
		EvolutionServerID: server.ID,
		InstanceName:      "worker-local-instance-a-" + testRunID,
		Status:            "open",
		Metadata:          map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("instances.Create(instanceA) error = %v", err)
	}
	_, err = instances.Create(ctx, repository.CreateInstanceParams{
		PhoneNumberID:     phoneB.ID,
		EvolutionServerID: server.ID,
		InstanceName:      "worker-local-instance-b-" + testRunID,
		Status:            "open",
		Metadata:          map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("instances.Create(instanceB) error = %v", err)
	}

	script, err := scripts.Create(ctx, repository.CreateConversationScriptParams{
		Name:            "worker-local-script-" + testRunID,
		Category:        "integration",
		Enabled:         true,
		Weight:          1,
		MinWarmingScore: 0,
		MaxWarmingScore: 100,
	})
	if err != nil {
		t.Fatalf("scripts.Create() error = %v", err)
	}
	_, err = steps.Create(ctx, repository.CreateConversationStepParams{
		ScriptID:        script.ID,
		StepOrder:       1,
		SenderRole:      "a",
		ActionType:      "send_presence",
		Payload:         map[string]any{"number": "550000000102", "presence": "composing", "delay": 1000},
		MinDelaySeconds: 1,
		MaxDelaySeconds: 1,
	})
	if err != nil {
		t.Fatalf("steps.Create() error = %v", err)
	}

	job, err := jobs.Create(ctx, repository.CreateWarmingJobParams{
		ScriptID:    &script.ID,
		PhoneAID:    phoneA.ID,
		PhoneBID:    phoneB.ID,
		ScheduledAt: time.Now().UTC(),
		Metadata:    map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("jobs.Create() error = %v", err)
	}

	executors := runner.NewInstanceExecutorFactory(servers, integrationSecretResolver{}, integrationStepClientFactory{})
	jobRunner := runner.NewWarmingJobRunner(jobs, steps, instances, executors, logs, nil)
	jobWorker := worker.NewWarmingJobWorker(jobRunner)
	consumer := queue.NewWarmingJobDueConsumer(jobWorker)

	body, err := json.Marshal(queue.NewWarmingJobDueMessage(job.ID, testRunID, time.Now().UTC()))
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	delivery := &fakeIntegrationDelivery{body: body}

	if err := ProcessDelivery(ctx, consumer, delivery, 3); err != nil {
		t.Fatalf("ProcessDelivery() error = %v", err)
	}
	if !delivery.acked {
		t.Fatal("acked = false")
	}
	if delivery.nacked {
		t.Fatal("nacked = true")
	}

	items, err := logs.List(ctx)
	if err != nil {
		t.Fatalf("logs.List() error = %v", err)
	}

	found := false
	for _, item := range items {
		if item.WarmingJobID != nil && *item.WarmingJobID == job.ID && item.InstanceID != nil && *item.InstanceID == instanceA.ID && item.Status == "success" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("success log for processed job was not found")
	}
}
