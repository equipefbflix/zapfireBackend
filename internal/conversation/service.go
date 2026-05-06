package conversation

import (
	"context"

	"aquecedor-evolution/backend/internal/repository"
)

type CreateScriptParams struct {
	Name            string
	Category        string
	Enabled         bool
	Weight          int
	MinWarmingScore float64
	MaxWarmingScore float64
	Steps           []CreateStepParams
}

type CreateStepParams struct {
	StepOrder       int
	SenderRole      string
	ActionType      string
	TemplateID      *string
	Payload         map[string]any
	MinDelaySeconds int
	MaxDelaySeconds int
}

type ScriptWithSteps struct {
	Script repository.ConversationScript
	Steps  []repository.ConversationStep
}

type ScriptStore interface {
	Create(ctx context.Context, params repository.CreateConversationScriptParams) (repository.ConversationScript, error)
	List(ctx context.Context) ([]repository.ConversationScript, error)
}

type StepStore interface {
	Create(ctx context.Context, params repository.CreateConversationStepParams) (repository.ConversationStep, error)
	ListByScriptID(ctx context.Context, scriptID string) ([]repository.ConversationStep, error)
}

type Service struct {
	scripts ScriptStore
	steps   StepStore
}

func NewService(scripts ScriptStore, steps StepStore) Service {
	return Service{scripts: scripts, steps: steps}
}

func (s Service) Create(ctx context.Context, params CreateScriptParams) (ScriptWithSteps, error) {
	if params.Weight <= 0 {
		params.Weight = 1
	}
	if params.MaxWarmingScore <= 0 {
		params.MaxWarmingScore = 100
	}

	script, err := s.scripts.Create(ctx, repository.CreateConversationScriptParams{
		Name:            params.Name,
		Category:        params.Category,
		Enabled:         params.Enabled,
		Weight:          params.Weight,
		MinWarmingScore: params.MinWarmingScore,
		MaxWarmingScore: params.MaxWarmingScore,
	})
	if err != nil {
		return ScriptWithSteps{}, err
	}

	createdSteps := make([]repository.ConversationStep, 0, len(params.Steps))
	for _, step := range params.Steps {
		if step.MinDelaySeconds <= 0 {
			step.MinDelaySeconds = 10
		}
		if step.MaxDelaySeconds <= 0 {
			step.MaxDelaySeconds = 120
		}
		created, err := s.steps.Create(ctx, repository.CreateConversationStepParams{
			ScriptID:        script.ID,
			StepOrder:       step.StepOrder,
			SenderRole:      step.SenderRole,
			ActionType:      step.ActionType,
			TemplateID:      step.TemplateID,
			Payload:         step.Payload,
			MinDelaySeconds: step.MinDelaySeconds,
			MaxDelaySeconds: step.MaxDelaySeconds,
		})
		if err != nil {
			return ScriptWithSteps{}, err
		}
		createdSteps = append(createdSteps, created)
	}

	return ScriptWithSteps{Script: script, Steps: createdSteps}, nil
}

func (s Service) List(ctx context.Context) ([]ScriptWithSteps, error) {
	scripts, err := s.scripts.List(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]ScriptWithSteps, 0, len(scripts))
	for _, script := range scripts {
		steps, err := s.steps.ListByScriptID(ctx, script.ID)
		if err != nil {
			return nil, err
		}
		items = append(items, ScriptWithSteps{Script: script, Steps: steps})
	}
	return items, nil
}
