package config

import (
	"testing"
	"time"
)

func TestLoadObservabilityConfigDefaults(t *testing.T) {
	cfg := LoadObservabilityConfig()

	if cfg.LookbackWindow != 60*time.Minute {
		t.Fatalf("LookbackWindow = %s", cfg.LookbackWindow)
	}
	if cfg.StaleRunningAfter != 20*time.Minute {
		t.Fatalf("StaleRunningAfter = %s", cfg.StaleRunningAfter)
	}
	if cfg.StaleCleanupReason != "stale running job cleanup" {
		t.Fatalf("StaleCleanupReason = %q", cfg.StaleCleanupReason)
	}
}

func TestLoadObservabilityConfigFromEnv(t *testing.T) {
	t.Setenv("OBSERVABILITY_LOOKBACK_MINUTES", "15")
	t.Setenv("WARMING_STALE_RUNNING_MINUTES", "45")
	t.Setenv("WARMING_STALE_CLEANUP_REASON", "custom reason")

	cfg := LoadObservabilityConfig()

	if cfg.LookbackWindow != 15*time.Minute {
		t.Fatalf("LookbackWindow = %s", cfg.LookbackWindow)
	}
	if cfg.StaleRunningAfter != 45*time.Minute {
		t.Fatalf("StaleRunningAfter = %s", cfg.StaleRunningAfter)
	}
	if cfg.StaleCleanupReason != "custom reason" {
		t.Fatalf("StaleCleanupReason = %q", cfg.StaleCleanupReason)
	}
}
