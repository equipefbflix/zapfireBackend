package config

type PlannerConfig struct {
	MinDelaySeconds                  int
	MaxDelaySeconds                  int
	PairCooldownMinutes              int
	InboundTriggerCooldownSeconds    int
	WindowStartHour                  int
	WindowEndHour                    int
	MaxRunningJobsPerPair            int
	MaxRunningJobsPerEvolutionServer int
}

func LoadPlannerConfig() PlannerConfig {
	return PlannerConfig{
		MinDelaySeconds:                  envInt("WARMING_MIN_DELAY_SECONDS", 20),
		MaxDelaySeconds:                  envInt("WARMING_MAX_DELAY_SECONDS", 240),
		PairCooldownMinutes:              envInt("WARMING_PAIR_COOLDOWN_MINUTES", 30),
		InboundTriggerCooldownSeconds:    envInt("WARMING_INBOUND_TRIGGER_COOLDOWN_SECONDS", 90),
		WindowStartHour:                  envInt("WARMING_WINDOW_START_HOUR", 8),
		WindowEndHour:                    envInt("WARMING_WINDOW_END_HOUR", 22),
		MaxRunningJobsPerPair:            envInt("WARMING_MAX_RUNNING_JOBS_PER_PAIR", 1),
		MaxRunningJobsPerEvolutionServer: envInt("WARMING_MAX_RUNNING_JOBS_PER_EVOLUTION_SERVER", 5),
	}
}
