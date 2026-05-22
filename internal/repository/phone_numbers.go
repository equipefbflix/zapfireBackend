package repository

import (
	"context"
	"fmt"
	"time"
)

const (
	PhoneTypeHeater = "heater"
	PhoneTypeTarget = "target"
)

type PhoneNumber struct {
	ID               string
	PhoneE164        string
	Label            string
	Type             string
	Status           string
	WarmingScore     float64
	ConnectionStatus string
	Metadata         map[string]any
}

type CreatePhoneNumberParams struct {
	PhoneE164 string
	Label     string
	Type      *string
	Metadata  map[string]any
}

type UpdatePhoneNumberParams struct {
	Label    *string
	Status   *string
	Metadata map[string]any
}

type PhoneWarmingMetrics struct {
	SuccessTextCount     int
	SuccessReplyCount    int
	SuccessReactionCount int
	FailureCount         int
	DisconnectedCount    int
	LastActivityAt       *time.Time
}

type PhoneNumberRepository struct {
	db Executor
}

func NewPhoneNumberRepository(db Executor) PhoneNumberRepository {
	return PhoneNumberRepository{db: db}
}

func (r PhoneNumberRepository) Create(ctx context.Context, params CreatePhoneNumberParams) (PhoneNumber, error) {
	metadata, err := encodeMetadata(params.Metadata)
	if err != nil {
		return PhoneNumber{}, err
	}

	phoneType := PhoneTypeTarget
	if params.Type != nil {
		phoneType = *params.Type
	}

	row := r.db.QueryRow(ctx, `
insert into public.phone_numbers (phone_e164, label, type, metadata)
values ($1, $2, $3::public.phone_type, $4::jsonb)
returning id::text, phone_e164, coalesce(label, ''), type::text, status::text, warming_score::float8, coalesce(null::text, ''), metadata::text::bytea
`, params.PhoneE164, params.Label, phoneType, metadata)

	phone, err := scanPhoneNumber(row)
	if err != nil {
		return PhoneNumber{}, fmt.Errorf("create phone number: %w", err)
	}
	return phone, nil
}

func (r PhoneNumberRepository) GetByID(ctx context.Context, id string) (PhoneNumber, error) {
	row := r.db.QueryRow(ctx, `
select pn.id::text, pn.phone_e164, coalesce(pn.label, ''), pn.type::text, pn.status::text, pn.warming_score::float8, coalesce(i.status::text, '')::text, pn.metadata::text::bytea
from public.phone_numbers pn
left join lateral (
  select i.status
  from public.instances i
  where i.phone_number_id = pn.id
  order by i.created_at desc
  limit 1
) i on true
where pn.id = $1
`, id)

	phone, err := scanPhoneNumber(row)
	if err != nil {
		return PhoneNumber{}, fmt.Errorf("get phone number: %w", err)
	}
	return phone, nil
}

