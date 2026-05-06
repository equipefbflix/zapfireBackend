package repository

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestEvolutionEventRepositoryCountSince(t *testing.T) {
	db := &fakeExecutor{
		row: fakeRow{values: []any{7}},
	}
	repo := NewEvolutionEventRepository(db)

	count, err := repo.CountSince(context.Background(), time.Date(2026, 5, 5, 15, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("CountSince() error = %v", err)
	}
	if count != 7 {
		t.Fatalf("count = %d", count)
	}
	if !strings.Contains(db.lastSQL, "from public.evolution_events") {
		t.Fatalf("sql = %s", db.lastSQL)
	}
}
