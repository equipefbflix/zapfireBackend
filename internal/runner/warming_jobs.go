package runner

import (
	"context"
	"fmt"
	"log/slog"

	"aquecedor-evolution/backend/internal/evolution"
	"aquecedor-evolution/backend/internal/executor"
	"aquecedor-evolution/backend/internal/repository"
)

type JobStore interface {
	GetByID(ctx context.Context, id string) (repository.WarmingJob, error)
	UpdateStatus(ctx context.Context, id string, status string, errorText string) error
}

type StepStore interface {
	ListByScriptID(ctx context.Context, scriptID string) ([]repository.ConversationStep, error)
}

type InstanceStore interface {
	GetOpenByPhoneNumberID(ctx context.Context, phoneNumberID string) (repository.Instance, error)
}

type ExecutionLogStore interface {
	Create(ctx context.Context, params repository.CreateExecutionLogParams) (repository.ExecutionLog, error)
	ExistsSuccessfulStep(ctx context.Context, warmingJobID string, stepID string) (bool, error)
}

type WarmingJobRunner struct {
	jobs          JobStore
	steps         StepStore
	instances     InstanceStore
	executors     InstanceStepExecutorFactory
	executionLogs ExecutionLogStore
	concurrency   ConcurrencyGate
}

type ConcurrencyGate interface {
	Check(ctx context.Context, serverID string, phoneAID string, phoneBID string) error
}

func NewWarmingJobRunner(jobs JobStore, steps StepStore, instances InstanceStore, executors InstanceStepExecutorFactory, executionLogs ExecutionLogStore, concurrency ConcurrencyGate) WarmingJobRunner {
	return WarmingJobRunner{
		jobs:          jobs,
		steps:         steps,
		instances:     instances,
		executors:     executors,
		executionLogs: executionLogs,
		concurrency:   concurrency,
	}
}

func (r WarmingJobRunner) Run(ctx context.Context, jobID string) (int, error) {
	slog.Info("warming job run started", "jobId", jobID)
	job, err := r.jobs.GetByID(ctx, jobID)
	if err != nil {
		slog.Error("warming job load failed", "jobId", jobID, "error", err)
		return 0, err
	}
	if job.ScriptID == nil || *job.ScriptID == "" {
		_ = r.jobs.UpdateStatus(ctx, job.ID, "failed", "warming job has no script")
		slog.Error("warming job missing script", "jobId", job.ID)
		return 0, fmt.Errorf("warming job %q has no script", job.ID)
	}

	steps, err := r.steps.ListByScriptID(ctx, *job.ScriptID)
	if err != nil {
		_ = r.jobs.UpdateStatus(ctx, job.ID, "failed", err.Error())
		slog.Error("warming job load steps failed", "jobId", job.ID, "error", err)
		return 0, err
	}

	instanceA, err := r.instances.GetOpenByPhoneNumberID(ctx, job.PhoneAID)
	if err != nil {
		_ = r.jobs.UpdateStatus(ctx, job.ID, "failed", err.Error())
		slog.Error("warming job load instance A failed", "jobId", job.ID, "error", err)
		return 0, err
	}
	instanceB, err := r.instances.GetOpenByPhoneNumberID(ctx, job.PhoneBID)
	if err != nil {
		_ = r.jobs.UpdateStatus(ctx, job.ID, "failed", err.Error())
		slog.Error("warming job load instance B failed", "jobId", job.ID, "error", err)
		return 0, err
	}
	if r.concurrency != nil {
		serverID := instanceA.EvolutionServerID
		if serverID == "" {
			serverID = instanceB.EvolutionServerID
		}
		if err := r.concurrency.Check(ctx, serverID, job.PhoneAID, job.PhoneBID); err != nil {
			_ = r.jobs.UpdateStatus(ctx, job.ID, "failed", err.Error())
			slog.Warn("warming job concurrency blocked", "jobId", job.ID, "error", err)
			return 0, err
		}
	}
	_ = r.jobs.UpdateStatus(ctx, job.ID, "running", "")
	executorA, err := r.executors.ForInstance(ctx, instanceA)
	if err != nil {
		_ = r.jobs.UpdateStatus(ctx, job.ID, "failed", err.Error())
		slog.Error("warming job executor A failed", "jobId", job.ID, "error", err)
		return 0, err
	}
	executorB, err := r.executors.ForInstance(ctx, instanceB)
	if err != nil {
		_ = r.jobs.UpdateStatus(ctx, job.ID, "failed", err.Error())
		slog.Error("warming job executor B failed", "jobId", job.ID, "error", err)
		return 0, err
	}

	executed := 0
	for index, step := range steps {
		exists, err := r.executionLogs.ExistsSuccessfulStep(ctx, job.ID, step.ID)
		if err != nil {
			return index, err
		}
		if exists {
			slog.Info("warming job step skipped", "jobId", job.ID, "stepId", step.ID, "stepOrder", step.StepOrder)
			continue
		}

		instance := instanceForStep(step, instanceA, instanceB)
		stepExecutor := executorForStep(step, executorA, executorB)
		resolvedStep := resolveConversationStep(step, job.Metadata)
		slog.Info("warming job step started", "jobId", job.ID, "stepId", step.ID, "stepOrder", step.StepOrder, "actionType", step.ActionType, "instanceName", instance.InstanceName)
		result, err := stepExecutor.Execute(ctx, instance.InstanceName, resolvedStep)
		if err != nil {
			_, _ = r.executionLogs.Create(ctx, r.logParams(job, instance, resolvedStep, "failed", nil, nil, err.Error()))
			_ = r.jobs.UpdateStatus(ctx, job.ID, "failed", err.Error())
			slog.Error("warming job step failed", "jobId", job.ID, "stepId", step.ID, "stepOrder", step.StepOrder, "actionType", step.ActionType, "error", err)
			return index, err
		}
		if _, err := r.executionLogs.Create(ctx, r.logParams(job, instance, resolvedStep, "success", result.MessageKey, result.ResponsePayload, "")); err != nil {
			slog.Error("warming job log create failed", "jobId", job.ID, "stepId", step.ID, "error", err)
			return index, err
		}
		slog.Info("warming job step finished",
			"jobId", job.ID,
			"stepId", step.ID,
			"stepOrder", step.StepOrder,
			"actionType", step.ActionType,
			"acceptedAsync", result.AcceptedAsync,
			"messageId", messageID(result.MessageKey),
		)
		executed++
	}
	_ = r.jobs.UpdateStatus(ctx, job.ID, "success", "")
	slog.Info("warming job run finished", "jobId", job.ID, "executedSteps", executed)

	return executed, nil
}

