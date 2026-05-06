package repository

import (
	"context"
	"fmt"
	"time"
)

type ExecutionLog struct {
	ID                  string
	WarmingJobID        *string
	InstanceID          *string
	ActionType          *string
	Status              string
	RequestPayload      map[string]any
	ResponsePayload     map[string]any
	EvolutionMessageKey map[string]any
	RemoteJID           string
	Error               string
	DurationMs          *int
	CreatedAt           time.Time
}

type CreateExecutionLogParams struct {
	WarmingJobID        *string
	InstanceID          *string
	ActionType          *string
	Status              string
	RequestPayload      map[string]any
	ResponsePayload     map[string]any
	EvolutionMessageKey map[string]any
	RemoteJID           string
	Error               string
	DurationMs          *int
}

type ExecutionLogRepository struct {
	db Executor
}

func NewExecutionLogRepository(db Executor) ExecutionLogRepository {
	return ExecutionLogRepository{db: db}
}

func (r ExecutionLogRepository) Create(ctx context.Context, params CreateExecutionLogParams) (ExecutionLog, error) {
	requestPayload, err := encodeMetadata(params.RequestPayload)
	if err != nil {
		return ExecutionLog{}, err
	}
	responsePayload, err := encodeMetadata(params.ResponsePayload)
	if err != nil {
		return ExecutionLog{}, err
	}
	evolutionMessageKey, err := encodeMetadata(params.EvolutionMessageKey)
	if err != nil {
		return ExecutionLog{}, err
	}

	row := r.db.QueryRow(ctx, `
insert into public.execution_logs (
  warming_job_id, instance_id, action_type, status, request_payload, response_payload,
  evolution_message_key, remote_jid, error, duration_ms
)
values ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7::jsonb, $8, $9, $10)
returning id::text, warming_job_id::text, instance_id::text, action_type::text, status::text,
  coalesce(request_payload, '{}'::jsonb)::text::bytea,
  coalesce(response_payload, '{}'::jsonb)::text::bytea,
  coalesce(evolution_message_key, '{}'::jsonb)::text::bytea,
  coalesce(remote_jid, ''), coalesce(error, ''), duration_ms, created_at
`, params.WarmingJobID, params.InstanceID, params.ActionType, params.Status, requestPayload, responsePayload, evolutionMessageKey, params.RemoteJID, params.Error, params.DurationMs)

	log, err := scanExecutionLog(row)
	if err != nil {
		return ExecutionLog{}, fmt.Errorf("create execution log: %w", err)
	}
	return log, nil
}

func (r ExecutionLogRepository) List(ctx context.Context) ([]ExecutionLog, error) {
	rows, err := r.db.Query(ctx, `
select id::text, warming_job_id::text, instance_id::text, action_type::text, status::text,
  coalesce(request_payload, '{}'::jsonb)::text::bytea,
  coalesce(response_payload, '{}'::jsonb)::text::bytea,
  coalesce(evolution_message_key, '{}'::jsonb)::text::bytea,
  coalesce(remote_jid, ''), coalesce(error, ''), duration_ms, created_at
from public.execution_logs
order by created_at desc
`)
	if err != nil {
		return nil, fmt.Errorf("list execution logs: %w", err)
	}
	defer rows.Close()

	var logs []ExecutionLog
	for rows.Next() {
		log, err := scanExecutionLog(rows)
		if err != nil {
			return nil, fmt.Errorf("scan execution log: %w", err)
		}
		logs = append(logs, log)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate execution logs: %w", err)
	}
	return logs, nil
}

func (r ExecutionLogRepository) ExistsSuccessfulStep(ctx context.Context, warmingJobID string, stepID string) (bool, error) {
	row := r.db.QueryRow(ctx, `
select exists (
  select 1
  from public.execution_logs
  where warming_job_id = $1
    and request_payload ->> 'stepId' = $2
    and status = 'success'
)
`, warmingJobID, stepID)

	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, fmt.Errorf("check successful execution step: %w", err)
	}
	return exists, nil
}

