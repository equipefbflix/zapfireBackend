package repository

import (
	"context"
	"fmt"
)

type MessageTemplate struct {
	ID              string
	Category        string
	Title           string
	Body            string
	Weight          int
	Enabled         bool
	MinWarmingScore float64
	MaxWarmingScore float64
	Metadata        map[string]any
}

type CreateMessageTemplateParams struct {
	Category        string
	Title           string
	Body            string
	Weight          int
	Enabled         bool
	MinWarmingScore float64
	MaxWarmingScore float64
	Metadata        map[string]any
}

type UpdateMessageTemplateParams struct {
	Category        *string
	Title           *string
	Body            *string
	Weight          *int
	Enabled         *bool
	MinWarmingScore *float64
	MaxWarmingScore *float64
	Metadata        map[string]any
}

type MessageTemplateRepository struct {
	db Executor
}

func NewMessageTemplateRepository(db Executor) MessageTemplateRepository {
	return MessageTemplateRepository{db: db}
}

func (r MessageTemplateRepository) Create(ctx context.Context, params CreateMessageTemplateParams) (MessageTemplate, error) {
	metadata, err := encodeMetadata(params.Metadata)
	if err != nil {
		return MessageTemplate{}, err
	}

	row := r.db.QueryRow(ctx, `
insert into public.message_templates (category, title, body, weight, enabled, min_warming_score, max_warming_score, metadata)
values ($1, $2, $3, $4, $5, $6, $7, $8::jsonb)
returning id::text, category, title, body, weight, enabled, min_warming_score::float8, max_warming_score::float8, metadata::text::bytea
`, params.Category, params.Title, params.Body, params.Weight, params.Enabled, params.MinWarmingScore, params.MaxWarmingScore, metadata)

	template, err := scanMessageTemplate(row)
	if err != nil {
		return MessageTemplate{}, fmt.Errorf("create message template: %w", err)
	}
	return template, nil
}

func (r MessageTemplateRepository) List(ctx context.Context) ([]MessageTemplate, error) {
	rows, err := r.db.Query(ctx, `
select id::text, category, title, body, weight, enabled, min_warming_score::float8, max_warming_score::float8, metadata::text::bytea
from public.message_templates
order by category asc, title asc
`)
	if err != nil {
		return nil, fmt.Errorf("list message templates: %w", err)
	}
	defer rows.Close()

	var templates []MessageTemplate
	for rows.Next() {
		template, err := scanMessageTemplate(rows)
		if err != nil {
			return nil, fmt.Errorf("scan message template: %w", err)
		}
		templates = append(templates, template)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate message templates: %w", err)
	}
	return templates, nil
}

func (r MessageTemplateRepository) Update(ctx context.Context, id string, params UpdateMessageTemplateParams) (MessageTemplate, error) {
	metadata, err := encodeMetadata(params.Metadata)
	if err != nil {
		return MessageTemplate{}, err
	}

	row := r.db.QueryRow(ctx, `
update public.message_templates
set category = coalesce($2, category),
    title = coalesce($3, title),
    body = coalesce($4, body),
    weight = coalesce($5, weight),
    enabled = coalesce($6, enabled),
    min_warming_score = coalesce($7, min_warming_score),
    max_warming_score = coalesce($8, max_warming_score),
    metadata = case when $9::jsonb = '{}'::jsonb then metadata else $9::jsonb end,
    updated_at = now()
where id = $1::uuid
returning id::text, category, title, body, weight, enabled, min_warming_score::float8, max_warming_score::float8, metadata::text::bytea
`, id, params.Category, params.Title, params.Body, params.Weight, params.Enabled, params.MinWarmingScore, params.MaxWarmingScore, metadata)

	template, err := scanMessageTemplate(row)
	if err != nil {
		return MessageTemplate{}, fmt.Errorf("update message template: %w", err)
	}
	return template, nil
}

func (r MessageTemplateRepository) DeleteByTestRunID(ctx context.Context, testRunID string) (int64, error) {
	tag, err := r.db.Exec(ctx, `
delete from public.message_templates
where metadata ->> 'testRunId' = $1
`, testRunID)
	if err != nil {
		return 0, fmt.Errorf("delete message templates by testRunId: %w", err)
	}
	return tag.RowsAffected, nil
}

func scanMessageTemplate(row Row) (MessageTemplate, error) {
	var template MessageTemplate
	var metadata []byte
	if err := row.Scan(
		&template.ID,
		&template.Category,
		&template.Title,
		&template.Body,
		&template.Weight,
		&template.Enabled,
		&template.MinWarmingScore,
		&template.MaxWarmingScore,
		&metadata,
	); err != nil {
		return MessageTemplate{}, err
	}

	decoded, err := decodeMetadata(metadata)
	if err != nil {
		return MessageTemplate{}, err
	}
	template.Metadata = decoded

	return template, nil
}
