package config

import "strconv"

type WarmingConfig struct {
	MinScoreToMarkWarm      float64
	ScoreMessageSuccess     float64
	ScoreReplySuccess       float64
	ScoreReactionSuccess    float64
	ScoreDailyActiveBonus   float64
	ScoreFailurePenalty     float64
	ScoreDisconnectedPenalty float64
}

func LoadWarmingConfig() WarmingConfig {
	return WarmingConfig{
		MinScoreToMarkWarm:       envFloat("WARMING_MIN_SCORE_TO_MARK_WARM", 80),
		ScoreMessageSuccess:      envFloat("WARMING_SCORE_MESSAGE_SUCCESS", 1.5),
		ScoreReplySuccess:        envFloat("WARMING_SCORE_REPLY_SUCCESS", 2.0),
		ScoreReactionSuccess:     envFloat("WARMING_SCORE_REACTION_SUCCESS", 0.5),
		ScoreDailyActiveBonus:    envFloat("WARMING_SCORE_DAILY_ACTIVE_BONUS", 3.0),
		ScoreFailurePenalty:      envFloat("WARMING_SCORE_FAILURE_PENALTY", 2.0),
		ScoreDisconnectedPenalty: envFloat("WARMING_SCORE_DISCONNECTED_PENALTY", 5.0),
	}
}

func envFloat(key string, fallback float64) float64 {
	value := envString(key, "")
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err == nil {
		return parsed
	}
	return fallback
}
