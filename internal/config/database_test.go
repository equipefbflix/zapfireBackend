package config

import (
	"testing"
	"time"
)

func TestLoadDatabaseConfig(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/postgres?sslmode=require")
	t.Setenv("DATABASE_MAX_CONNS", "12")
	t.Setenv("DATABASE_MIN_CONNS", "2")
	t.Setenv("DATABASE_CONNECT_TIMEOUT_SECONDS", "9")

	cfg, err := LoadDatabaseConfig()
	if err != nil {
		t.Fatalf("LoadDatabaseConfig() error = %v", err)
	}

	if cfg.URL != "postgres://user:pass@localhost:5432/postgres?sslmode=require" {
		t.Fatalf("URL = %q", cfg.URL)
	}
	if cfg.MaxConns != 12 {
		t.Fatalf("MaxConns = %d", cfg.MaxConns)
	}
	if cfg.MinConns != 2 {
		t.Fatalf("MinConns = %d", cfg.MinConns)
	}
	if cfg.ConnectTimeout != 9*time.Second {
		t.Fatalf("ConnectTimeout = %s", cfg.ConnectTimeout)
	}
}

func TestLoadDatabaseConfigDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/postgres?sslmode=require")

	cfg, err := LoadDatabaseConfig()
	if err != nil {
		t.Fatalf("LoadDatabaseConfig() error = %v", err)
	}

	if cfg.MaxConns != 10 {
		t.Fatalf("MaxConns = %d", cfg.MaxConns)
	}
	if cfg.MinConns != 1 {
		t.Fatalf("MinConns = %d", cfg.MinConns)
	}
	if cfg.ConnectTimeout != 10*time.Second {
		t.Fatalf("ConnectTimeout = %s", cfg.ConnectTimeout)
	}
}

func TestLoadDatabaseConfigRequiresURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	if _, err := LoadDatabaseConfig(); err == nil {
		t.Fatal("LoadDatabaseConfig() error = nil, want error")
	}
}
