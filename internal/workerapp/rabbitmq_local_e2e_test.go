//go:build integration

package workerapp

import (
	"context"
	"os"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"aquecedor-evolution/backend/internal/config"
	"aquecedor-evolution/backend/internal/database"
	"aquecedor-evolution/backend/internal/queue"
	"aquecedor-evolution/backend/internal/repository"
	"aquecedor-evolution/backend/internal/runner"
	schedulerpkg "aquecedor-evolution/backend/internal/scheduler"
	"aquecedor-evolution/backend/internal/worker"
)

type amqpIntegrationDelivery struct {
	delivery amqp.Delivery
}

func (d amqpIntegrationDelivery) Body() []byte { return d.delivery.Body }
func (d amqpIntegrationDelivery) Ack() error   { return d.delivery.Ack(false) }
func (d amqpIntegrationDelivery) Nack(requeue bool) error {
	return d.delivery.Nack(false, requeue)
}
func (d amqpIntegrationDelivery) Attempt() int { return 1 }

type testRunFilteredDueJobStore struct {
	repo      repository.WarmingJobRepository
	testRunID string
}

func (s testRunFilteredDueJobStore) ListDuePending(ctx context.Context, now time.Time, limit int) ([]repository.WarmingJob, error) {
	items, err := s.repo.ListDuePending(ctx, now, limit)
	if err != nil {
		return nil, err
	}
	filtered := make([]repository.WarmingJob, 0, len(items))
	for _, item := range items {
		value, _ := item.Metadata["testRunId"].(string)
		if value == s.testRunID {
			filtered = append(filtered, item)
		}
	}
	return filtered, nil
}

