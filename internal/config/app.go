package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const DefaultReadHeaderTimeout = 5 * time.Second

type AppConfig struct {
	Env                    string
	Port                   string
	PublicURL              string
	SupabaseURL            string
	AuthEnabled            bool
	WebhookEvolutionSecret string
	EvolutionTimeout       time.Duration
}

type EvolutionServerConfig struct {
	Name              string
	BaseURL           string
	APIKey            string
	Enabled           bool
	Weight            int
	MaxConcurrentJobs int
}

func LoadAppConfig() (AppConfig, error) {
	cfg := AppConfig{
		Env:                    envString("APP_ENV", "development"),
		Port:                   envString("APP_PORT", "8080"),
		PublicURL:              envString("APP_PUBLIC_URL", "http://localhost:8080"),
		SupabaseURL:            strings.TrimRight(envString("SUPABASE_URL", ""), "/"),
		AuthEnabled:            envBool("API_AUTH_ENABLED", false),
		WebhookEvolutionSecret: envString("WEBHOOK_EVOLUTION_SECRET", ""),
		EvolutionTimeout:       time.Duration(envInt("EVOLUTION_TIMEOUT_SECONDS", 30)) * time.Second,
	}

	if cfg.Port == "" {
		return AppConfig{}, fmt.Errorf("APP_PORT is required")
	}
	if cfg.EvolutionTimeout <= 0 {
		return AppConfig{}, fmt.Errorf("EVOLUTION_TIMEOUT_SECONDS must be greater than zero")
	}
	if cfg.AuthEnabled && cfg.SupabaseURL == "" {
		return AppConfig{}, fmt.Errorf("SUPABASE_URL is required when API_AUTH_ENABLED=true")
	}

	return cfg, nil
}

func LoadEvolutionServers() ([]EvolutionServerConfig, error) {
	rawServers := strings.TrimSpace(os.Getenv("EVOLUTION_SERVERS"))
	if rawServers == "" {
		return loadSingleEvolutionServerFallback()
	}

	keys := strings.Split(rawServers, ",")
	servers := make([]EvolutionServerConfig, 0, len(keys))
	for _, rawKey := range keys {
		key := strings.TrimSpace(rawKey)
		if key == "" {
			continue
		}

		prefix := "EVOLUTION_" + strings.ToUpper(key) + "_"
		name := envString(prefix+"NAME", key)
		baseURL := strings.TrimRight(envString(prefix+"BASE_URL", ""), "/")
		apiKey := envString(prefix+"API_KEY", "")
		if baseURL == "" {
			return nil, fmt.Errorf("%sBASE_URL is required", prefix)
		}
		if apiKey == "" {
			return nil, fmt.Errorf("%sAPI_KEY is required", prefix)
		}

		servers = append(servers, EvolutionServerConfig{
			Name:              name,
			BaseURL:           baseURL,
			APIKey:            apiKey,
			Enabled:           envBool(prefix+"ENABLED", true),
			Weight:            envInt(prefix+"WEIGHT", 1),
			MaxConcurrentJobs: envInt(prefix+"MAX_CONCURRENT_JOBS", 5),
		})
	}

	if len(servers) == 0 {
		return nil, fmt.Errorf("EVOLUTION_SERVERS did not contain any server keys")
	}

	return servers, nil
}

func loadSingleEvolutionServerFallback() ([]EvolutionServerConfig, error) {
	baseURL := strings.TrimRight(envString("SERVER_URL", ""), "/")
	apiKey := envString("AUTHENTICATION_API_KEY", "")
	if baseURL == "" || apiKey == "" {
		return nil, fmt.Errorf("EVOLUTION_SERVERS is required")
	}

	return []EvolutionServerConfig{
		{
			Name:              envString("EVOLUTION_DEFAULT_NAME", "default"),
			BaseURL:           baseURL,
			APIKey:            apiKey,
			Enabled:           true,
			Weight:            envInt("EVOLUTION_DEFAULT_WEIGHT", 1),
			MaxConcurrentJobs: envInt("EVOLUTION_DEFAULT_MAX_CONCURRENT_JOBS", 5),
		},
	}, nil
}
