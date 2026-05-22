package heater

import (
	"context"
	"fmt"
	"log/slog"

	"aquecedor-evolution/backend/internal/config"
	"aquecedor-evolution/backend/internal/planner"
	"aquecedor-evolution/backend/internal/repository"
)

type PhoneStore interface {
	GetByID(ctx context.Context, id string) (repository.PhoneNumber, error)
	ListByType(ctx context.Context, phoneType string) ([]repository.PhoneNumber, error)
}

type Planner interface {
	Plan(ctx context.Context, params planner.Params) (planner.Plan, error)
}

type JobStore interface {
	Create(ctx context.Context, params repository.CreateWarmingJobParams) (repository.WarmingJob, error)
}

type Service struct {
	cfg     config.PlannerConfig
	phones  PhoneStore
	planner Planner
	jobs    JobStore
}

func NewService(cfg config.PlannerConfig, phones PhoneStore, p Planner, jobs JobStore) Service {
	return Service{
		cfg:     cfg,
		phones:  phones,
		planner: p,
		jobs:    jobs,
	}
}

func (s Service) Activate(ctx context.Context, targetPhoneID string) error {
	target, err := s.phones.GetByID(ctx, targetPhoneID)
	if err != nil {
		return fmt.Errorf("get target phone: %w", err)
	}

	heaters, err := s.phones.ListByType(ctx, repository.PhoneTypeHeater)
	if err != nil {
		return fmt.Errorf("list heaters: %w", err)
	}

	if len(heaters) == 0 {
		slog.Warn("no heater numbers found to warm target", "targetPhoneID", targetPhoneID)
		return nil
	}

	var created int
	for _, heater := range heaters {
		if heater.ConnectionStatus != "open" {
			continue
		}
		if heater.ID == target.ID {
			continue
		}

		plan, err := s.planner.Plan(ctx, planner.Params{
			PhoneAID:  heater.ID,
			PhoneBID:  target.ID,
			PairScore: target.WarmingScore,
			Category:  "",
		})
		if err != nil {
			slog.Warn("skip heater: no eligible script",
				"heaterID", heater.ID,
				"targetID", target.ID,
				"error", err,
			)
			continue
		}

		_, err = s.jobs.Create(ctx, repository.CreateWarmingJobParams{
			ScriptID:    &plan.Script.ID,
			PhoneAID:    heater.ID,
			PhoneBID:    target.ID,
			ScheduledAt: plan.ScheduledAt,
			Metadata: map[string]any{
				"autoHeater":   true,
				"heaterPhone":  heater.PhoneE164,
				"targetPhone":  target.PhoneE164,
				"heaterID":      heater.ID,
				"delaySeconds": plan.DelaySeconds,
			},
		})
		if err != nil {
			slog.Error("failed to create warming job",
				"heaterID", heater.ID,
				"targetID", target.ID,
				"error", err,
			)
			continue
		}
		created++
	}

	slog.Info("heater activation complete",
		"targetPhoneID", targetPhoneID,
		"jobsCreated", created,
		"heatersConsidered", len(heaters),
	)

	if created == 0 {
		slog.Warn("no warming jobs were created for target",
			"targetPhoneID", targetPhoneID,
			"heatersAvailable", len(heaters),
		)
	}

	return nil
}