func TestRabbitMQLocalRealFlow(t *testing.T) {
	if os.Getenv("ENABLE_REAL_TESTS") != "true" {
		t.Skip("set ENABLE_REAL_TESTS=true to run real database integration tests")
	}

	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@127.0.0.1:5672/"
	}

	testRunID := os.Getenv("TEST_RUN_ID")
	if testRunID == "" {
		testRunID = "worker-rabbit-local-" + time.Now().UTC().Format("20060102T150405")
	}

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

	broker, err := queue.DialRabbitMQ(rabbitURL)
	if err != nil {
		t.Fatalf("DialRabbitMQ() error = %v", err)
	}
	defer broker.Close()

	topologySuffix := sanitizeQueueSuffix(testRunID)
	topology := queue.DefaultTopology(queue.TopologyConfig{
		Exchange:             "aquecedor.test." + topologySuffix + ".events",
		WarmingJobsQueue:     "aquecedor.test." + topologySuffix + ".warming.jobs",
		EvolutionEventsQueue: "aquecedor.test." + topologySuffix + ".evolution.events",
		DeadLetterQueue:      "aquecedor.test." + topologySuffix + ".dead_letter",
	})
	if err := queue.DeclareTopology(ctx, broker, topology); err != nil {
		t.Fatalf("DeclareTopology() error = %v", err)
	}
	if err := broker.SetPrefetch(1); err != nil {
		t.Fatalf("SetPrefetch() error = %v", err)
	}

	exec := repository.NewPgxExecutor(pool)
	scripts := repository.NewConversationScriptRepository(exec)
	steps := repository.NewConversationStepRepository(exec)
	jobs := repository.NewWarmingJobRepository(exec)
	logs := repository.NewExecutionLogRepository(exec)
	instances := repository.NewInstanceRepository(exec)
	servers := repository.NewEvolutionServerRepository(exec)

	scriptName := "worker-rabbit-local-script-" + testRunID
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		_, _ = exec.Exec(cleanupCtx, `
delete from public.execution_logs
where warming_job_id in (
  select id from public.warming_jobs where metadata ->> 'testRunId' = $1
)`, testRunID)
		_, _ = jobs.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = scripts.DeleteByName(cleanupCtx, scriptName)
	}()

	var phoneAID string
	var phoneBID string
	row := exec.QueryRow(ctx, `
select
  max(case when phone_e164 = '5519989411105' then id::text end),
  max(case when phone_e164 = '5519995081355' then id::text end)
from public.phone_numbers
where phone_e164 in ('5519989411105','5519995081355')
`)
	if err := row.Scan(&phoneAID, &phoneBID); err != nil {
		t.Fatalf("load real phone ids error = %v", err)
	}
	if phoneAID == "" || phoneBID == "" {
		t.Fatalf("real phone ids not found: phoneA=%q phoneB=%q", phoneAID, phoneBID)
	}

	script, err := scripts.Create(ctx, repository.CreateConversationScriptParams{
		Name:            scriptName,
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
		ActionType:      "send_typing",
		Payload:         map[string]any{"number": "5519995081355", "delay": 900},
		MinDelaySeconds: 1,
		MaxDelaySeconds: 1,
	})
	if err != nil {
		t.Fatalf("steps.Create() error = %v", err)
	}

	job, err := jobs.Create(ctx, repository.CreateWarmingJobParams{
		ScriptID:    &script.ID,
		PhoneAID:    phoneAID,
		PhoneBID:    phoneBID,
		ScheduledAt: time.Now().UTC().Add(-time.Second),
		Metadata:    map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("jobs.Create() error = %v", err)
	}

	publisher := queue.NewPublisher(broker, topology.Exchange.Name)
	s := schedulerpkg.NewWarmingJobScheduler(testRunFilteredDueJobStore{
		repo:      jobs,
		testRunID: testRunID,
	}, publisher, 10)
	published, err := s.PublishDue(ctx, time.Now().UTC())
	if err != nil {
		t.Fatalf("PublishDue() error = %v", err)
	}
	if published != 1 {
		t.Fatalf("published = %d", published)
	}

	deliveries, err := broker.Consume(topology.Queues[0].Name)
	if err != nil {
		t.Fatalf("Consume() error = %v", err)
	}

	executors := runner.NewInstanceExecutorFactory(servers, localRabbitSecretResolver{}, runner.DefaultStepClientFactory{})
	jobRunner := runner.NewWarmingJobRunner(jobs, steps, instances, executors, logs, nil)
	jobWorker := worker.NewWarmingJobWorker(jobRunner)
	consumer := queue.NewWarmingJobDueConsumer(jobWorker)

	var delivery amqp.Delivery
	select {
	case delivery = <-deliveries:
	case <-time.After(15 * time.Second):
		t.Fatal("timeout waiting rabbitmq delivery")
	}

	if err := ProcessDelivery(ctx, consumer, amqpIntegrationDelivery{delivery: delivery}, 3); err != nil {
		t.Fatalf("ProcessDelivery() error = %v", err)
	}

	updatedJob, err := jobs.GetByID(ctx, job.ID)
	if err != nil {
		t.Fatalf("jobs.GetByID() error = %v", err)
	}
	if updatedJob.Status != "success" {
		t.Fatalf("updatedJob.Status = %q error=%q", updatedJob.Status, updatedJob.Error)
	}

	items, err := logs.List(ctx)
	if err != nil {
		t.Fatalf("logs.List() error = %v", err)
	}

	found := false
	for _, item := range items {
		if item.WarmingJobID != nil && *item.WarmingJobID == job.ID && item.Status == "success" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("success log for rabbitmq processed job was not found")
	}
}

type localRabbitSecretResolver struct{}

func (localRabbitSecretResolver) Resolve(secretName string) string {
	value := os.Getenv(secretName)
	if value == "" {
		return os.Getenv("AUTHENTICATION_API_KEY")
	}
	return value
}

func sanitizeQueueSuffix(value string) string {
	result := make([]rune, 0, len(value))
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			result = append(result, r)
		case r >= 'A' && r <= 'Z':
			result = append(result, r+('a'-'A'))
		case r >= '0' && r <= '9':
			result = append(result, r)
		default:
			result = append(result, '.')
		}
	}
	return string(result)
}
