package repository

import (
	"context"
	"fmt"
	"time"
)

type EvolutionEvent struct {
	ID                string
	EvolutionServerID *string
	InstanceName      string
	EventType         string
	Payload           map[string]any
	ReceivedAt        time.Time
	ProcessedAt       *time.Time
	ProcessingError   string
}

type CreateEvolutionEventParams struct {
	EvolutionServerID *string
	InstanceName      string
	EventType         string
	Payload           map[string]any
}

type EvolutionEventRepository struct {
	db Executor
}

func NewEvolutionEventRepository(db Executor) EvolutionEventRepository {
	return EvolutionEventRepository{db: db}
}

func (r EvolutionEventRepository) Create(ctx context.Context, params CreateEvolutionEventParams) (EvolutionEvent, error) {
	payload, err := encodeMetadata(params.Payload)
	if err != nil {
		return EvolutionEvent{}, err
	}

	row := r.db.QueryRow(ctx, `
insert into public.evolution_events (evolution_server_id, instance_name, event_type, payload)
values ($1, $2, $3, $4::jsonb)
returning id::text, evolution_server_id::text, coalesce(instance_name, ''), event_type, payload::text::bytea, received_at
`, params.EvolutionServerID, params.InstanceName, params.EventType, payload)

	event, err := scanEvolutionEvent(row)
	if err != nil {
		return EvolutionEvent{}, fmt.Errorf("create evolution event: %w", err)
	}
	return event, nil
}

func (r EvolutionEventRepository) MarkProcessed(ctx context.Context, eventID string, processingError string) error {
	_, err := r.db.Exec(ctx, `
update public.evolution_events
set processed_at = now(),
    processing_error = nullif($2, '')
where id = $1
`, eventID, processingError)
	if err != nil {
		return fmt.Errorf("mark evolution event processed: %w", err)
	}
	return nil
}

func (r EvolutionEventRepository) CountSince(ctx context.Context, since time.Time) (int, error) {
	row := r.db.QueryRow(ctx, `
select count(*)::int
from public.evolution_events
where received_at >= $1
`, since)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("count evolution events since: %w", err)
	}
	return count, nil
}

func scanEvolutionEvent(row Row) (EvolutionEvent, error) {
	var event EvolutionEvent
	var payload []byte
	if err := row.Scan(
		&event.ID,
		&event.EvolutionServerID,
		&event.InstanceName,
		&event.EventType,
		&payload,
		&event.ReceivedAt,
	); err != nil {
		return EvolutionEvent{}, err
	}

	decoded, err := decodeMetadata(payload)
	if err != nil {
		return EvolutionEvent{}, err
	}
	event.Payload = decoded

	return event, nil
}
