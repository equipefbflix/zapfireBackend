package repository

import (
	"context"
	"fmt"
)

type Instance struct {
	ID                       string
	PhoneNumberID            string
	EvolutionServerID        string
	ProxyID                  *string
	InstanceName             string
	EvolutionInstanceID      *string
	InstanceAPIKeySecretName *string
	Status                   string
	Metadata                 map[string]any
}

type CreateInstanceParams struct {
	PhoneNumberID            string
	EvolutionServerID        string
	ProxyID                  *string
	InstanceName             string
	EvolutionInstanceID      *string
	InstanceAPIKeySecretName *string
	Status                   string
	Metadata                 map[string]any
}

type InstanceRepository struct {
	db Executor
}

func NewInstanceRepository(db Executor) InstanceRepository {
	return InstanceRepository{db: db}
}

func (r InstanceRepository) Create(ctx context.Context, params CreateInstanceParams) (Instance, error) {
	metadata, err := encodeMetadata(params.Metadata)
	if err != nil {
		return Instance{}, err
	}

	status := params.Status
	if status == "" {
		status = "created"
	}

	row := r.db.QueryRow(ctx, `
insert into public.instances (
  phone_number_id,
  evolution_server_id,
  proxy_id,
  instance_name,
  evolution_instance_id,
  instance_api_key_secret_name,
  status,
  metadata
)
values ($1, $2, $3, $4, $5, $6, $7::public.instance_status, $8::jsonb)
returning id::text, phone_number_id::text, evolution_server_id::text, proxy_id::text, instance_name, evolution_instance_id, instance_api_key_secret_name, status::text, metadata::text::bytea
`, params.PhoneNumberID, params.EvolutionServerID, params.ProxyID, params.InstanceName, params.EvolutionInstanceID, params.InstanceAPIKeySecretName, status, metadata)

	instance, err := scanInstance(row)
	if err != nil {
		return Instance{}, fmt.Errorf("create instance: %w", err)
	}
	return instance, nil
}

func (r InstanceRepository) GetOpenByPhoneNumberID(ctx context.Context, phoneNumberID string) (Instance, error) {
	row := r.db.QueryRow(ctx, `
select id::text, phone_number_id::text, evolution_server_id::text, proxy_id::text, instance_name, evolution_instance_id, instance_api_key_secret_name, status::text, metadata::text::bytea
from public.instances
where phone_number_id = $1
  and status = 'open'
order by last_connected_at desc nulls last, created_at desc
limit 1
`, phoneNumberID)

	instance, err := scanInstance(row)
	if err != nil {
		return Instance{}, fmt.Errorf("get open instance by phone number: %w", err)
	}
	return instance, nil
}

func (r InstanceRepository) GetByInstanceName(ctx context.Context, instanceName string) (Instance, error) {
	row := r.db.QueryRow(ctx, `
select id::text, phone_number_id::text, evolution_server_id::text, proxy_id::text, instance_name, evolution_instance_id, instance_api_key_secret_name, status::text, metadata::text::bytea
from public.instances
where instance_name = $1
`, instanceName)

	instance, err := scanInstance(row)
	if err != nil {
		return Instance{}, fmt.Errorf("get instance by name: %w", err)
	}
	return instance, nil
}

func (r InstanceRepository) DeleteByTestRunID(ctx context.Context, testRunID string) (int64, error) {
	tag, err := r.db.Exec(ctx, `
delete from public.instances
where metadata ->> 'testRunId' = $1
`, testRunID)
	if err != nil {
		return 0, fmt.Errorf("delete instances by testRunId: %w", err)
	}
	return tag.RowsAffected, nil
}

func (r InstanceRepository) UpdateConnectionStateByName(ctx context.Context, instanceName string, status string, lastError string) error {
	sql := `
update public.instances
set status = $2::public.instance_status,
    last_error = nullif($3, ''),
    last_connection_check_at = now(),
    updated_at = now(),
    last_connected_at = case when $2 = 'open' then now() else last_connected_at end,
    last_disconnected_at = case when $2 = 'close' then now() else last_disconnected_at end
where instance_name = $1
`
	_, err := r.db.Exec(ctx, sql, instanceName, status, lastError)
	if err != nil {
		return fmt.Errorf("update instance connection state: %w", err)
	}
	return nil
}

func scanInstance(row Row) (Instance, error) {
	var instance Instance
	var metadata []byte
	if err := row.Scan(
		&instance.ID,
		&instance.PhoneNumberID,
		&instance.EvolutionServerID,
		&instance.ProxyID,
		&instance.InstanceName,
		&instance.EvolutionInstanceID,
		&instance.InstanceAPIKeySecretName,
		&instance.Status,
		&metadata,
	); err != nil {
		return Instance{}, err
	}

	decoded, err := decodeMetadata(metadata)
	if err != nil {
		return Instance{}, err
	}
	instance.Metadata = decoded

	return instance, nil
}
