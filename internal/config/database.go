package config

import (
	"fmt"
	"strings"
	"time"
)

type DatabaseConfig struct {
	URL            string
	MaxConns       int
	MinConns       int
	ConnectTimeout time.Duration
}

func LoadDatabaseConfig() (DatabaseConfig, error) {
	cfg := DatabaseConfig{
		URL:            strings.TrimSpace(envString("DATABASE_URL", "")),
		MaxConns:       envInt("DATABASE_MAX_CONNS", 10),
		MinConns:       envInt("DATABASE_MIN_CONNS", 1),
		ConnectTimeout: time.Duration(envInt("DATABASE_CONNECT_TIMEOUT_SECONDS", 10)) * time.Second,
	}

	if cfg.URL == "" {
		return DatabaseConfig{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.MaxConns <= 0 {
		return DatabaseConfig{}, fmt.Errorf("DATABASE_MAX_CONNS must be greater than zero")
	}
	if cfg.MinConns < 0 {
		return DatabaseConfig{}, fmt.Errorf("DATABASE_MIN_CONNS must be greater than or equal to zero")
	}
	if cfg.MinConns > cfg.MaxConns {
		return DatabaseConfig{}, fmt.Errorf("DATABASE_MIN_CONNS must be less than or equal to DATABASE_MAX_CONNS")
	}
	if cfg.ConnectTimeout <= 0 {
		return DatabaseConfig{}, fmt.Errorf("DATABASE_CONNECT_TIMEOUT_SECONDS must be greater than zero")
	}

	return cfg, nil
}
