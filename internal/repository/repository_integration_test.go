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

func TestRepositoriesRealDatabase(t *testing.T) {
	if os.Getenv("ENABLE_REAL_TESTS") != "true" {
		t.Skip("set ENABLE_REAL_TESTS=true to run real database integration tests")
	}

	testRunID := os.Getenv("TEST_RUN_ID")
	if testRunID == "" {
		testRunID = "repository-" + time.Now().UTC().Format("20060102T150405")
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
	proxies := NewProxyRepository(executor)
	instances := NewInstanceRepository(executor)
	jobs := NewWarmingJobRepository(executor)
	logs := NewExecutionLogRepository(executor)
	events := NewEvolutionEventRepository(executor)

	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		_, _ = executor.Exec(cleanupCtx, `
delete from public.evolution_events
where payload ->> 'testRunId' = $1
`, testRunID)
		_, _ = executor.Exec(cleanupCtx, `
delete from public.execution_logs
where warming_job_id in (
  select id from public.warming_jobs where metadata ->> 'testRunId' = $1
)
`, testRunID)
		_, _ = jobs.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = instances.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = phones.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = servers.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = proxies.DeleteByTestRunID(cleanupCtx, testRunID)
	}()

	phone, err := phones.Create(ctx, CreatePhoneNumberParams{
		PhoneE164: "+550000000001",
		Label:     "test_repository_phone",
		Metadata:  map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("phones.Create() error = %v", err)
	}
	if phone.ID == "" {
		t.Fatal("phone ID is empty")
	}

	phoneB, err := phones.Create(ctx, CreatePhoneNumberParams{
		PhoneE164: "+550000000002",
		Label:     "test_repository_phone_b",
		Metadata:  map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("phones.Create(phoneB) error = %v", err)
	}
	if phoneB.ID == "" {
		t.Fatal("phoneB ID is empty")
	}

	server, err := servers.Create(ctx, CreateEvolutionServerParams{
		Name:              "test_repository_evo_" + testRunID,
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
	if server.ID == "" {
		t.Fatal("server ID is empty")
	}

	proxy, err := proxies.Create(ctx, CreateProxyParams{
		Name:               "test_repository_proxy_" + testRunID,
		Host:               "proxy.example.com",
		Port:               8000,
		Protocol:           "http",
		Username:           stringPointer("user"),
		PasswordSecretName: stringPointer("PROXY_TEST_PASSWORD"),
		Enabled:            true,
		MaxInstances:       intPointer(20),
		Metadata:           map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("proxies.Create() error = %v", err)
	}
	if proxy.ID == "" {
		t.Fatal("proxy ID is empty")
	}

	enabledServers, err := servers.ListEnabled(ctx)
	if err != nil {
		t.Fatalf("servers.ListEnabled() error = %v", err)
	}
	if len(enabledServers) == 0 {
		t.Fatal("enabled servers is empty")
	}

	enabledProxies, err := proxies.ListEnabled(ctx)
	if err != nil {
		t.Fatalf("proxies.ListEnabled() error = %v", err)
	}
	if len(enabledProxies) == 0 {
		t.Fatal("enabled proxies is empty")
	}

	instanceA, err := instances.Create(ctx, CreateInstanceParams{
		PhoneNumberID:     phone.ID,
		EvolutionServerID: server.ID,
		InstanceName:      "test_repository_instance_a_" + testRunID,
		Status:            "open",
		Metadata:          map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("instances.Create(instanceA) error = %v", err)
	}
	if instanceA.ID == "" {
		t.Fatal("instanceA ID is empty")
	}

	instanceB, err := instances.Create(ctx, CreateInstanceParams{
		PhoneNumberID:     phoneB.ID,
		EvolutionServerID: server.ID,
		InstanceName:      "test_repository_instance_b_" + testRunID,
		Status:            "open",
		Metadata:          map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("instances.Create(instanceB) error = %v", err)
	}
	if instanceB.ID == "" {
		t.Fatal("instanceB ID is empty")
	}

	job, err := jobs.Create(ctx, CreateWarmingJobParams{
		PhoneAID:    phone.ID,
		PhoneBID:    phoneB.ID,
		ScheduledAt: time.Now().UTC(),
		Metadata:    map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("jobs.Create() error = %v", err)
	}
	if job.ID == "" {
		t.Fatal("job ID is empty")
	}

	log, err := logs.Create(ctx, CreateExecutionLogParams{
		WarmingJobID:   &job.ID,
		Status:         "running",
		RequestPayload: map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("logs.Create() error = %v", err)
	}
	if log.ID == "" {
		t.Fatal("log ID is empty")
	}

	failedAction := "send_text"
	_, err = logs.Create(ctx, CreateExecutionLogParams{
		WarmingJobID:   &job.ID,
		InstanceID:     &instanceA.ID,
		ActionType:     &failedAction,
		Status:         "failed",
		RequestPayload: map[string]any{"testRunId": testRunID},
		Error:          "integration failure",
	})
	if err != nil {
		t.Fatalf("logs.Create(failed) error = %v", err)
	}

	if err := jobs.UpdateStatus(ctx, job.ID, "running", ""); err != nil {
		t.Fatalf("jobs.UpdateStatus(running) error = %v", err)
	}

	runningCountByPair, err := jobs.CountRunningByPair(ctx, phone.ID, phoneB.ID)
	if err != nil {
		t.Fatalf("jobs.CountRunningByPair() error = %v", err)
	}
	if runningCountByPair != 1 {
		t.Fatalf("runningCountByPair = %d", runningCountByPair)
	}

	runningCountByServer, err := jobs.CountRunningByEvolutionServer(ctx, server.ID)
	if err != nil {
		t.Fatalf("jobs.CountRunningByEvolutionServer() error = %v", err)
	}
	if runningCountByServer != 1 {
		t.Fatalf("runningCountByServer = %d", runningCountByServer)
	}

	if err := jobs.UpdateStatus(ctx, job.ID, "failed", "integration failed"); err != nil {
		t.Fatalf("jobs.UpdateStatus(failed) error = %v", err)
	}

	updatedJob, err := jobs.GetByID(ctx, job.ID)
	if err != nil {
		t.Fatalf("jobs.GetByID() error = %v", err)
	}
	if updatedJob.Status != "failed" {
		t.Fatalf("updatedJob.Status = %q", updatedJob.Status)
	}
	if updatedJob.Error != "integration failed" {
		t.Fatalf("updatedJob.Error = %q", updatedJob.Error)
	}

	counts, err := jobs.CountByStatus(ctx)
	if err != nil {
		t.Fatalf("jobs.CountByStatus() error = %v", err)
	}
	if counts["failed"] < 1 {
		t.Fatalf("failed count = %d", counts["failed"])
	}

	event, err := events.Create(ctx, CreateEvolutionEventParams{
		EvolutionServerID: &server.ID,
		InstanceName:      instanceA.InstanceName,
		EventType:         "MESSAGES_UPSERT",
		Payload:           map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("events.Create() error = %v", err)
	}
	if event.ID == "" {
		t.Fatal("event ID is empty")
	}

	eventCount, err := events.CountSince(ctx, time.Now().UTC().Add(-time.Hour))
	if err != nil {
		t.Fatalf("events.CountSince() error = %v", err)
	}
	if eventCount < 1 {
		t.Fatalf("eventCount = %d", eventCount)
	}

	failureCount, err := logs.CountFailuresSince(ctx, time.Now().UTC().Add(-time.Hour))
	if err != nil {
		t.Fatalf("logs.CountFailuresSince() error = %v", err)
	}
	if failureCount < 1 {
		t.Fatalf("failureCount = %d", failureCount)
	}

	staleJob, err := jobs.Create(ctx, CreateWarmingJobParams{
		PhoneAID:    phone.ID,
		PhoneBID:    phoneB.ID,
		ScheduledAt: time.Now().UTC().Add(-time.Hour),
		Metadata:    map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("jobs.Create(staleJob) error = %v", err)
	}
	if err := jobs.UpdateStatus(ctx, staleJob.ID, "running", ""); err != nil {
		t.Fatalf("jobs.UpdateStatus(staleJob running) error = %v", err)
	}
	_, err = executor.Exec(ctx, `
update public.warming_jobs
set started_at = now() - interval '30 minutes',
    updated_at = now() - interval '30 minutes'
where id = $1
`, staleJob.ID)
	if err != nil {
		t.Fatalf("make stale job old error = %v", err)
	}

	staleCount, err := jobs.CountStaleRunning(ctx, time.Now().UTC().Add(-10*time.Minute))
	if err != nil {
		t.Fatalf("jobs.CountStaleRunning() error = %v", err)
	}
	if staleCount < 1 {
		t.Fatalf("staleCount = %d", staleCount)
	}

	affected, err := jobs.FailStaleRunning(ctx, time.Now().UTC().Add(-10*time.Minute), "integration stale cleanup")
	if err != nil {
		t.Fatalf("jobs.FailStaleRunning() error = %v", err)
	}
	if affected < 1 {
		t.Fatalf("affected = %d", affected)
	}

	_, err = executor.Exec(ctx, `
delete from public.execution_logs
where warming_job_id = $1
`, job.ID)
	if err != nil {
		t.Fatalf("delete execution logs error = %v", err)
	}

	deletedJobs, err := jobs.DeleteByTestRunID(ctx, testRunID)
	if err != nil {
		t.Fatalf("jobs.DeleteByTestRunID() error = %v", err)
	}
	if deletedJobs != 2 {
		t.Fatalf("deleted jobs = %d", deletedJobs)
	}

	_, err = executor.Exec(ctx, `
delete from public.evolution_events
where payload ->> 'testRunId' = $1
`, testRunID)
	if err != nil {
		t.Fatalf("delete evolution events error = %v", err)
	}

	deletedInstances, err := instances.DeleteByTestRunID(ctx, testRunID)
	if err != nil {
		t.Fatalf("instances.DeleteByTestRunID() error = %v", err)
	}
	if deletedInstances != 2 {
		t.Fatalf("deleted instances = %d", deletedInstances)
	}

	deletedPhones, err := phones.DeleteByTestRunID(ctx, testRunID)
	if err != nil {
		t.Fatalf("phones.DeleteByTestRunID() error = %v", err)
	}
	if deletedPhones != 2 {
		t.Fatalf("deleted phones = %d", deletedPhones)
	}

	deletedServers, err := servers.DeleteByTestRunID(ctx, testRunID)
	if err != nil {
		t.Fatalf("servers.DeleteByTestRunID() error = %v", err)
	}
	if deletedServers != 1 {
		t.Fatalf("deleted servers = %d", deletedServers)
	}

	deletedProxies, err := proxies.DeleteByTestRunID(ctx, testRunID)
	if err != nil {
		t.Fatalf("proxies.DeleteByTestRunID() error = %v", err)
	}
	if deletedProxies != 1 {
		t.Fatalf("deleted proxies = %d", deletedProxies)
	}
}

func stringPointer(value string) *string {
	return &value
}

func intPointer(value int) *int {
	return &value
}
