package config

import "testing"

func TestLoadWarmingConfigDefaults(t *testing.T) {
	cfg := LoadWarmingConfig()
	if cfg.MinScoreToMarkWarm != 80 {
		t.Fatalf("MinScoreToMarkWarm = %v", cfg.MinScoreToMarkWarm)
	}
	if cfg.ScoreReplySuccess != 2.0 {
		t.Fatalf("ScoreReplySuccess = %v", cfg.ScoreReplySuccess)
	}
}

func TestLoadWarmingConfigFromEnv(t *testing.T) {
	t.Setenv("WARMING_MIN_SCORE_TO_MARK_WARM", "65")
	t.Setenv("WARMING_SCORE_MESSAGE_SUCCESS", "2.5")
	t.Setenv("WARMING_SCORE_FAILURE_PENALTY", "4.5")

	cfg := LoadWarmingConfig()
	if cfg.MinScoreToMarkWarm != 65 {
		t.Fatalf("MinScoreToMarkWarm = %v", cfg.MinScoreToMarkWarm)
	}
	if cfg.ScoreMessageSuccess != 2.5 {
		t.Fatalf("ScoreMessageSuccess = %v", cfg.ScoreMessageSuccess)
	}
	if cfg.ScoreFailurePenalty != 4.5 {
		t.Fatalf("ScoreFailurePenalty = %v", cfg.ScoreFailurePenalty)
	}
}
