package repository

import (
	"context"
	"encoding/json"
	"fmt"
)

type CommandTag struct {
	RowsAffected int64
}

type Row interface {
	Scan(dest ...any) error
}

type Rows interface {
	Close()
	Err() error
	Next() bool
	Scan(dest ...any) error
}

type Executor interface {
	QueryRow(ctx context.Context, sql string, args ...any) Row
	Query(ctx context.Context, sql string, args ...any) (Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (CommandTag, error)
}

type metadataBytes []byte

func encodeMetadata(metadata map[string]any) ([]byte, error) {
	if metadata == nil {
		metadata = map[string]any{}
	}
	data, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}
	return data, nil
}

func decodeMetadata(data []byte) (map[string]any, error) {
	if len(data) == 0 {
		return map[string]any{}, nil
	}

	var metadata map[string]any
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("decode metadata: %w", err)
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	return metadata, nil
}
