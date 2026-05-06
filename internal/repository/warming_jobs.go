package repository

import (
	"context"
	"fmt"
	"time"
)

type WarmingJob struct {
	ID               string
	ScriptID         *string
	PhoneAID         string
	PhoneBID         string
	Status           string
	ScheduledAt      time.Time
	CurrentStepOrder int
	Error            string
	Metadata         map[string]any
}

type CreateWarmingJobParams struct {
	ScriptID    *string
	PhoneAID    string
	PhoneBID    string
	ScheduledAt time.Time
	Metadata    map[string]any
}

type WarmingJobRepository struct {
	db Executor
}

func NewWarmingJobRepository(db Executor) WarmingJobRepository {
	return WarmingJobRepository{db: db}
}

func (r WarmingJobRepository) Create(ctx context.Context, params CreateWarmingJobParams) (WarmingJob, error) {
	metadata, err := encodeMetadata(params.Metadata)
	if err != nil {
		return WarmingJob{}, err
	}

	row := r.db.QueryRow(ctx, `
insert into public.warming_jobs (script_id, phone_a_id, phone_b_id, scheduled_at, metadata)
values ($1, $2, $3, $4, $5::jsonb)
returning id::text, script_id::text, phone_a_id::text, phone_b_id::text, status::text, scheduled_at, current_step_order, coalesce(error, ''), metadata::text::bytea
`, params.ScriptID, params.PhoneAID, params.PhoneBID, params.ScheduledAt, metadata)

	job, err := scanWarmingJob(row)
	if err != nil {
		return WarmingJob{}, fmt.Errorf("create warming job: %w", err)
	}
	return job, nil
}

func (r WarmingJobRepository) List(ctx context.Context) ([]WarmingJob, error) {
	rows, err := r.db.Query(ctx, `
select id::text, script_id::text, phone_a_id::text, phone_b_id::text, status::text, scheduled_at, current_step_order, coalesce(error, ''), metadata::text::bytea
from public.warming_jobs
order by scheduled_at asc, created_at asc
`)
	if err != nil {
		return nil, fmt.Errorf("list warming jobs: %w", err)
	}
	defer rows.Close()

	var jobs []WarmingJob
	for rows.Next() {
		job, err := scanWarmingJob(rows)
		if err != nil {
			return nil, fmt.Errorf("scan warming job: %w", err)
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate warming jobs: %w", err)
	}
	return jobs, nil
}

func (r WarmingJobRepository) ListDuePending(ctx context.Context, now time.Time, limit int) ([]WarmingJob, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := r.db.Query(ctx, `
select id::text, script_id::text, phone_a_id::text, phone_b_id::text, status::text, scheduled_at, current_step_order, coalesce(error, ''), metadata::text::bytea
from public.warming_jobs
where status = 'pending'
  and scheduled_at <= $1
order by scheduled_at asc, created_at asc
limit $2
`, now, limit)
	if err != nil {
		return nil, fmt.Errorf("list due pending warming jobs: %w", err)
	}
	defer rows.Close()

	var jobs []WarmingJob
	for rows.Next() {
		job, err := scanWarmingJob(rows)
		if err != nil {
			return nil, fmt.Errorf("scan warming job: %w", err)
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate warming jobs: %w", err)
	}
	return jobs, nil
}

func (r WarmingJobRepository) GetByID(ctx context.Context, id string) (WarmingJob, error) {
	row := r.db.QueryRow(ctx, `
select id::text, script_id::text, phone_a_id::text, phone_b_id::text, status::text, scheduled_at, current_step_order, coalesce(error, ''), metadata::text::bytea
from public.warming_jobs
where id = $1
`, id)

	job, err := scanWarmingJob(row)
	if err != nil {
		return WarmingJob{}, fmt.Errorf("get warming job: %w", err)
	}
	return job, nil
}

func (r WarmingJobRepository) UpdateStatus(ctx context.Context, id string, status string, errorText string) error {
	_, err := r.db.Exec(ctx, `
update public.warming_jobs
set status = $2::public.execution_status,
    error = nullif($3, ''),
    started_at = case when $2 = 'running' and started_at is null then now() else started_at end,
    finished_at = case when $2 in ('success', 'failed', 'cancelled', 'skipped') then now() else finished_at end,
    updated_at = now()
where id = $1
`, id, status, errorText)
	if err != nil {
		return fmt.Errorf("update warming job status: %w", err)
	}
	return nil
}

func (r WarmingJobRepository) CountRunningByPair(ctx context.Context, phoneAID string, phoneBID string) (int, error) {
	row := r.db.QueryRow(ctx, `
select count(*)::int
from public.warming_jobs
where status = 'running'
  and (
    (phone_a_id = $1 and phone_b_id = $2)
    or
    (phone_a_id = $2 and phone_b_id = $1)
  )
`, phoneAID, phoneBID)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("count running warming jobs by pair: %w", err)
	}
	return count, nil
}

func (r WarmingJobRepository) CountRunningByEvolutionServer(ctx context.Context, evolutionServerID string) (int, error) {
	row := r.db.QueryRow(ctx, `
select count(distinct wj.id)::int
from public.warming_jobs wj
join public.instances ia on ia.phone_number_id = wj.phone_a_id
join public.instances ib on ib.phone_number_id = wj.phone_b_id
where wj.status = 'running'
  and (ia.evolution_server_id = $1 or ib.evolution_server_id = $1)
`, evolutionServerID)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("count running warming jobs by evolution server: %w", err)
	}
	return count, nil
}

