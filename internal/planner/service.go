package planner

import (
	"context"
	"errors"
	"hash/fnv"
	"sort"
	"time"

	"aquecedor-evolution/backend/internal/config"
	"aquecedor-evolution/backend/internal/repository"
)

type ScriptStore interface {
	List(ctx context.Context) ([]repository.ConversationScript, error)
}

type RecentJobStore interface {
	ListRecentByPair(ctx context.Context, phoneAID, phoneBID string, since time.Time) ([]repository.WarmingJob, error)
}

type Params struct {
	PhoneAID  string
	PhoneBID  string
	PairScore float64
	Category  string
}

type Plan struct {
	Script       repository.ConversationScript
	DelaySeconds int
	ScheduledAt  time.Time
}

type Service struct {
	cfg    config.PlannerConfig
	script ScriptStore
	jobs   RecentJobStore
	now    func() time.Time
}

func NewService(cfg config.PlannerConfig, script ScriptStore, jobs RecentJobStore, now func() time.Time) Service {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return Service{cfg: cfg, script: script, jobs: jobs, now: now}
}

func (s Service) Plan(ctx context.Context, params Params) (Plan, error) {
	scripts, err := s.script.List(ctx)
	if err != nil {
		return Plan{}, err
	}

	eligible := make([]repository.ConversationScript, 0, len(scripts))
	for _, script := range scripts {
		if !script.Enabled {
			continue
		}
		if params.PairScore < script.MinWarmingScore || params.PairScore > script.MaxWarmingScore {
			continue
		}
		if params.Category != "" && script.Category != params.Category {
			continue
		}
		eligible = append(eligible, script)
	}
	if len(eligible) == 0 {
		return Plan{}, errors.New("no eligible conversation script")
	}

	since := s.now().Add(-time.Duration(s.cfg.PairCooldownMinutes) * time.Minute)
	recentJobs, err := s.jobs.ListRecentByPair(ctx, params.PhoneAID, params.PhoneBID, since)
	if err != nil {
		return Plan{}, err
	}
	recentScripts := map[string]time.Time{}
	for _, job := range recentJobs {
		if job.ScriptID != nil {
			if existing, ok := recentScripts[*job.ScriptID]; !ok || job.ScheduledAt.After(existing) {
				recentScripts[*job.ScriptID] = job.ScheduledAt
			}
		}
	}

	candidates := make([]repository.ConversationScript, 0, len(eligible))
	for _, script := range eligible {
		if _, used := recentScripts[script.ID]; !used {
			candidates = append(candidates, script)
		}
	}
	if len(candidates) == 0 {
		candidates = eligible
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Weight != candidates[j].Weight {
			return candidates[i].Weight > candidates[j].Weight
		}
		ti := recentScripts[candidates[i].ID]
		tj := recentScripts[candidates[j].ID]
		if !ti.Equal(tj) {
			return ti.Before(tj)
		}
		return candidates[i].Name < candidates[j].Name
	})

	selected := candidates[0]
	delay := s.computeDelay(params.PhoneAID, params.PhoneBID, selected.ID)
	scheduledAt := moveIntoWindow(s.now().Add(time.Duration(delay)*time.Second), s.cfg.WindowStartHour, s.cfg.WindowEndHour)

	return Plan{
		Script:       selected,
		DelaySeconds: delay,
		ScheduledAt:  scheduledAt,
	}, nil
}

func (s Service) computeDelay(phoneAID, phoneBID, scriptID string) int {
	min := s.cfg.MinDelaySeconds
	max := s.cfg.MaxDelaySeconds
	if max <= min {
		return min
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(phoneAID + ":" + phoneBID + ":" + scriptID + ":" + s.now().Format("2006-01-02")))
	span := max - min + 1
	return min + int(h.Sum32()%uint32(span))
}

func moveIntoWindow(candidate time.Time, startHour, endHour int) time.Time {
	if startHour < 0 {
		startHour = 0
	}
	if endHour > 23 {
		endHour = 23
	}
	if startHour >= endHour {
		return candidate
	}
	if candidate.Hour() < startHour {
		return time.Date(candidate.Year(), candidate.Month(), candidate.Day(), startHour, 0, 0, 0, candidate.Location())
	}
	if candidate.Hour() >= endHour {
		next := candidate.AddDate(0, 0, 1)
		return time.Date(next.Year(), next.Month(), next.Day(), startHour, 0, 0, 0, candidate.Location())
	}
	return candidate
}
