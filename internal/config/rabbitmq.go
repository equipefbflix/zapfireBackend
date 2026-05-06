package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type RabbitMQConfig struct {
	URL                  string
	Exchange             string
	WarmingJobsQueue     string
	EvolutionEventsQueue string
	DeadLetterQueue      string
	Prefetch             int
	MaxRetries           int
	PublishTimeout       time.Duration
	ConsumerEnabled      bool
}

func LoadRabbitMQConfig() (RabbitMQConfig, error) {
	cfg := RabbitMQConfig{
		URL:                  strings.TrimSpace(os.Getenv("RABBITMQ_URL")),
		Exchange:             envString("RABBITMQ_EXCHANGE", "aquecedor.events"),
		WarmingJobsQueue:     envString("RABBITMQ_QUEUE_WARMING_JOBS", "aquecedor.warming.jobs"),
		EvolutionEventsQueue: envString("RABBITMQ_QUEUE_EVOLUTION_EVENTS", "aquecedor.evolution.events"),
		DeadLetterQueue:      envString("RABBITMQ_QUEUE_DEAD_LETTER", "aquecedor.dead_letter"),
		Prefetch:             envInt("RABBITMQ_PREFETCH", 10),
		MaxRetries:           envInt("RABBITMQ_MAX_RETRIES", 3),
		PublishTimeout:       time.Duration(envInt("RABBITMQ_PUBLISH_TIMEOUT_SECONDS", 5)) * time.Second,
		ConsumerEnabled:      envBool("RABBITMQ_CONSUMER_ENABLED", true),
	}

	if cfg.URL == "" {
		return RabbitMQConfig{}, fmt.Errorf("RABBITMQ_URL is required")
	}
	if cfg.Prefetch <= 0 {
		return RabbitMQConfig{}, fmt.Errorf("RABBITMQ_PREFETCH must be greater than zero")
	}
	if cfg.MaxRetries < 0 {
		return RabbitMQConfig{}, fmt.Errorf("RABBITMQ_MAX_RETRIES must be greater than or equal to zero")
	}
	if cfg.PublishTimeout <= 0 {
		return RabbitMQConfig{}, fmt.Errorf("RABBITMQ_PUBLISH_TIMEOUT_SECONDS must be greater than zero")
	}

	return cfg, nil
}

func envString(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