func (r PhoneNumberRepository) FindByE164(ctx context.Context, phoneE164 string) (*PhoneNumber, error) {
	row := r.db.QueryRow(ctx, `
select pn.id::text, pn.phone_e164, coalesce(pn.label, ''), pn.type::text, pn.status::text, pn.warming_score::float8, coalesce(i.status::text, '')::text, pn.metadata::text::bytea
from public.phone_numbers pn
left join lateral (
  select i.status
  from public.instances i
  where i.phone_number_id = pn.id
  order by i.created_at desc
  limit 1
) i on true
where pn.phone_e164 = $1
limit 1
`, phoneE164)

	phone, err := scanPhoneNumber(row)
	if err != nil {
		if isNoRowsError(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("find phone number by e164: %w", err)
	}
	return &phone, nil
}

func (r PhoneNumberRepository) List(ctx context.Context) ([]PhoneNumber, error) {
	rows, err := r.db.Query(ctx, `
select pn.id::text, pn.phone_e164, coalesce(pn.label, ''), pn.type::text, pn.status::text, pn.warming_score::float8, coalesce(i.status::text, '')::text, pn.metadata::text::bytea
from public.phone_numbers pn
left join lateral (
  select i.status
  from public.instances i
  where i.phone_number_id = pn.id
  order by i.created_at desc
  limit 1
) i on true
order by pn.created_at desc
`)
	if err != nil {
		return nil, fmt.Errorf("list phone numbers: %w", err)
	}
	defer rows.Close()

	var phones []PhoneNumber
	for rows.Next() {
		phone, err := scanPhoneNumber(rows)
		if err != nil {
			return nil, fmt.Errorf("scan phone number: %w", err)
		}
		phones = append(phones, phone)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate phone numbers: %w", err)
	}
	return phones, nil
}

func (r PhoneNumberRepository) ListByType(ctx context.Context, phoneType string) ([]PhoneNumber, error) {
	rows, err := r.db.Query(ctx, `
select pn.id::text, pn.phone_e164, coalesce(pn.label, ''), pn.type::text, pn.status::text, pn.warming_score::float8, coalesce(i.status::text, '')::text, pn.metadata::text::bytea
from public.phone_numbers pn
left join lateral (
  select i.status
  from public.instances i
  where i.phone_number_id = pn.id
  order by i.created_at desc
  limit 1
) i on true
where pn.type = $1::public.phone_type
order by pn.created_at desc
`, phoneType)
	if err != nil {
		return nil, fmt.Errorf("list phone numbers by type: %w", err)
	}
	defer rows.Close()

	var phones []PhoneNumber
	for rows.Next() {
		phone, err := scanPhoneNumber(rows)
		if err != nil {
			return nil, fmt.Errorf("scan phone number: %w", err)
		}
		phones = append(phones, phone)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate phone numbers: %w", err)
	}
	return phones, nil
}

func (r PhoneNumberRepository) DeleteByTestRunID(ctx context.Context, testRunID string) (int64, error) {
	tag, err := r.db.Exec(ctx, `
delete from public.phone_numbers
where metadata ->> 'testRunId' = $1
`, testRunID)
	if err != nil {
		return 0, fmt.Errorf("delete phone numbers by testRunId: %w", err)
	}
	return tag.RowsAffected, nil
}

func (r PhoneNumberRepository) UpdateWarmingState(ctx context.Context, id string, score float64, status string) error {
	_, err := r.db.Exec(ctx, `
update public.phone_numbers
set warming_score = $2::numeric,
    status = $3::public.phone_status,
    updated_at = now(),
    last_activity_at = case when $2::numeric > 0 then now() else last_activity_at end
where id = $1
`, id, score, status)
	if err != nil {
		return fmt.Errorf("update phone warming state: %w", err)
	}
	return nil
}

func (r PhoneNumberRepository) Update(ctx context.Context, id string, params UpdatePhoneNumberParams) (PhoneNumber, error) {
	metadata, err := encodeMetadata(params.Metadata)
	if err != nil {
		return PhoneNumber{}, err
	}

	row := r.db.QueryRow(ctx, `
update public.phone_numbers
set label = coalesce($2, label),
    status = coalesce($3::public.phone_status, status),
    metadata = coalesce($4::jsonb, metadata),
    updated_at = now()
where id = $1
returning id::text, phone_e164, coalesce(label, ''), type::text, status::text, warming_score::float8, coalesce((select status::text from public.instances where phone_number_id = $1 order by created_at desc limit 1), ''), metadata::text::bytea
`, id, params.Label, params.Status, metadata)

	phone, err := scanPhoneNumber(row)
	if err != nil {
		return PhoneNumber{}, fmt.Errorf("update phone number: %w", err)
	}
	return phone, nil
}

func (r PhoneNumberRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `
delete from public.phone_numbers
where id = $1
`, id)
	if err != nil {
		return fmt.Errorf("delete phone number: %w", err)
	}
	return nil
}

func (r PhoneNumberRepository) GetDailyMessageCount(ctx context.Context, phoneNumberID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
select coalesce(daily_message_count, 0)
from public.phone_numbers
where id = $1
`, phoneNumberID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get daily message count: %w", err)
	}
	return count, nil
}

func (r PhoneNumberRepository) IncrementDailyMessageCount(ctx context.Context, phoneNumberID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
update public.phone_numbers
set daily_message_count = coalesce(daily_message_count, 0) + 1,
    updated_at = now()
where id = $1
returning coalesce(daily_message_count, 0)
`, phoneNumberID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("increment daily message count: %w", err)
	}
	return count, nil
}

func (r PhoneNumberRepository) ResetDailyMessageCounts(ctx context.Context) (int64, error) {
	tag, err := r.db.Exec(ctx, `
update public.phone_numbers
set daily_message_count = 0,
    updated_at = now()
where daily_message_count > 0
`)
	if err != nil {
		return 0, fmt.Errorf("reset daily message counts: %w", err)
	}
	return tag.RowsAffected, nil
}

func scanPhoneNumber(row Row) (PhoneNumber, error) {
	var phone PhoneNumber
	var metadata []byte
	if err := row.Scan(
		&phone.ID,
		&phone.PhoneE164,
		&phone.Label,
		&phone.Type,
		&phone.Status,
		&phone.WarmingScore,
		&phone.ConnectionStatus,
		&metadata,
	); err != nil {
		return PhoneNumber{}, err
	}

	decoded, err := decodeMetadata(metadata)
	if err != nil {
		return PhoneNumber{}, err
	}
	phone.Metadata = decoded

	return phone, nil
}
