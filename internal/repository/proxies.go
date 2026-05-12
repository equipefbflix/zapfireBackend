package repository

import (
	"context"
	"fmt"
)

type Proxy struct {
	ID                 string
	Name               string
	Host               string
	Port               int
	Protocol           string
	Username           *string
	PasswordSecretName *string
	Enabled            bool
	MaxInstances       *int
	CurrentInstances   int
	Metadata           map[string]any
}

type CreateProxyParams struct {
	Name               string
	Host               string
	Port               int
	Protocol           string
	Username           *string
	PasswordSecretName *string
	Enabled            bool
	MaxInstances       *int
	Metadata           map[string]any
}

type ProxyRepository struct {
	db Executor
}

func NewProxyRepository(db Executor) ProxyRepository {
	return ProxyRepository{db: db}
}

func (r ProxyRepository) Create(ctx context.Context, params CreateProxyParams) (Proxy, error) {
	metadata, err := encodeMetadata(params.Metadata)
	if err != nil {
		return Proxy{}, err
	}

	row := r.db.QueryRow(ctx, `
insert into public.proxies (name, host, port, protocol, username, password_secret_name, enabled, max_instances, metadata)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb)
returning id::text, name, host, port, protocol, username, password_secret_name, enabled, max_instances, current_instances, metadata::text::bytea
`, params.Name, params.Host, params.Port, params.Protocol, params.Username, params.PasswordSecretName, params.Enabled, params.MaxInstances, metadata)

	proxy, err := scanProxy(row)
	if err != nil {
		return Proxy{}, fmt.Errorf("create proxy: %w", err)
	}
	return proxy, nil
}

func (r ProxyRepository) Upsert(ctx context.Context, params CreateProxyParams) (Proxy, error) {
	metadata, err := encodeMetadata(params.Metadata)
	if err != nil {
		return Proxy{}, err
	}

	row := r.db.QueryRow(ctx, `
insert into public.proxies (name, host, port, protocol, username, password_secret_name, enabled, max_instances, metadata)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb)
on conflict (host, port) do update set
	name = excluded.name,
	protocol = excluded.protocol,
	username = excluded.username,
	password_secret_name = excluded.password_secret_name,
	enabled = excluded.enabled,
	max_instances = excluded.max_instances,
	metadata = public.proxies.metadata || excluded.metadata
returning id::text, name, host, port, protocol, username, password_secret_name, enabled, max_instances, current_instances, metadata::text::bytea
`, params.Name, params.Host, params.Port, params.Protocol, params.Username, params.PasswordSecretName, params.Enabled, params.MaxInstances, metadata)

	proxy, err := scanProxy(row)
	if err != nil {
		return Proxy{}, fmt.Errorf("upsert proxy: %w", err)
	}
	return proxy, nil
}

func (r ProxyRepository) ListEnabled(ctx context.Context) ([]Proxy, error) {
	rows, err := r.db.Query(ctx, `
select id::text, name, host, port, protocol, username, password_secret_name, enabled, max_instances, current_instances, metadata::text::bytea
from public.proxies
where enabled = true
order by current_instances asc, name asc
`)
	if err != nil {
		return nil, fmt.Errorf("list enabled proxies: %w", err)
	}
	defer rows.Close()

	var proxies []Proxy
	for rows.Next() {
		proxy, err := scanProxy(rows)
		if err != nil {
			return nil, fmt.Errorf("scan proxy: %w", err)
		}
		proxies = append(proxies, proxy)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate proxies: %w", err)
	}
	return proxies, nil
}

func (r ProxyRepository) List(ctx context.Context) ([]Proxy, error) {
	rows, err := r.db.Query(ctx, `
select id::text, name, host, port, protocol, username, password_secret_name, enabled, max_instances, current_instances, metadata::text::bytea
from public.proxies
order by name asc
`)
	if err != nil {
		return nil, fmt.Errorf("list proxies: %w", err)
	}
	defer rows.Close()

	var proxies []Proxy
	for rows.Next() {
		proxy, err := scanProxy(rows)
		if err != nil {
			return nil, fmt.Errorf("scan proxy: %w", err)
		}
		proxies = append(proxies, proxy)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate proxies: %w", err)
	}
	return proxies, nil
}

func (r ProxyRepository) DeleteByTestRunID(ctx context.Context, testRunID string) (int64, error) {
	tag, err := r.db.Exec(ctx, `
delete from public.proxies
where metadata ->> 'testRunId' = $1
`, testRunID)
	if err != nil {
		return 0, fmt.Errorf("delete proxies by testRunId: %w", err)
	}
	return tag.RowsAffected, nil
}

func scanProxy(row Row) (Proxy, error) {
	var proxy Proxy
	var metadata []byte
	if err := row.Scan(
		&proxy.ID,
		&proxy.Name,
		&proxy.Host,
		&proxy.Port,
		&proxy.Protocol,
		&proxy.Username,
		&proxy.PasswordSecretName,
		&proxy.Enabled,
		&proxy.MaxInstances,
		&proxy.CurrentInstances,
		&metadata,
	); err != nil {
		return Proxy{}, err
	}

	decoded, err := decodeMetadata(metadata)
	if err != nil {
		return Proxy{}, err
	}
	proxy.Metadata = decoded

	return proxy, nil
}
