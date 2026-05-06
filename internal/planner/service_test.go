package planner

import (
	"context"
	"testing"
	"time"

	"aquecedor-evolution/backend/internal/config"
	"aquecedor-evolution/backend/internal/repository"
)

type fakeScriptStore struct {
	items []repository.ConversationScript
}

func (s fakeScriptStore) List(ctx context.Context) ([]repository.ConversationScript, error) {
	return s.items, nil
}

type fakeRecentJobStore struct {
	items []repository.WarmingJob
}

func (s fakeRecentJobStore) ListRecentByPair(ctx context.Context, phoneAID, phoneBID string, since time.Time) ([]repository.WarmingJob, error) {
	return s.items, nil
}

func TestPlannerSelectsByScoreAndCategory(t *testing.T) {
	now := time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC)
	service := NewService(config.PlannerConfig{
		MinDelaySeconds:       20,
		MaxDelaySeconds:       40,
		PairCooldownMinutes:   30,
		WindowStartHour:       8,
		WindowEndHour:         22,
	}, fakeScriptStore{items: []repository.ConversationScript{
		{ID: "script-a", Name: "casual_1", Category: "casual", Enabled: true, Weight: 5, MinWarmingScore: 0, MaxWarmingScore: 50},
		{ID: "script-b", Name: "business_1", Category: "business", Enabled: true, Weight: 10, MinWarmingScore: 0, MaxWarmingScore: 50},
	}}, fakeRecentJobStore{}, func() time.Time { return now })

	plan, err := service.Plan(context.Background(), Params{
		PhoneAID:   "phone-a",
		PhoneBID:   "phone-b",
		PairScore:  20,
		Category:   "business",
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if plan.Script.ID != "script-b" {
		t.Fatalf("script id = %q", plan.Script.ID)
	}
	if plan.DelaySeconds < 20 || plan.DelaySeconds > 40 {
		t.Fatalf("delay = %d", plan.DelaySeconds)
	}
}

func TestPlannerSkipsRecentScriptWithinCooldown(t *testing.T) {
	now := time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC)
	scriptA := "script-a"
	service := NewService(config.PlannerConfig{
		MinDelaySeconds:       20,
		MaxDelaySeconds:       20,
		PairCooldownMinutes:   30,
		WindowStartHour:       8,
		WindowEndHour:         22,
	}, fakeScriptStore{items: []repository.ConversationScript{
		{ID: "script-a", Name: "casual_1", Category: "casual", Enabled: true, Weight: 10, MinWarmingScore: 0, MaxWarmingScore: 100},
		{ID: "script-b", Name: "casual_2", Category: "casual", Enabled: true, Weight: 5, MinWarmingScore: 0, MaxWarmingScore: 100},
	}}, fakeRecentJobStore{items: []repository.WarmingJob{
		{ID: "job-1", ScriptID: &scriptA, PhoneAID: "phone-a", PhoneBID: "phone-b", ScheduledAt: now.Add(-10 * time.Minute)},
	}}, func() time.Time { return now })

	plan, err := service.Plan(context.Background(), Params{
		PhoneAID:  "phone-a",
		PhoneBID:  "phone-b",
		PairScore: 30,
		Category:  "casual",
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if plan.Script.ID != "script-b" {
		t.Fatalf("script id = %q", plan.Script.ID)
	}
}

func TestPlannerFallsBackWhenAllScriptsRecent(t *testing.T) {
	now := time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC)
	scriptA := "script-a"
	service := NewService(config.PlannerConfig{
		MinDelaySeconds:       20,
		MaxDelaySeconds:       20,
		PairCooldownMinutes:   30,
		WindowStartHour:       8,
		WindowEndHour:         22,
	}, fakeScriptStore{items: []repository.ConversationScript{
		{ID: "script-a", Name: "casual_1", Category: "casual", Enabled: true, Weight: 10, MinWarmingScore: 0, MaxWarmingScore: 100},
	}}, fakeRecentJobStore{items: []repository.WarmingJob{
		{ID: "job-1", ScriptID: &scriptA, PhoneAID: "phone-a", PhoneBID: "phone-b", ScheduledAt: now.Add(-10 * time.Minute)},
	}}, func() time.Time { return now })

	plan, err := service.Plan(context.Background(), Params{
		PhoneAID:  "phone-a",
		PhoneBID:  "phone-b",
		PairScore: 30,
		Category:  "casual",
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if plan.Script.ID != "script-a" {
		t.Fatalf("script id = %q", plan.Script.ID)
	}
}

func TestPlannerMovesScheduleIntoWindow(t *testing.T) {
	now := time.Date(2026, 5, 5, 23, 30, 0, 0, time.UTC)
	service := NewService(config.PlannerConfig{
		MinDelaySeconds:       20,
		MaxDelaySeconds:       20,
		PairCooldownMinutes:   30,
		WindowStartHour:       8,
		WindowEndHour:         22,
	}, fakeScriptStore{items: []repository.ConversationScript{
		{ID: "script-a", Name: "casual_1", Category: "casual", Enabled: true, Weight: 10, MinWarmingScore: 0, MaxWarmingScore: 100},
	}}, fakeRecentJobStore{}, func() time.Time { return now })

	plan, err := service.Plan(context.Background(), Params{
		PhoneAID:  "phone-a",
		PhoneBID:  "phone-b",
		PairScore: 30,
		Category:  "casual",
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if plan.ScheduledAt.Hour() != 8 {
		t.Fatalf("scheduled hour = %d", plan.ScheduledAt.Hour())
	}
}
