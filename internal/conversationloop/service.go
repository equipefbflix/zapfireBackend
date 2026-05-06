package conversationloop

import (
	"context"
	"regexp"
	"strings"
	"time"

	"aquecedor-evolution/backend/internal/config"
	"aquecedor-evolution/backend/internal/planner"
	"aquecedor-evolution/backend/internal/repository"
)

var nonDigits = regexp.MustCompile(`\D+`)

type InstanceStore interface {
	GetByInstanceName(ctx context.Context, instanceName string) (repository.Instance, error)
}

type PhoneStore interface {
	GetByID(ctx context.Context, id string) (repository.PhoneNumber, error)
	FindByE164(ctx context.Context, phoneE164 string) (*repository.PhoneNumber, error)
}

type Planner interface {
	Plan(ctx context.Context, params planner.Params) (planner.Plan, error)
}

type RecentJobStore interface {
	ListRecentByPair(ctx context.Context, phoneAID, phoneBID string, since time.Time) ([]repository.WarmingJob, error)
}

type JobStore interface {
	Create(ctx context.Context, params repository.CreateWarmingJobParams) (repository.WarmingJob, error)
}

type Service struct {
	cfg       config.PlannerConfig
	instances InstanceStore
	phones    PhoneStore
	planner   Planner
	recent    RecentJobStore
	jobs      JobStore
	now       func() time.Time
}

func NewService(cfg config.PlannerConfig, instances InstanceStore, phones PhoneStore, jobPlanner Planner, recent RecentJobStore, jobs JobStore, now func() time.Time) Service {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return Service{
		cfg:       cfg,
		instances: instances,
		phones:    phones,
		planner:   jobPlanner,
		recent:    recent,
		jobs:      jobs,
		now:       now,
	}
}

func (s Service) HandleInbound(ctx context.Context, event repository.EvolutionEvent) (*repository.WarmingJob, error) {
	if strings.ToUpper(event.EventType) != "MESSAGES_UPSERT" {
		return nil, nil
	}
	if boolField(event.Payload, "data", "key", "fromMe") {
		return nil, nil
	}

	remoteJID := firstString(
		stringField(event.Payload, "data", "key", "remoteJid"),
		stringField(event.Payload, "data", "key", "remoteJidAlt"),
	)
	if remoteJID == "" || strings.EqualFold(remoteJID, "status@broadcast") {
		return nil, nil
	}

	instance, err := s.instances.GetByInstanceName(ctx, event.InstanceName)
	if err != nil {
		return nil, err
	}
	localPhone, err := s.phones.GetByID(ctx, instance.PhoneNumberID)
	if err != nil {
		return nil, err
	}

	remotePhone, err := s.phones.FindByE164(ctx, normalizePhoneE164(remoteJID))
	if err != nil {
		return nil, err
	}
	if remotePhone == nil || remotePhone.ID == localPhone.ID {
		return nil, nil
	}

	cooldown := time.Duration(s.cfg.InboundTriggerCooldownSeconds) * time.Second
	if cooldown > 0 {
		recent, err := s.recent.ListRecentByPair(ctx, localPhone.ID, remotePhone.ID, s.now().Add(-cooldown))
		if err != nil {
			return nil, err
		}
		if len(recent) > 0 {
			return nil, nil
		}
	}

	pairScore := localPhone.WarmingScore
	if remotePhone.WarmingScore < pairScore {
		pairScore = remotePhone.WarmingScore
	}
	plan, err := s.planner.Plan(ctx, planner.Params{
		PhoneAID:  localPhone.ID,
		PhoneBID:  remotePhone.ID,
		PairScore: pairScore,
		Category:  "reactive",
	})
	if err != nil {
		return nil, err
	}

	messageID := firstString(
		stringField(event.Payload, "data", "key", "id"),
		stringField(event.Payload, "data", "messageId"),
	)
	messageType := stringField(event.Payload, "data", "messageType")
	created, err := s.jobs.Create(ctx, repository.CreateWarmingJobParams{
		ScriptID:    &plan.Script.ID,
		PhoneAID:    localPhone.ID,
		PhoneBID:    remotePhone.ID,
		ScheduledAt: plan.ScheduledAt,
		Metadata: map[string]any{
			"autoReactive":        true,
			"phoneAE164":          localPhone.PhoneE164,
			"phoneBE164":          remotePhone.PhoneE164,
			"triggerEventId":      event.ID,
			"triggerMessageId":    messageID,
			"triggerInstanceName": event.InstanceName,
			"triggerRemoteJid":    remoteJID,
			"triggerMessageType":  messageType,
		},
	})
	if err != nil {
		return nil, err
	}
	return &created, nil
}

func stringField(payload map[string]any, keys ...string) string {
	current := any(payload)
	for _, key := range keys {
		m, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current, ok = m[key]
		if !ok {
			return ""
		}
	}
	text, _ := current.(string)
	return text
}

func boolField(payload map[string]any, keys ...string) bool {
	current := any(payload)
	for _, key := range keys {
		m, ok := current.(map[string]any)
		if !ok {
			return false
		}
		current, ok = m[key]
		if !ok {
			return false
		}
	}
	value, _ := current.(bool)
	return value
}

func firstString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func normalizePhoneE164(value string) string {
	base := value
	if index := strings.Index(base, ":"); index >= 0 {
		base = base[:index]
	}
	if index := strings.Index(base, "@"); index >= 0 {
		base = base[:index]
	}
	return nonDigits.ReplaceAllString(base, "")
}