func (r WarmingJobRunner) logParams(job repository.WarmingJob, instance repository.Instance, step repository.ConversationStep, status string, messageKey any, extraResponse map[string]any, errorText string) repository.CreateExecutionLogParams {
	requestPayload := map[string]any{
		"stepId":     step.ID,
		"stepOrder":  step.StepOrder,
		"senderRole": step.SenderRole,
		"payload":    step.Payload,
	}
	responsePayload := map[string]any{}
	for key, value := range extraResponse {
		responsePayload[key] = value
	}
	evolutionMessageKey := map[string]any{}
	if messageKey != nil {
		responsePayload["messageKey"] = messageKey
		evolutionMessageKey = messageKeyPayload(messageKey)
	}
	actionType := step.ActionType
	return repository.CreateExecutionLogParams{
		WarmingJobID:        &job.ID,
		InstanceID:          &instance.ID,
		ActionType:          &actionType,
		Status:              status,
		RequestPayload:      requestPayload,
		ResponsePayload:     responsePayload,
		EvolutionMessageKey: evolutionMessageKey,
		Error:               errorText,
	}
}

func messageKeyPayload(messageKey any) map[string]any {
	switch typed := messageKey.(type) {
	case *evolution.MessageKey:
		if typed == nil {
			return map[string]any{}
		}
		return map[string]any{
			"id":        typed.ID,
			"remoteJid": typed.RemoteJID,
			"fromMe":    typed.FromMe,
		}
	case evolution.MessageKey:
		return map[string]any{
			"id":        typed.ID,
			"remoteJid": typed.RemoteJID,
			"fromMe":    typed.FromMe,
		}
	case map[string]any:
		return typed
	default:
		return map[string]any{}
	}
}

func messageID(messageKey *evolution.MessageKey) string {
	if messageKey == nil {
		return ""
	}
	return messageKey.ID
}

func resolveConversationStep(step repository.ConversationStep, metadata map[string]any) repository.ConversationStep {
	if len(step.Payload) == 0 || len(metadata) == 0 {
		return step
	}

	resolved := step
	resolved.Payload = make(map[string]any, len(step.Payload))
	for key, value := range step.Payload {
		resolved.Payload[key] = resolvePayloadValue(value, metadata)
	}
	return resolved
}

func resolvePayloadValue(value any, metadata map[string]any) any {
	switch typed := value.(type) {
	case string:
		return resolvePlaceholderString(typed, metadata)
	case []any:
		items := make([]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, resolvePayloadValue(item, metadata))
		}
		return items
	case map[string]any:
		resolved := make(map[string]any, len(typed))
		for key, item := range typed {
			resolved[key] = resolvePayloadValue(item, metadata)
		}
		return resolved
	default:
		return value
	}
}

func resolvePlaceholderString(value string, metadata map[string]any) string {
	switch value {
	case "{{phoneA}}":
		return metadataString(metadata, "phoneAE164")
	case "{{phoneB}}":
		return metadataString(metadata, "phoneBE164")
	case "{{triggerMessageId}}":
		return metadataString(metadata, "triggerMessageId")
	case "{{triggerRemoteJid}}":
		return metadataString(metadata, "triggerRemoteJid")
	default:
		return value
	}
}

func metadataString(metadata map[string]any, key string) string {
	value, _ := metadata[key].(string)
	return value
}

func instanceForStep(step repository.ConversationStep, instanceA repository.Instance, instanceB repository.Instance) repository.Instance {
	if step.SenderRole == "b" {
		return instanceB
	}
	return instanceA
}

func executorForStep(step repository.ConversationStep, executorA *executor.StepExecutor, executorB *executor.StepExecutor) *executor.StepExecutor {
	if step.SenderRole == "b" {
		return executorB
	}
	return executorA
}
