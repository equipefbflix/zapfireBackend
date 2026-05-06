package config

import (
	"testing"
	"time"
)

func TestLoadRabbitMQConfigFromEnv(t *testing.T) {
	t.Setenv("RABBITMQ_URL", "amqp://user:pass@rabbitmq.example.com:5672/")
	t.Setenv("RABBITMQ_EXCHANGE", "aquecedor.events")
	t.Setenv("RABBITMQ_QUEUE_WARMING_JOBS", "aquecedor.warming.jobs")
	t.Setenv("RABBITMQ_QUEUE_EVOLUTION_EVENTS", "aquecedor.evolution.events")
	t.Setenv("RABBITMQ_QUEUE_DEAD_LETTER", "aquecedor.dead_letter")
	t.Setenv("RABBITMQ_PREFETCH", "15")
	t.Setenv("RABBITMQ_PUBLISH_TIMEOUT_SECONDS", "7")
	t.Setenv("RABBITMQ_CONSUMER_ENABLED", "true")
	t.Setenv("RABBITMQ_MAX_RETRIES", "4")

	cfg, err := LoadRabbitMQConfig()
	if err != nil {
		t.Fatalf("LoadRabbitMQConfig() error = %v", err)
	}

	if cfg.URL != "amqp://user:pass@rabbitmq.example.com:5672/" {
		t.Fatalf("URL = %q", cfg.URL)
	}
	if cfg.Exchange != "aquecedor.events" {
		t.Fatalf("Exchange = %q", cfg.Exchange)
	}
	if cfg.WarmingJobsQueue != "aquecedor.warming.jobs" {
		t.Fatalf("WarmingJobsQueue = %q", cfg.WarmingJobsQueue)
	}
	if cfg.EvolutionEventsQueue != "aquecedor.evolution.events" {
		t.Fatalf("EvolutionEventsQueue = %q", cfg.EvolutionEventsQueue)
	}
	if cfg.DeadLetterQueue != "aquecedor.dead_letter" {
		t.Fatalf("DeadLetterQueue = %q", cfg.DeadLetterQueue)
	}
	if cfg.Prefetch != 15 {
		t.Fatalf("Prefetch = %d", cfg.Prefetch)
	}
	if cfg.PublishTimeout != 7*time.Second {
		t.Fatalf("PublishTimeout = %s", cfg.PublishTimeout)
	}
	if !cfg.ConsumerEnabled {
		t.Fatal("ConsumerEnabled = false")
	}
	if cfg.MaxRetries != 4 {
		t.Fatalf("MaxRetries = %d", cfg.MaxRetries)
	}
}

func TestLoadRabbitMQConfigDefaults(t *testing.T) {
	t.Setenv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")

	cfg, err := LoadRabbitMQConfig()
	if err != nil {
		t.Fatalf("LoadRabbitMQConfig() error = %v", err)
	}

	if cfg.Exchange != "aquecedor.events" {
		t.Fatalf("Exchange = %q", cfg.Exchange)
	}
	if cfg.WarmingJobsQueue != "aquecedor.warming.jobs" {
		t.Fatalf("WarmingJobsQueue = %q", cfg.WarmingJobsQueue)
	}
	if cfg.EvolutionEventsQueue != "aquecedor.evolution.events" {
		t.Fatalf("EvolutionEventsQueue = %q", cfg.EvolutionEventsQueue)
	}
	if cfg.DeadLetterQueue != "aquecedor.dead_letter" {
		t.Fatalf("DeadLetterQueue = %q", cfg.DeadLetterQueue)
	}
	if cfg.Prefetch != 10 {
		t.Fatalf("Prefetch = %d", cfg.Prefetch)
	}
	if cfg.PublishTimeout != 5*time.Second {
		t.Fatalf("PublishTimeout = %s", cfg.PublishTimeout)
	}
	if !cfg.ConsumerEnabled {
		t.Fatal("ConsumerEnabled = false")
	}
	if cfg.MaxRetries != 3 {
		t.Fatalf("MaxRetries = %d", cfg.MaxRetries)
	}
}

func TestLoadRabbitMQConfigRequiresURL(t *testing.T) {
	t.Setenv("RABBITMQ_URL", "")

	if _, err := LoadRabbitMQConfig(); err == nil {
		t.Fatal("LoadRabbitMQConfig() error = nil, want error")
	}
}
