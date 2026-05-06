package config

import (
	"testing"
	"time"
)

func TestLoadAppConfigDefaults(t *testing.T) {
	cfg, err := LoadAppConfig()
	if err != nil {
		t.Fatalf("LoadAppConfig() error = %v", err)
	}

	if cfg.Env != "development" {
		t.Fatalf("Env = %q", cfg.Env)
	}
	if cfg.Port != "8080" {
		t.Fatalf("Port = %q", cfg.Port)
	}
	if cfg.PublicURL != "http://localhost:8080" {
		t.Fatalf("PublicURL = %q", cfg.PublicURL)
	}
	if cfg.SupabaseURL != "" {
		t.Fatalf("SupabaseURL = %q", cfg.SupabaseURL)
	}
	if cfg.AuthEnabled {
		t.Fatal("AuthEnabled = true")
	}
	if cfg.WebhookEvolutionSecret != "" {
		t.Fatalf("WebhookEvolutionSecret = %q", cfg.WebhookEvolutionSecret)
	}
	if cfg.EvolutionTimeout != 30*time.Second {
		t.Fatalf("EvolutionTimeout = %s", cfg.EvolutionTimeout)
	}
}

func TestLoadAppConfigWebhookSecret(t *testing.T) {
	t.Setenv("WEBHOOK_EVOLUTION_SECRET", "shared-secret")

	cfg, err := LoadAppConfig()
	if err != nil {
		t.Fatalf("LoadAppConfig() error = %v", err)
	}

	if cfg.WebhookEvolutionSecret != "shared-secret" {
		t.Fatalf("WebhookEvolutionSecret = %q", cfg.WebhookEvolutionSecret)
	}
}

func TestLoadAppConfigEvolutionTimeout(t *testing.T) {
	t.Setenv("EVOLUTION_TIMEOUT_SECONDS", "120")

	cfg, err := LoadAppConfig()
	if err != nil {
		t.Fatalf("LoadAppConfig() error = %v", err)
	}

	if cfg.EvolutionTimeout != 120*time.Second {
		t.Fatalf("EvolutionTimeout = %s", cfg.EvolutionTimeout)
	}
}

func TestLoadAppConfigAuth(t *testing.T) {
	t.Setenv("API_AUTH_ENABLED", "true")
	t.Setenv("SUPABASE_URL", "https://project-ref.supabase.co/")

	cfg, err := LoadAppConfig()
	if err != nil {
		t.Fatalf("LoadAppConfig() error = %v", err)
	}

	if !cfg.AuthEnabled {
		t.Fatal("AuthEnabled = false")
	}
	if cfg.SupabaseURL != "https://project-ref.supabase.co" {
		t.Fatalf("SupabaseURL = %q", cfg.SupabaseURL)
	}
}

func TestLoadAppConfigAuthRequiresSupabaseURL(t *testing.T) {
	t.Setenv("API_AUTH_ENABLED", "true")

	if _, err := LoadAppConfig(); err == nil {
		t.Fatal("LoadAppConfig() error = nil, want error")
	}
}

func TestLoadEvolutionServers(t *testing.T) {
	t.Setenv("EVOLUTION_SERVERS", "evo1,evo2")
	t.Setenv("EVOLUTION_EVO1_NAME", "primary")
	t.Setenv("EVOLUTION_EVO1_BASE_URL", "https://evo1.example.com/")
	t.Setenv("EVOLUTION_EVO1_API_KEY", "key-1")
	t.Setenv("EVOLUTION_EVO1_ENABLED", "true")
	t.Setenv("EVOLUTION_EVO1_WEIGHT", "2")
	t.Setenv("EVOLUTION_EVO1_MAX_CONCURRENT_JOBS", "7")
	t.Setenv("EVOLUTION_EVO2_BASE_URL", "https://evo2.example.com")
	t.Setenv("EVOLUTION_EVO2_API_KEY", "key-2")
	t.Setenv("EVOLUTION_EVO2_ENABLED", "false")

	servers, err := LoadEvolutionServers()
	if err != nil {
		t.Fatalf("LoadEvolutionServers() error = %v", err)
	}

	if len(servers) != 2 {
		t.Fatalf("servers len = %d", len(servers))
	}
	if servers[0].Name != "primary" {
		t.Fatalf("server 0 name = %q", servers[0].Name)
	}
	if servers[0].BaseURL != "https://evo1.example.com" {
		t.Fatalf("server 0 base url = %q", servers[0].BaseURL)
	}
	if servers[0].APIKey != "key-1" {
		t.Fatalf("server 0 api key = %q", servers[0].APIKey)
	}
	if servers[0].Weight != 2 {
		t.Fatalf("server 0 weight = %d", servers[0].Weight)
	}
	if servers[0].MaxConcurrentJobs != 7 {
		t.Fatalf("server 0 max concurrent = %d", servers[0].MaxConcurrentJobs)
	}
	if servers[1].Name != "evo2" {
		t.Fatalf("server 1 name = %q", servers[1].Name)
	}
	if servers[1].Enabled {
		t.Fatal("server 1 enabled = true")
	}
}

func TestLoadEvolutionServersRequiresConfiguredServers(t *testing.T) {
	t.Setenv("EVOLUTION_SERVERS", "")

	if _, err := LoadEvolutionServers(); err == nil {
		t.Fatal("LoadEvolutionServers() error = nil, want error")
	}
}

func TestLoadEvolutionServersFromSingleEvolutionEnvFallback(t *testing.T) {
	t.Setenv("EVOLUTION_SERVERS", "")
	t.Setenv("SERVER_URL", "https://evo.askgeni.us/")
	t.Setenv("AUTHENTICATION_API_KEY", "secret")

	servers, err := LoadEvolutionServers()
	if err != nil {
		t.Fatalf("LoadEvolutionServers() error = %v", err)
	}

	if len(servers) != 1 {
		t.Fatalf("servers len = %d", len(servers))
	}
	if servers[0].Name != "default" {
		t.Fatalf("name = %q", servers[0].Name)
	}
	if servers[0].BaseURL != "https://evo.askgeni.us" {
		t.Fatalf("base url = %q", servers[0].BaseURL)
	}
	if servers[0].APIKey != "secret" {
		t.Fatalf("api key = %q", servers[0].APIKey)
	}
	if servers[0].MaxConcurrentJobs != 5 {
		t.Fatalf("max concurrent = %d", servers[0].MaxConcurrentJobs)
	}
}
