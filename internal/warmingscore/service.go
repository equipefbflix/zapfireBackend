package warmingscore

import (
	"context"
	"math"
	"time"

	"aquecedor-evolution/backend/internal/config"
	"aquecedor-evolution/backend/internal/repository"
)

type MetricsStore interface {
	BuildWarmingMetrics(ctx context.Context, phoneNumberID string) (repository.PhoneWarmingMetrics, error)
}

type PhoneStateStore interface {
	UpdateWarmingState(ctx context.Context, phoneNumberID string, score float64, status string) error
}

type Service struct {
	cfg     config.WarmingConfig
	metrics MetricsStore
	phones  PhoneStateStore
}

func NewService(cfg config.WarmingConfig, metrics MetricsStore, phones PhoneStateStore) Service {
	return Service{cfg: cfg, metrics: metrics, phones: phones}
}

func (s Service) Recalculate(ctx context.Context, phoneNumberID string) (float64, string, error) {
	metrics, err := s.metrics.BuildWarmingMetrics(ctx, phoneNumberID)
	if err != nil {
		return 0, "", err
	}

	score := 0.0
	score += float64(metrics.SuccessTextCount) * s.cfg.ScoreMessageSuccess
	score += float64(metrics.SuccessReplyCount) * s.cfg.ScoreReplySuccess
	score += float64(metrics.SuccessReactionCount) * s.cfg.ScoreReactionSuccess
	score -= float64(metrics.FailureCount) * s.cfg.ScoreFailurePenalty
	score -= float64(metrics.DisconnectedCount) * s.cfg.ScoreDisconnectedPenalty
	if isActiveToday(metrics.LastActivityAt) {
		score += s.cfg.ScoreDailyActiveBonus
	}

	score = math.Max(0, math.Min(100, score))

	status := "new"
	if score >= s.cfg.MinScoreToMarkWarm {
		status = "warm"
	} else if score > 0 {
		status = "warming"
	}

	if err := s.phones.UpdateWarmingState(ctx, phoneNumberID, score, status); err != nil {
		return 0, "", err
	}

	return score, status, nil
}

func isActiveToday(lastActivityAt *time.Time) bool {
	if lastActivityAt == nil {
		return false
	}
	now := time.Now().UTC()
	y1, m1, d1 := now.Date()
	y2, m2, d2 := lastActivityAt.UTC().Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}
