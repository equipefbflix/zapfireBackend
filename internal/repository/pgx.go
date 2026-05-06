package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type PgxExecutor struct {
	db pgxDB
}

type pgxDB interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func NewPgxExecutor(db pgxDB) PgxExecutor {
	return PgxExecutor{db: db}
}

func (e PgxExecutor) QueryRow(ctx context.Context, sql string, args ...any) Row {
	return e.db.QueryRow(ctx, sql, args...)
}

func (e PgxExecutor) Query(ctx context.Context, sql string, args ...any) (Rows, error) {
	return e.db.Query(ctx, sql, args...)
}

func (e PgxExecutor) Exec(ctx context.Context, sql string, args ...any) (CommandTag, error) {
	tag, err := e.db.Exec(ctx, sql, args...)
	if err != nil {
		return CommandTag{}, err
	}
	return CommandTag{RowsAffected: tag.RowsAffected()}, nil
}
