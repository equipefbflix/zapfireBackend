package database

import (
	"context"
	"fmt"

	"aquecedor-evolution/backend/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

func BuildPoolConfig(cfg config.DatabaseConfig) (*pgxpool.Config, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.MaxConns)
	poolConfig.MinConns = int32(cfg.MinConns)
	poolConfig.ConnConfig.ConnectTimeout = cfg.ConnectTimeout

	return poolConfig, nil
}

func Open(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	poolConfig, err := BuildPoolConfig(cfg)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("open database pool: %w", err)
	}

	if err := Ping(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

func Ping(ctx context.Context, pinger Pinger) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if pinger == nil {
		return fmt.Errorf("database pinger is nil")
	}
	if err := pinger.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	return nil
}
