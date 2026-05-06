package conversationloop

import (
	"context"
	"testing"
	"time"

	"aquecedor-evolution/backend/internal/config"
	"aquecedor-evolution/backend/internal/planner"
	"aquecedor-evolution/backend/internal/repository"
)

type fakeInstanceStore struct {
	instance repository.Instance
	name     string
}

func (s *fakeInstanceStore) GetByInstanceName(ctx context.Context, instanceName string) (repository.Instance, error) {
	s.name = instanceName
	return s.instance, nil
}

type fakePhoneStore struct {
	local      repository.PhoneNumber
	remote     *repository.PhoneNumber
	localID    string
	remoteE164 string
}

func (s *fakePhoneStore) GetByID(ctx context.Context, id string) (repository.PhoneNumber, error) {
	s.localID = id
	return s.local, nil
}

func (s *fakePhoneStore) FindByE164(ctx context.Context, phoneE164 string) (*repository.PhoneNumber, error) {
	s.remoteE164 = phoneE164
	return s.remote, nil
}

type fakePlanner struct {
	params planner.Params
	plan   planner.Plan
}

func (p *fakePlanner) Plan(ctx context.Context, params planner.Params) (planner.Plan, error) {
	p.params = params
	return p.plan, nil
}

type fakeRecentJobs struct {
	jobs []repository.WarmingJob
}

func (s *fakeRecentJobs) ListRecentByPair(ctx context.Context, phoneAID, phoneBID string, since time.Time) ([]repository.WarmingJob, error) {
	return s.jobs, nil
}

type fakeJobStore struct {
	params repository.CreateWarmingJobParams
	job    repository.WarmingJob
}

func (s *fakeJobStore) Create(ctx context.Context, params repository.CreateWarmingJobParams) (repository.WarmingJob, error) {
	s.params = params
	return s.job, nil
}

func TestServiceIgnoresFromMe(t *testing.T) {
	service := NewService(
		config.PlannerConfig{InboundTriggerCooldownSeconds: 90},
		&fakeInstanceStore{},
		&fakePhoneStore{},
		&fakePlanner{},
		&fakeRecentJobs{},
		&fakeJobStore{},
		func() time.Time { return time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC) },
	)

	job, err := service.HandleInbound(context.Background(), repository.EvolutionEvent{
		ID:           "event-id",
		InstanceName: "chip-a",
		EventType:    "MESSAGES_UPSERT",
		Payload: map[string]any{
			"data": map[string]any{
				"key": map[string]any{
					"fromMe": true,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("HandleInbound() error = %v", err)
	}
	if job != nil {
		t.Fatalf("expected nil job")
	}
}

func TestServiceCreatesReactiveJobForManagedInbound(t *testing.T) {
	now := time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC)
	remote := &repository.PhoneNumber{ID: "phone-b", PhoneE164: "5511999999999", WarmingScore: 42}
	plan := planner.Plan{
		Script:       repository.ConversationScript{ID: "script-1", Name: "reactive-1"},
		ScheduledAt:  now.Add(45 * time.Second),
		DelaySeconds: 45,
	}
	jobs := &fakeJobStore{job: repository.WarmingJob{ID: "job-1"}}
	service := NewService(
		config.PlannerConfig{InboundTriggerCooldownSeconds: 90},
		&fakeInstanceStore{instance: repository.Instance{PhoneNumberID: "phone-a"}},
		&fakePhoneStore{local: repository.PhoneNumber{ID: "phone-a", PhoneE164: "5511888888888", WarmingScore: 15}, remote: remote},
		&fakePlanner{plan: plan},
		&fakeRecentJobs{},
		jobs,
		func() time.Time { return now },
	)

	job, err := service.HandleInbound(context.Background(), repository.EvolutionEvent{
		ID:           "event-id",
		InstanceName: "chip-a",
		EventType:    "MESSAGES_UPSERT",
		Payload: map[string]any{
			"data": map[string]any{
				"key": map[string]any{
					"id":           "message-id",
					"fromMe":       false,
					"remoteJid":    "5511999999999:42@s.whatsapp.net",
					"remoteJidAlt": "106129128444155@lid",
				},
				"messageType": "conversation",
			},
		},
	})
	if err != nil {
		t.Fatalf("HandleInbound() error = %v", err)
	}
	if job == nil {
		t.Fatalf("expected created job")
	}
	if jobs.params.PhoneAID != "phone-a" {
		t.Fatalf("PhoneAID = %q", jobs.params.PhoneAID)
	}
	if jobs.params.PhoneBID != "phone-b" {
		t.Fatalf("PhoneBID = %q", jobs.params.PhoneBID)
	}
	if jobs.params.ScriptID == nil || *jobs.params.ScriptID != "script-1" {
		t.Fatalf("ScriptID = %v", jobs.params.ScriptID)
	}
	if jobs.params.Metadata["autoReactive"] != true {
		t.Fatalf("autoReactive metadata missing: %#v", jobs.params.Metadata)
	}
	if jobs.params.Metadata["triggerMessageId"] != "message-id" {
		t.Fatalf("triggerMessageId = %#v", jobs.params.Metadata["triggerMessageId"])
	}
}

func TestServiceSkipsWhenPairHasRecentJob(t *testing.T) {
	now := time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC)
	service := NewService(
		config.PlannerConfig{InboundTriggerCooldownSeconds: 90},
		&fakeInstanceStore{instance: repository.Instance{PhoneNumberID: "phone-a"}},
		&fakePhoneStore{
			local:  repository.PhoneNumber{ID: "phone-a", PhoneE164: "5511888888888", WarmingScore: 15},
			remote: &repository.PhoneNumber{ID: "phone-b", PhoneE164: "5511999999999", WarmingScore: 42},
		},
		&fakePlanner{},
		&fakeRecentJobs{jobs: []repository.WarmingJob{{ID: "recent-job"}}},
		&fakeJobStore{},
		func() time.Time { return now },
	)

	job, err := service.HandleInbound(context.Background(), repository.EvolutionEvent{
		ID:           "event-id",
		InstanceName: "chip-a",
		EventType:    "MESSAGES_UPSERT",
		Payload: map[string]any{
			"data": map[string]any{
				"key": map[string]any{
					"fromMe":    false,
					"remoteJid": "5511999999999@s.whatsapp.net",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("HandleInbound() error = %v", err)
	}
	if job != nil {
		t.Fatalf("expected nil job")
	}
}
