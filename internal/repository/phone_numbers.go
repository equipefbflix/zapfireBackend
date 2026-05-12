package repository

import (
	"context"
	"fmt"
	"time"
)

type PhoneNumber struct {
	ID           string
	PhoneE164    string
	Label        string
	Status       string
	WarmingScore float64
	Metadata     map[string]any
}

type CreatePhoneNumberParams struct {
	PhoneE164 string
	Label     string
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

	row := r.db.QueryRow(ctx, `
insert into public.phone_numbers (phone_e164, label, metadata)
values ($1, $2, $3::jsonb)
returning id::text, phone_e164, coalesce(label, ''), status::text, warming_score::float8, metadata::text::bytea
`, params.PhoneE164, params.Label, metadata)

	phone, err := scanPhoneNumber(row)
	if err != nil {
		return PhoneNumber{}, fmt.Errorf("create phone number: %w", err)
	}
	return phone, nil
}

func (r PhoneNumberRepository) GetByID(ctx context.Context, id string) (PhoneNumber, error) {
	row := r.db.QueryRow(ctx, `
select id::text, phone_e164, coalesce(label, ''), status::text, warming_score::float8, metadata::text::bytea
from public.phone_numbers
where id = $1
`, id)

	phone, err := scanPhoneNumber(row)
	if err != nil {
		return PhoneNumber{}, fmt.Errorf("get phone number: %w", err)
	}
	return phone, nil
}

func (r PhoneNumberRepository) FindByE164(ctx context.Context, phoneE164 string) (*PhoneNumber, error) {
	row := r.db.QueryRow(ctx, `
select id::text, phone_e164, coalesce(label, ''), status::text, warming_score::float8, metadata::text::bytea
from public.phone_numbers
where phone_e164 = $1
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
select id::text, phone_e164, coalesce(label, ''), status::text, warming_score::float8, metadata::text::bytea
from public.phone_numbers
order by created_at desc
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
returning id::text, phone_e164, coalesce(label, ''), status::text, warming_score::float8, metadata::text::bytea
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

func scanPhoneNumber(row Row) (PhoneNumber, error) {
	var phone PhoneNumber
	var metadata []byte
	if err := row.Scan(
		&phone.ID,
		&phone.PhoneE164,
		&phone.Label,
		&phone.Status,
		&phone.WarmingScore,
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
