package warmingscore

import (
	"context"
	"testing"
	"time"

	"aquecedor-evolution/backend/internal/config"
	"aquecedor-evolution/backend/internal/repository"
)

type fakeMetricsStore struct {
	phoneID string
	metrics repository.PhoneWarmingMetrics
}

func (s *fakeMetricsStore) BuildWarmingMetrics(ctx context.Context, phoneNumberID string) (repository.PhoneWarmingMetrics, error) {
	s.phoneID = phoneNumberID
	return s.metrics, nil
}

type fakePhoneStateStore struct {
	phoneID string
	score   float64
	status  string
}

func (s *fakePhoneStateStore) UpdateWarmingState(ctx context.Context, phoneNumberID string, score float64, status string) error {
	s.phoneID = phoneNumberID
	s.score = score
	s.status = status
	return nil
}

func TestServiceRecalculateWarm(t *testing.T) {
	metrics := &fakeMetricsStore{metrics: repository.PhoneWarmingMetrics{
		SuccessTextCount:     20,
		SuccessReplyCount:    10,
		SuccessReactionCount: 10,
		FailureCount:         2,
		LastActivityAt:       timePointer(time.Now().UTC()),
	}}
	phones := &fakePhoneStateStore{}
	service := NewService(config.WarmingConfig{
		MinScoreToMarkWarm:       30,
		ScoreMessageSuccess:      1.5,
		ScoreReplySuccess:        2.0,
		ScoreReactionSuccess:     0.5,
		ScoreDailyActiveBonus:    3.0,
		ScoreFailurePenalty:      2.0,
		ScoreDisconnectedPenalty: 5.0,
	}, metrics, phones)

	score, status, err := service.Recalculate(context.Background(), "phone-id")
	if err != nil {
		t.Fatalf("Recalculate() error = %v", err)
	}
	if score <= 30 {
		t.Fatalf("score = %v", score)
	}
	if status != "warm" {
		t.Fatalf("status = %q", status)
	}
	if phones.phoneID != "phone-id" {
		t.Fatalf("phoneID = %q", phones.phoneID)
	}
}

func TestServiceRecalculateWarming(t *testing.T) {
	metrics := &fakeMetricsStore{metrics: repository.PhoneWarmingMetrics{
		SuccessTextCount: 2,
		LastActivityAt:   timePointer(time.Now().UTC()),
	}}
	phones := &fakePhoneStateStore{}
	service := NewService(config.WarmingConfig{
		MinScoreToMarkWarm:       80,
		ScoreMessageSuccess:      1.5,
		ScoreReplySuccess:        2.0,
		ScoreReactionSuccess:     0.5,
		ScoreDailyActiveBonus:    3.0,
		ScoreFailurePenalty:      2.0,
		ScoreDisconnectedPenalty: 5.0,
	}, metrics, phones)

	score, status, err := service.Recalculate(context.Background(), "phone-id")
	if err != nil {
		t.Fatalf("Recalculate() error = %v", err)
	}
	if score <= 0 {
		t.Fatalf("score = %v", score)
	}
	if status != "warming" {
		t.Fatalf("status = %q", status)
	}
}

func TestServiceRecalculateNewWhenNoActivity(t *testing.T) {
	service := NewService(config.WarmingConfig{
		MinScoreToMarkWarm:       80,
		ScoreMessageSuccess:      1.5,
		ScoreReplySuccess:        2.0,
		ScoreReactionSuccess:     0.5,
		ScoreDailyActiveBonus:    3.0,
		ScoreFailurePenalty:      2.0,
		ScoreDisconnectedPenalty: 5.0,
	}, &fakeMetricsStore{}, &fakePhoneStateStore{})

	score, status, err := service.Recalculate(context.Background(), "phone-id")
	if err != nil {
		t.Fatalf("Recalculate() error = %v", err)
	}
	if score != 0 {
		t.Fatalf("score = %v", score)
	}
	if status != "new" {
		t.Fatalf("status = %q", status)
	}
}

func timePointer(value time.Time) *time.Time { return &value }
