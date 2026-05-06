package config

import "time"

type ObservabilityConfig struct {
	LookbackWindow      time.Duration
	StaleRunningAfter   time.Duration
	StaleCleanupReason  string
}

func LoadObservabilityConfig() ObservabilityConfig {
	return ObservabilityConfig{
		LookbackWindow:     time.Duration(envInt("OBSERVABILITY_LOOKBACK_MINUTES", 60)) * time.Minute,
		StaleRunningAfter:  time.Duration(envInt("WARMING_STALE_RUNNING_MINUTES", 20)) * time.Minute,
		StaleCleanupReason: envString("WARMING_STALE_CLEANUP_REASON", "stale running job cleanup"),
	}
}
