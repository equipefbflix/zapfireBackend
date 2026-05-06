package config

import (
	"testing"
	"time"
)

func TestLoadSchedulerConfigFromEnv(t *testing.T) {
	t.Setenv("SCHEDULER_ENABLED", "true")
	t.Setenv("SCHEDULER_TICK_SECONDS", "7")

	cfg := LoadSchedulerConfig()

	if !cfg.Enabled {
		t.Fatalf("Enabled = false")
	}
	if cfg.TickInterval != 7*time.Second {
		t.Fatalf("TickInterval = %s", cfg.TickInterval)
	}
}

func TestLoadSchedulerConfigDefaults(t *testing.T) {
	cfg := LoadSchedulerConfig()

	if !cfg.Enabled {
		t.Fatalf("Enabled = false")
	}
	if cfg.TickInterval != 5*time.Second {
		t.Fatalf("TickInterval = %s", cfg.TickInterval)
	}
}
