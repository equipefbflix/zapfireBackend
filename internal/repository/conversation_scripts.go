package repository

import (
	"context"
	"fmt"
)

type ConversationScript struct {
	ID              string
	Name            string
	Category        string
	Enabled         bool
	Weight          int
	MinWarmingScore float64
	MaxWarmingScore float64
}

type CreateConversationScriptParams struct {
	Name            string
	Category        string
	Enabled         bool
	Weight          int
	MinWarmingScore float64
	MaxWarmingScore float64
}

type ConversationStep struct {
	ID              string
	ScriptID        string
	StepOrder       int
	SenderRole      string
	ActionType      string
	TemplateID      *string
	Payload         map[string]any
	MinDelaySeconds int
	MaxDelaySeconds int
}

type CreateConversationStepParams struct {
	ScriptID        string
	StepOrder       int
	SenderRole      string
	ActionType      string
	TemplateID      *string
	Payload         map[string]any
	MinDelaySeconds int
	MaxDelaySeconds int
}

type ConversationScriptRepository struct {
	db Executor
}

func NewConversationScriptRepository(db Executor) ConversationScriptRepository {
	return ConversationScriptRepository{db: db}
}

func (r ConversationScriptRepository) Create(ctx context.Context, params CreateConversationScriptParams) (ConversationScript, error) {
	row := r.db.QueryRow(ctx, `
insert into public.conversation_scripts (name, category, enabled, weight, min_warming_score, max_warming_score)
values ($1, $2, $3, $4, $5, $6)
returning id::text, name, category, enabled, weight, min_warming_score::float8, max_warming_score::float8
`, params.Name, params.Category, params.Enabled, params.Weight, params.MinWarmingScore, params.MaxWarmingScore)

	script, err := scanConversationScript(row)
	if err != nil {
		return ConversationScript{}, fmt.Errorf("create conversation script: %w", err)
	}
	return script, nil
}

func (r ConversationScriptRepository) List(ctx context.Context) ([]ConversationScript, error) {
	rows, err := r.db.Query(ctx, `
select id::text, name, category, enabled, weight, min_warming_score::float8, max_warming_score::float8
from public.conversation_scripts
order by category asc, name asc
`)
	if err != nil {
		return nil, fmt.Errorf("list conversation scripts: %w", err)
	}
	defer rows.Close()

	var scripts []ConversationScript
	for rows.Next() {
		script, err := scanConversationScript(rows)
		if err != nil {
			return nil, fmt.Errorf("scan conversation script: %w", err)
		}
		scripts = append(scripts, script)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate conversation scripts: %w", err)
	}
	return scripts, nil
}

func (r ConversationScriptRepository) DeleteByName(ctx context.Context, name string) (int64, error) {
	tag, err := r.db.Exec(ctx, `
delete from public.conversation_scripts
where name = $1
`, name)
	if err != nil {
		return 0, fmt.Errorf("delete conversation script by name: %w", err)
	}
	return tag.RowsAffected, nil
}

type ConversationStepRepository struct {
	db Executor
}

func NewConversationStepRepository(db Executor) ConversationStepRepository {
	return ConversationStepRepository{db: db}
}

func (r ConversationStepRepository) Create(ctx context.Context, params CreateConversationStepParams) (ConversationStep, error) {
	payload, err := encodeMetadata(params.Payload)
	if err != nil {
		return ConversationStep{}, err
	}

	row := r.db.QueryRow(ctx, `
insert into public.conversation_steps (script_id, step_order, sender_role, action_type, template_id, payload, min_delay_seconds, max_delay_seconds)
values ($1, $2, $3, $4, $5, $6::jsonb, $7, $8)
returning id::text, script_id::text, step_order, sender_role, action_type::text, template_id::text, payload::text::bytea, min_delay_seconds, max_delay_seconds
`, params.ScriptID, params.StepOrder, params.SenderRole, params.ActionType, params.TemplateID, payload, params.MinDelaySeconds, params.MaxDelaySeconds)

	step, err := scanConversationStep(row)
	if err != nil {
		return ConversationStep{}, fmt.Errorf("create conversation step: %w", err)
	}
	return step, nil
}

func (r ConversationStepRepository) ListByScriptID(ctx context.Context, scriptID string) ([]ConversationStep, error) {
	rows, err := r.db.Query(ctx, `
select id::text, script_id::text, step_order, sender_role, action_type::text, template_id::text, payload::text::bytea, min_delay_seconds, max_delay_seconds
from public.conversation_steps
where script_id = $1
order by step_order asc
`, scriptID)
	if err != nil {
		return nil, fmt.Errorf("list conversation steps: %w", err)
	}
	defer rows.Close()

	var steps []ConversationStep
	for rows.Next() {
		step, err := scanConversationStep(rows)
		if err != nil {
			return nil, fmt.Errorf("scan conversation step: %w", err)
		}
		steps = append(steps, step)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate conversation steps: %w", err)
	}
	return steps, nil
}

func scanConversationScript(row Row) (ConversationScript, error) {
	var script ConversationScript
	if err := row.Scan(
		&script.ID,
		&script.Name,
		&script.Category,
		&script.Enabled,
		&script.Weight,
		&script.MinWarmingScore,
		&script.MaxWarmingScore,
	); err != nil {
		return ConversationScript{}, err
	}
	return script, nil
}

func scanConversationStep(row Row) (ConversationStep, error) {
	var step ConversationStep
	var payload []byte
	if err := row.Scan(
		&step.ID,
		&step.ScriptID,
		&step.StepOrder,
		&step.SenderRole,
		&step.ActionType,
		&step.TemplateID,
		&payload,
		&step.MinDelaySeconds,
		&step.MaxDelaySeconds,
	); err != nil {
		return ConversationStep{}, err
	}

	decoded, err := decodeMetadata(payload)
	if err != nil {
		return ConversationStep{}, err
	}
	step.Payload = decoded

	return step, nil
}
