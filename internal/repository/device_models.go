package repository

import (
	"context"
	"fmt"
)

type DeviceModel struct {
	ID           string
	Name         string
	OS           string
	SystemLabel  string
	VersionLabel string
	ImageURL     string
	SortOrder    int
	Enabled      bool
	Metadata     map[string]any
}

type CreateDeviceModelParams struct {
	Name         string
	OS           string
	SystemLabel  string
	VersionLabel string
	ImageURL     string
	SortOrder    int
	Enabled      bool
	Metadata     map[string]any
}

type UpdateDeviceModelParams struct {
	Name         *string
	OS           *string
	SystemLabel  *string
	VersionLabel *string
	ImageURL     *string
	SortOrder    *int
	Enabled      *bool
	Metadata     map[string]any
}

type DeviceModelRepository struct {
	db Executor
}

func NewDeviceModelRepository(db Executor) DeviceModelRepository {
	return DeviceModelRepository{db: db}
}

func (r DeviceModelRepository) Create(ctx context.Context, params CreateDeviceModelParams) (DeviceModel, error) {
	metadata, err := encodeMetadata(params.Metadata)
	if err != nil {
		return DeviceModel{}, err
	}

	row := r.db.QueryRow(ctx, `
insert into public.device_models (name, os, system_label, version_label, image_url, sort_order, enabled, metadata)
values ($1, $2, $3, $4, $5, $6, $7, $8::jsonb)
returning id::text, name, os, system_label, version_label, coalesce(image_url, ''), sort_order, enabled, metadata::text::bytea
`, params.Name, params.OS, params.SystemLabel, params.VersionLabel, params.ImageURL, params.SortOrder, params.Enabled, metadata)

	model, err := scanDeviceModel(row)
	if err != nil {
		return DeviceModel{}, fmt.Errorf("create device model: %w", err)
	}
	return model, nil
}

func (r DeviceModelRepository) List(ctx context.Context, includeDisabled bool) ([]DeviceModel, error) {
	sql := `
select id::text, name, os, system_label, version_label, coalesce(image_url, ''), sort_order, enabled, metadata::text::bytea
from public.device_models
`
	args := []any{}
	if !includeDisabled {
		sql += ` where enabled = true`
	}
	sql += ` order by sort_order asc, created_at asc`

	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("list device models: %w", err)
	}
	defer rows.Close()

	items := make([]DeviceModel, 0)
	for rows.Next() {
		item, err := scanDeviceModel(rows)
		if err != nil {
			return nil, fmt.Errorf("scan device model: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate device models: %w", err)
	}
	return items, nil
}

func (r DeviceModelRepository) ListEnabled(ctx context.Context) ([]DeviceModel, error) {
	return r.List(ctx, false)
}

func (r DeviceModelRepository) Update(ctx context.Context, id string, params UpdateDeviceModelParams) (DeviceModel, error) {
	metadata, err := encodeMetadata(params.Metadata)
	if err != nil {
		return DeviceModel{}, err
	}

	row := r.db.QueryRow(ctx, `
update public.device_models
set name = coalesce($2, name),
    os = coalesce($3, os),
    system_label = coalesce($4, system_label),
    version_label = coalesce($5, version_label),
    image_url = coalesce($6, image_url),
    sort_order = coalesce($7, sort_order),
    enabled = coalesce($8, enabled),
    metadata = case when $9::jsonb = '{}'::jsonb then metadata else $9::jsonb end,
    updated_at = now()
where id = $1::uuid
returning id::text, name, os, system_label, version_label, coalesce(image_url, ''), sort_order, enabled, metadata::text::bytea
`, id, params.Name, params.OS, params.SystemLabel, params.VersionLabel, params.ImageURL, params.SortOrder, params.Enabled, metadata)

	model, err := scanDeviceModel(row)
	if err != nil {
		return DeviceModel{}, fmt.Errorf("update device model: %w", err)
	}
	return model, nil
}

func scanDeviceModel(row Row) (DeviceModel, error) {
	var model DeviceModel
	var metadata []byte
	if err := row.Scan(
		&model.ID,
		&model.Name,
		&model.OS,
		&model.SystemLabel,
		&model.VersionLabel,
		&model.ImageURL,
		&model.SortOrder,
		&model.Enabled,
		&metadata,
	); err != nil {
		return DeviceModel{}, err
	}

	decoded, err := decodeMetadata(metadata)
	if err != nil {
		return DeviceModel{}, err
	}
	model.Metadata = decoded
	return model, nil
}