func (r ExecutionLogRepository) UpdateStatusByMessageID(ctx context.Context, messageID string, remoteJID string, status string, responsePayload map[string]any, errorText string) error {
	responseJSON, err := encodeMetadata(responsePayload)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, `
update public.execution_logs
set status = $2::public.execution_status,
    remote_jid = case when $3 = '' then remote_jid else $3 end,
    response_payload = $4::jsonb,
    error = case when $5 = '' then error else $5 end
where id = (
  select id
  from public.execution_logs
  where evolution_message_key ->> 'id' = $1
     or request_payload ->> 'messageId' = $1
  order by created_at desc
  limit 1
)
`, messageID, status, remoteJID, responseJSON, errorText)
	if err != nil {
		return fmt.Errorf("update execution log by message id: %w", err)
	}
	return nil
}

func (r ExecutionLogRepository) FindPhoneNumberIDByMessageID(ctx context.Context, messageID string) (string, error) {
	row := r.db.QueryRow(ctx, `
select i.phone_number_id::text
from public.execution_logs el
join public.instances i on i.id = el.instance_id
where el.evolution_message_key ->> 'id' = $1
   or el.request_payload ->> 'messageId' = $1
order by el.created_at desc
limit 1
`, messageID)

	var phoneNumberID string
	if err := row.Scan(&phoneNumberID); err != nil {
		if isNoRowsError(err) {
			return "", nil
		}
		return "", fmt.Errorf("find phone number by message id: %w", err)
	}
	return phoneNumberID, nil
}

func (r ExecutionLogRepository) CountFailuresSince(ctx context.Context, since time.Time) (int, error) {
	row := r.db.QueryRow(ctx, `
select count(*)::int
from public.execution_logs
where status = 'failed'
  and created_at >= $1
`, since)

	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("count execution log failures since: %w", err)
	}
	return count, nil
}

func (r ExecutionLogRepository) BuildWarmingMetrics(ctx context.Context, phoneNumberID string) (PhoneWarmingMetrics, error) {
	row := r.db.QueryRow(ctx, `
select
  count(*) filter (where action_type = 'send_text' and status = 'success')::int,
  count(*) filter (where action_type = 'send_reply' and status = 'success')::int,
  count(*) filter (where action_type = 'send_reaction' and status = 'success')::int,
  count(*) filter (where status = 'failed')::int,
  count(*) filter (where response_payload ->> 'eventType' = 'CONNECTION_UPDATE' and response_payload -> 'data' ->> 'state' = 'close')::int,
  max(created_at)
from public.execution_logs
where instance_id in (
  select id
  from public.instances
  where phone_number_id = $1
)
`, phoneNumberID)

	var metrics PhoneWarmingMetrics
	if err := row.Scan(
		&metrics.SuccessTextCount,
		&metrics.SuccessReplyCount,
		&metrics.SuccessReactionCount,
		&metrics.FailureCount,
		&metrics.DisconnectedCount,
		&metrics.LastActivityAt,
	); err != nil {
		return PhoneWarmingMetrics{}, fmt.Errorf("build warming metrics: %w", err)
	}
	return metrics, nil
}

func scanExecutionLog(row Row) (ExecutionLog, error) {
	var log ExecutionLog
	var requestPayload []byte
	var responsePayload []byte
	var evolutionMessageKey []byte
	if err := row.Scan(
		&log.ID,
		&log.WarmingJobID,
		&log.InstanceID,
		&log.ActionType,
		&log.Status,
		&requestPayload,
		&responsePayload,
		&evolutionMessageKey,
		&log.RemoteJID,
		&log.Error,
		&log.DurationMs,
		&log.CreatedAt,
	); err != nil {
		return ExecutionLog{}, err
	}

	decodedRequest, err := decodeMetadata(requestPayload)
	if err != nil {
		return ExecutionLog{}, err
	}
	decodedResponse, err := decodeMetadata(responsePayload)
	if err != nil {
		return ExecutionLog{}, err
	}
	decodedKey, err := decodeMetadata(evolutionMessageKey)
	if err != nil {
		return ExecutionLog{}, err
	}
	log.RequestPayload = decodedRequest
	log.ResponsePayload = decodedResponse
	log.EvolutionMessageKey = decodedKey

	return log, nil
}
