package repository

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestWarmingJobRepositoryListRecentByPair(t *testing.T) {
	scriptID := "script-id"
	db := &fakeExecutor{
		rows: fakeRows{values: [][]any{
			{"job-id", scriptID, "phone-a", "phone-b", "pending", time.Now().UTC(), 0, "", []byte(`{}`)},
		}},
	}
	repo := NewWarmingJobRepository(db)

	items, err := repo.ListRecentByPair(context.Background(), "phone-a", "phone-b", time.Now().UTC().Add(-time.Hour))
	if err != nil {
		t.Fatalf("ListRecentByPair() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items len = %d", len(items))
	}
	if !strings.Contains(db.lastSQL, "phone_a_id = $1 and phone_b_id = $2") {
		t.Fatalf("sql = %s", db.lastSQL)
	}
}
