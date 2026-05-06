package config

import "time"

type SchedulerConfig struct {
	Enabled      bool
	TickInterval time.Duration
}

func LoadSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		Enabled:      envBool("SCHEDULER_ENABLED", true),
		TickInterval: time.Duration(envInt("SCHEDULER_TICK_SECONDS", 5)) * time.Second,
	}
}
