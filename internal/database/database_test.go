package database

import (
	"context"
	"testing"
	"time"

	"aquecedor-evolution/backend/internal/config"
)

func TestBuildPoolConfig(t *testing.T) {
	cfg, err := BuildPoolConfig(config.DatabaseConfig{
		URL:            "postgres://user:pass@localhost:5432/postgres?sslmode=require",
		MaxConns:       12,
		MinConns:       2,
		ConnectTimeout: 9 * time.Second,
	})
	if err != nil {
		t.Fatalf("BuildPoolConfig() error = %v", err)
	}

	if cfg.MaxConns != 12 {
		t.Fatalf("MaxConns = %d", cfg.MaxConns)
	}
	if cfg.MinConns != 2 {
		t.Fatalf("MinConns = %d", cfg.MinConns)
	}
	if cfg.ConnConfig.ConnectTimeout != 9*time.Second {
		t.Fatalf("ConnectTimeout = %s", cfg.ConnConfig.ConnectTimeout)
	}
}

func TestPingReturnsContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := Ping(ctx, nil); err == nil {
		t.Fatal("Ping() error = nil, want context error")
	}
}
