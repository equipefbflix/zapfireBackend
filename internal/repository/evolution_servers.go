package repository

import (
	"context"
	"fmt"
)

type EvolutionServer struct {
	ID                string
	Name              string
	BaseURL           string
	APIKeySecretName  string
	Enabled           bool
	Weight            int
	MaxConcurrentJobs int
	HealthStatus      string
	Metadata          map[string]any
}

type CreateEvolutionServerParams struct {
	Name              string
	BaseURL           string
	APIKeySecretName  string
	Enabled           bool
	Weight            int
	MaxConcurrentJobs int
	Metadata          map[string]any
}

type EvolutionServerRepository struct {
	db Executor
}

func NewEvolutionServerRepository(db Executor) EvolutionServerRepository {
	return EvolutionServerRepository{db: db}
}

func (r EvolutionServerRepository) Create(ctx context.Context, params CreateEvolutionServerParams) (EvolutionServer, error) {
	metadata, err := encodeMetadata(params.Metadata)
	if err != nil {
		return EvolutionServer{}, err
	}

	row := r.db.QueryRow(ctx, `
insert into public.evolution_servers (name, base_url, api_key_secret_name, enabled, weight, max_concurrent_jobs, metadata)
values ($1, $2, $3, $4, $5, $6, $7::jsonb)
returning id::text, name, base_url, api_key_secret_name, enabled, weight, max_concurrent_jobs, health_status::text, metadata::text::bytea
`, params.Name, params.BaseURL, params.APIKeySecretName, params.Enabled, params.Weight, params.MaxConcurrentJobs, metadata)

	server, err := scanEvolutionServer(row)
	if err != nil {
		return EvolutionServer{}, fmt.Errorf("create evolution server: %w", err)
	}
	return server, nil
}

func (r EvolutionServerRepository) ListEnabled(ctx context.Context) ([]EvolutionServer, error) {
	rows, err := r.db.Query(ctx, `
select id::text, name, base_url, api_key_secret_name, enabled, weight, max_concurrent_jobs, health_status::text, metadata::text::bytea
from public.evolution_servers
where enabled = true
order by name
`)
	if err != nil {
		return nil, fmt.Errorf("list enabled evolution servers: %w", err)
	}
	defer rows.Close()

	var servers []EvolutionServer
	for rows.Next() {
		server, err := scanEvolutionServer(rows)
		if err != nil {
			return nil, fmt.Errorf("scan evolution server: %w", err)
		}
		servers = append(servers, server)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate evolution servers: %w", err)
	}
	return servers, nil
}

func (r EvolutionServerRepository) List(ctx context.Context) ([]EvolutionServer, error) {
	rows, err := r.db.Query(ctx, `
select id::text, name, base_url, api_key_secret_name, enabled, weight, max_concurrent_jobs, health_status::text, metadata::text::bytea
from public.evolution_servers
order by name
`)
	if err != nil {
		return nil, fmt.Errorf("list evolution servers: %w", err)
	}
	defer rows.Close()

	var servers []EvolutionServer
	for rows.Next() {
		server, err := scanEvolutionServer(rows)
		if err != nil {
			return nil, fmt.Errorf("scan evolution server: %w", err)
		}
		servers = append(servers, server)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate evolution servers: %w", err)
	}
	return servers, nil
}

func (r EvolutionServerRepository) GetByID(ctx context.Context, id string) (EvolutionServer, error) {
	row := r.db.QueryRow(ctx, `
select id::text, name, base_url, api_key_secret_name, enabled, weight, max_concurrent_jobs, health_status::text, metadata::text::bytea
from public.evolution_servers
where id = $1
`, id)

	server, err := scanEvolutionServer(row)
	if err != nil {
		return EvolutionServer{}, fmt.Errorf("get evolution server: %w", err)
	}
	return server, nil
}

func (r EvolutionServerRepository) DeleteByTestRunID(ctx context.Context, testRunID string) (int64, error) {
	tag, err := r.db.Exec(ctx, `
delete from public.evolution_servers
where metadata ->> 'testRunId' = $1
`, testRunID)
	if err != nil {
		return 0, fmt.Errorf("delete evolution servers by testRunId: %w", err)
	}
	return tag.RowsAffected, nil
}

func scanEvolutionServer(row Row) (EvolutionServer, error) {
	var server EvolutionServer
	var metadata []byte
	if err := row.Scan(
		&server.ID,
		&server.Name,
		&server.BaseURL,
		&server.APIKeySecretName,
		&server.Enabled,
		&server.Weight,
		&server.MaxConcurrentJobs,
		&server.HealthStatus,
		&metadata,
	); err != nil {
		return EvolutionServer{}, err
	}

	decoded, err := decodeMetadata(metadata)
	if err != nil {
		return EvolutionServer{}, err
	}
	server.Metadata = decoded

	return server, nil
}