func (r WarmingJobRepository) CountByStatus(ctx context.Context) (map[string]int, error) {
	rows, err := r.db.Query(ctx, `
select status::text, count(*)::int
from public.warming_jobs
group by status
`)
	if err != nil {
		return nil, fmt.Errorf("count warming jobs by status: %w", err)
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan warming job status count: %w", err)
		}
		counts[status] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate warming job status counts: %w", err)
	}
	return counts, nil
}

func (r WarmingJobRepository) CountStaleRunning(ctx context.Context, olderThan time.Time) (int, error) {
	row := r.db.QueryRow(ctx, `
select count(*)::int
from public.warming_jobs
where status = 'running'
  and coalesce(started_at, updated_at, created_at) < $1
`, olderThan)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("count stale running warming jobs: %w", err)
	}
	return count, nil
}

func (r WarmingJobRepository) FailStaleRunning(ctx context.Context, olderThan time.Time, reason string) (int64, error) {
	tag, err := r.db.Exec(ctx, `
update public.warming_jobs
set status = 'failed'::public.execution_status,
    error = nullif($2, ''),
    finished_at = now(),
    updated_at = now()
where status = 'running'
  and coalesce(started_at, updated_at, created_at) < $1
`, olderThan, reason)
	if err != nil {
		return 0, fmt.Errorf("fail stale running warming jobs: %w", err)
	}
	return tag.RowsAffected, nil
}

func (r WarmingJobRepository) ListRecentByPair(ctx context.Context, phoneAID, phoneBID string, since time.Time) ([]WarmingJob, error) {
	rows, err := r.db.Query(ctx, `
select id::text, script_id::text, phone_a_id::text, phone_b_id::text, status::text, scheduled_at, current_step_order, coalesce(error, ''), metadata::text::bytea
from public.warming_jobs
where scheduled_at >= $3
  and (
    (phone_a_id = $1 and phone_b_id = $2)
    or
    (phone_a_id = $2 and phone_b_id = $1)
  )
order by scheduled_at desc, created_at desc
`, phoneAID, phoneBID, since)
	if err != nil {
		return nil, fmt.Errorf("list recent warming jobs by pair: %w", err)
	}
	defer rows.Close()

	var jobs []WarmingJob
	for rows.Next() {
		job, err := scanWarmingJob(rows)
		if err != nil {
			return nil, fmt.Errorf("scan recent warming job: %w", err)
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recent warming jobs: %w", err)
	}
	return jobs, nil
}

func (r WarmingJobRepository) DeleteByTestRunID(ctx context.Context, testRunID string) (int64, error) {
	tag, err := r.db.Exec(ctx, `
delete from public.warming_jobs
where metadata ->> 'testRunId' = $1
`, testRunID)
	if err != nil {
		return 0, fmt.Errorf("delete warming jobs by testRunId: %w", err)
	}
	return tag.RowsAffected, nil
}

func scanWarmingJob(row Row) (WarmingJob, error) {
	var job WarmingJob
	var metadata []byte
	if err := row.Scan(
		&job.ID,
		&job.ScriptID,
		&job.PhoneAID,
		&job.PhoneBID,
		&job.Status,
		&job.ScheduledAt,
		&job.CurrentStepOrder,
		&job.Error,
		&metadata,
	); err != nil {
		return WarmingJob{}, err
	}

	decoded, err := decodeMetadata(metadata)
	if err != nil {
		return WarmingJob{}, err
	}
	job.Metadata = decoded

	return job, nil
}
