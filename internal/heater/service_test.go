package heater

import (
	"context"
	"testing"
	"time"

	"aquecedor-evolution/backend/internal/config"
	"aquecedor-evolution/backend/internal/planner"
	"aquecedor-evolution/backend/internal/repository"
)

type fakePhoneStore struct {
	phone   repository.PhoneNumber
	heaters []repository.PhoneNumber
}

func (s *fakePhoneStore) GetByID(ctx context.Context, id string) (repository.PhoneNumber, error) {
	return s.phone, nil
}

func (s *fakePhoneStore) ListByType(ctx context.Context, phoneType string) ([]repository.PhoneNumber, error) {
	return s.heaters, nil
}

type fakePlanner struct {
	plan planner.Plan
	err  error
}

func (p *fakePlanner) Plan(ctx context.Context, params planner.Params) (planner.Plan, error) {
	return p.plan, p.err
}

type fakeJobStore struct {
	created int
}

func (s *fakeJobStore) Create(ctx context.Context, params repository.CreateWarmingJobParams) (repository.WarmingJob, error) {
	s.created++
	return repository.WarmingJob{
		ID:          "job-id",
		PhoneAID:    params.PhoneAID,
		PhoneBID:    params.PhoneBID,
		Status:      "pending",
		ScheduledAt: params.ScheduledAt,
	}, nil
}

func TestActivateCreatesJobsForEachHeater(t *testing.T) {
	svc := NewService(
		config.PlannerConfig{},
		&fakePhoneStore{
			phone: repository.PhoneNumber{ID: "target-id", PhoneE164: "5511111111111", Status: "new"},
			heaters: []repository.PhoneNumber{
				{ID: "heater-1", PhoneE164: "5511999999999", ConnectionStatus: "open"},
				{ID: "heater-2", PhoneE164: "5511888888888", ConnectionStatus: "open"},
			},
		},
		&fakePlanner{
			plan: planner.Plan{
				Script:       repository.ConversationScript{ID: "script-id", Name: "test-script"},
				DelaySeconds: 60,
				ScheduledAt:  time.Now().UTC().Add(60 * time.Second),
			},
		},
		&fakeJobStore{},
	)

	if err := svc.Activate(context.Background(), "target-id"); err != nil {
		t.Fatalf("Activate() error = %v", err)
	}
}

func TestActivateSkipsHeatersWithoutOpenConnection(t *testing.T) {
	jobStore := &fakeJobStore{}
	svc := NewService(
		config.PlannerConfig{},
		&fakePhoneStore{
			phone: repository.PhoneNumber{ID: "target-id", PhoneE164: "5511111111111", Status: "new"},
			heaters: []repository.PhoneNumber{
				{ID: "heater-1", PhoneE164: "5511999999999", ConnectionStatus: "open"},
				{ID: "heater-2", PhoneE164: "5511888888888", ConnectionStatus: "close"},
				{ID: "heater-3", PhoneE164: "5511777777777", ConnectionStatus: ""},
			},
		},
		&fakePlanner{
			plan: planner.Plan{
				Script:       repository.ConversationScript{ID: "script-id", Name: "test-script"},
				DelaySeconds: 60,
				ScheduledAt:  time.Now().UTC().Add(60 * time.Second),
			},
		},
		jobStore,
	)

	if err := svc.Activate(context.Background(), "target-id"); err != nil {
		t.Fatalf("Activate() error = %v", err)
	}
	if jobStore.created != 1 {
		t.Fatalf("expected 1 job created, got %d", jobStore.created)
	}
}

func TestActivateNoHeaters(t *testing.T) {
	svc := NewService(
		config.PlannerConfig{},
		&fakePhoneStore{
			phone:   repository.PhoneNumber{ID: "target-id", PhoneE164: "5511111111111"},
			heaters: []repository.PhoneNumber{},
		},
		&fakePlanner{},
		&fakeJobStore{},
	)

	if err := svc.Activate(context.Background(), "target-id"); err != nil {
		t.Fatalf("Activate() error = %v", err)
	}
}

func TestActivateSkipsHeaterOnPlannerError(t *testing.T) {
	jobStore := &fakeJobStore{}
	svc := NewService(
		config.PlannerConfig{},
		&fakePhoneStore{
			phone: repository.PhoneNumber{ID: "target-id", PhoneE164: "5511111111111"},
			heaters: []repository.PhoneNumber{
				{ID: "heater-1", PhoneE164: "5511999999999", ConnectionStatus: "open"},
			},
		},
		&fakePlanner{
			err: nil,
			plan: planner.Plan{
				Script:       repository.ConversationScript{ID: "", Name: "invalid"},
				DelaySeconds: 0,
				ScheduledAt:  time.Time{},
			},
		},
		jobStore,
	)

	if err := svc.Activate(context.Background(), "target-id"); err != nil {
		t.Fatalf("Activate() error = %v", err)
	}
	if jobStore.created != 1 {
		t.Fatalf("expected 1 job created, got %d", jobStore.created)
	}
}
