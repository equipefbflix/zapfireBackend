package repository

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type fakeExecutor struct {
	row         fakeRow
	rows        fakeRows
	commandTag  CommandTag
	lastSQL     string
	lastArgs    []any
	queryRowErr error
	queryErr    error
	execErr     error
}

func (f *fakeExecutor) QueryRow(ctx context.Context, sql string, args ...any) Row {
	f.lastSQL = sql
	f.lastArgs = args
	if f.queryRowErr != nil {
		return fakeRow{err: f.queryRowErr}
	}
	return f.row
}

func (f *fakeExecutor) Query(ctx context.Context, sql string, args ...any) (Rows, error) {
	f.lastSQL = sql
	f.lastArgs = args
	if f.queryErr != nil {
		return nil, f.queryErr
	}
	return &f.rows, nil
}

func (f *fakeExecutor) Exec(ctx context.Context, sql string, args ...any) (CommandTag, error) {
	f.lastSQL = sql
	f.lastArgs = args
	if f.execErr != nil {
		return CommandTag{}, f.execErr
	}
	return f.commandTag, nil
}

type fakeRow struct {
	values []any
	err    error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.values) {
		return errors.New("scan destination length mismatch")
	}
	for i := range dest {
		assign(dest[i], r.values[i])
	}
	return nil
}

type fakeRows struct {
	values [][]any
	index  int
	err    error
}

func (r *fakeRows) Close() {}

func (r *fakeRows) Err() error {
	return r.err
}

func (r *fakeRows) Next() bool {
	return r.index < len(r.values)
}

func (r *fakeRows) Scan(dest ...any) error {
	if r.index >= len(r.values) {
		return errors.New("no current row")
	}
	values := r.values[r.index]
	r.index++
	if len(dest) != len(values) {
		return errors.New("scan destination length mismatch")
	}
	for i := range dest {
		assign(dest[i], values[i])
	}
	return nil
}

func assign(dest any, value any) {
	switch d := dest.(type) {
	case *string:
		*d = value.(string)
	case **string:
		if value == nil {
			*d = nil
			return
		}
		v := value.(string)
		*d = &v
	case *bool:
		*d = value.(bool)
	case *int:
		*d = value.(int)
	case **int:
		if value == nil {
			*d = nil
			return
		}
		v := value.(int)
		*d = &v
	case *float64:
		*d = value.(float64)
	case *time.Time:
		*d = value.(time.Time)
	case *map[string]any:
		*d = value.(map[string]any)
	case *[]byte:
		*d = value.([]byte)
	default:
		panic(fmt.Sprintf("unsupported scan destination %T", dest))
	}
}
