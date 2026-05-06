package config

import "testing"

func TestLoadPlannerConfigFromEnv(t *testing.T) {
	t.Setenv("WARMING_MIN_DELAY_SECONDS", "15")
	t.Setenv("WARMING_MAX_DELAY_SECONDS", "180")
	t.Setenv("WARMING_PAIR_COOLDOWN_MINUTES", "45")
	t.Setenv("WARMING_WINDOW_START_HOUR", "9")
	t.Setenv("WARMING_WINDOW_END_HOUR", "21")
	t.Setenv("WARMING_MAX_RUNNING_JOBS_PER_PAIR", "2")
	t.Setenv("WARMING_MAX_RUNNING_JOBS_PER_EVOLUTION_SERVER", "7")
	t.Setenv("WARMING_INBOUND_TRIGGER_COOLDOWN_SECONDS", "90")

	cfg := LoadPlannerConfig()

	if cfg.MinDelaySeconds != 15 {
		t.Fatalf("MinDelaySeconds = %d", cfg.MinDelaySeconds)
	}
	if cfg.MaxDelaySeconds != 180 {
		t.Fatalf("MaxDelaySeconds = %d", cfg.MaxDelaySeconds)
	}
	if cfg.PairCooldownMinutes != 45 {
		t.Fatalf("PairCooldownMinutes = %d", cfg.PairCooldownMinutes)
	}
	if cfg.WindowStartHour != 9 {
		t.Fatalf("WindowStartHour = %d", cfg.WindowStartHour)
	}
	if cfg.WindowEndHour != 21 {
		t.Fatalf("WindowEndHour = %d", cfg.WindowEndHour)
	}
	if cfg.MaxRunningJobsPerPair != 2 {
		t.Fatalf("MaxRunningJobsPerPair = %d", cfg.MaxRunningJobsPerPair)
	}
	if cfg.MaxRunningJobsPerEvolutionServer != 7 {
		t.Fatalf("MaxRunningJobsPerEvolutionServer = %d", cfg.MaxRunningJobsPerEvolutionServer)
	}
	if cfg.InboundTriggerCooldownSeconds != 90 {
		t.Fatalf("InboundTriggerCooldownSeconds = %d", cfg.InboundTriggerCooldownSeconds)
	}
}

func TestLoadPlannerConfigDefaults(t *testing.T) {
	cfg := LoadPlannerConfig()

	if cfg.MinDelaySeconds != 20 {
		t.Fatalf("MinDelaySeconds = %d", cfg.MinDelaySeconds)
	}
	if cfg.MaxDelaySeconds != 240 {
		t.Fatalf("MaxDelaySeconds = %d", cfg.MaxDelaySeconds)
	}
	if cfg.PairCooldownMinutes != 30 {
		t.Fatalf("PairCooldownMinutes = %d", cfg.PairCooldownMinutes)
	}
	if cfg.WindowStartHour != 8 {
		t.Fatalf("WindowStartHour = %d", cfg.WindowStartHour)
	}
	if cfg.WindowEndHour != 22 {
		t.Fatalf("WindowEndHour = %d", cfg.WindowEndHour)
	}
	if cfg.MaxRunningJobsPerPair != 1 {
		t.Fatalf("MaxRunningJobsPerPair = %d", cfg.MaxRunningJobsPerPair)
	}
	if cfg.MaxRunningJobsPerEvolutionServer != 5 {
		t.Fatalf("MaxRunningJobsPerEvolutionServer = %d", cfg.MaxRunningJobsPerEvolutionServer)
	}
	if cfg.InboundTriggerCooldownSeconds != 90 {
		t.Fatalf("InboundTriggerCooldownSeconds = %d", cfg.InboundTriggerCooldownSeconds)
	}
}
