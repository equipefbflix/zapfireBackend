package repository

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestWarmingJobRepositoryCreate(t *testing.T) {
	scriptID := "script-id"
	scheduledAt := time.Date(2026, 5, 4, 15, 0, 0, 0, time.UTC)
	db := &fakeExecutor{
		row: fakeRow{values: []any{
			"job-id",
			scriptID,
			"phone-a-id",
			"phone-b-id",
			"pending",
			scheduledAt,
			0,
			"",
			[]byte(`{"testRunId":"test-run"}`),
		}},
	}
	repo := NewWarmingJobRepository(db)

	job, err := repo.Create(context.Background(), CreateWarmingJobParams{
		ScriptID:    &scriptID,
		PhoneAID:    "phone-a-id",
		PhoneBID:    "phone-b-id",
		ScheduledAt: scheduledAt,
		Metadata:    map[string]any{"testRunId": "test-run"},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if job.ID != "job-id" {
		t.Fatalf("ID = %q", job.ID)
	}
	if job.ScriptID == nil || *job.ScriptID != "script-id" {
		t.Fatalf("ScriptID = %v", job.ScriptID)
	}
	if !strings.Contains(db.lastSQL, "insert into public.warming_jobs") {
		t.Fatalf("sql = %s", db.lastSQL)
	}
}

func TestWarmingJobRepositoryList(t *testing.T) {
	scriptID := "script-id"
	scheduledAt := time.Date(2026, 5, 4, 15, 0, 0, 0, time.UTC)
	db := &fakeExecutor{
		rows: fakeRows{values: [][]any{
			{"job-id", scriptID, "phone-a-id", "phone-b-id", "pending", scheduledAt, 0, "", []byte(`{"source":"manual"}`)},
		}},
	}
	repo := NewWarmingJobRepository(db)

	items, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items len = %d", len(items))
	}
	if items[0].Status != "pending" {
		t.Fatalf("Status = %q", items[0].Status)
	}
	if !strings.Contains(db.lastSQL, "from public.warming_jobs") {
		t.Fatalf("sql = %s", db.lastSQL)
	}
}

func TestWarmingJobRepositoryDeleteByTestRunID(t *testing.T) {
	db := &fakeExecutor{commandTag: CommandTag{RowsAffected: 1}}
	repo := NewWarmingJobRepository(db)

	deleted, err := repo.DeleteByTestRunID(context.Background(), "test-run")
	if err != nil {
		t.Fatalf("DeleteByTestRunID() error = %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d", deleted)
	}
	if !strings.Contains(db.lastSQL, "metadata ->> 'testRunId'") {
		t.Fatalf("sql = %s", db.lastSQL)
	}
}

func TestWarmingJobRepositoryListDuePending(t *testing.T) {
	now := time.Date(2026, 5, 4, 15, 0, 0, 0, time.UTC)
	db := &fakeExecutor{
		rows: fakeRows{values: [][]any{
			{"job-id", nil, "phone-a-id", "phone-b-id", "pending", now, 0, "", []byte(`{"testRunId":"test-run"}`)},
		}},
	}
	repo := NewWarmingJobRepository(db)

	items, err := repo.ListDuePending(context.Background(), now, 25)
	if err != nil {
		t.Fatalf("ListDuePending() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items len = %d", len(items))
	}
	if db.lastArgs[0] != now {
		t.Fatalf("now arg = %v", db.lastArgs[0])
	}
	if db.lastArgs[1] != 25 {
		t.Fatalf("limit arg = %v", db.lastArgs[1])
	}
	if !strings.Contains(db.lastSQL, "status = 'pending'") {
		t.Fatalf("sql = %s", db.lastSQL)
	}
	if !strings.Contains(db.lastSQL, "scheduled_at <= $1") {
		t.Fatalf("sql = %s", db.lastSQL)
	}
}

func TestWarmingJobRepositoryGetByID(t *testing.T) {
	scheduledAt := time.Date(2026, 5, 4, 15, 0, 0, 0, time.UTC)
	db := &fakeExecutor{
		row: fakeRow{values: []any{
			"job-id",
			nil,
			"phone-a-id",
			"phone-b-id",
			"pending",
			scheduledAt,
			0,
			"",
			[]byte(`{"testRunId":"test-run"}`),
		}},
	}
	repo := NewWarmingJobRepository(db)

	job, err := repo.GetByID(context.Background(), "job-id")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if job.ID != "job-id" {
		t.Fatalf("ID = %q", job.ID)
	}
	if db.lastArgs[0] != "job-id" {
		t.Fatalf("id arg = %v", db.lastArgs[0])
	}
	if !strings.Contains(db.lastSQL, "where id = $1") {
		t.Fatalf("sql = %s", db.lastSQL)
	}
}

func TestWarmingJobRepositoryUpdateStatus(t *testing.T) {
	db := &fakeExecutor{commandTag: CommandTag{RowsAffected: 1}}
	repo := NewWarmingJobRepository(db)

	if err := repo.UpdateStatus(context.Background(), "job-id", "running", ""); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	if !strings.Contains(db.lastSQL, "update public.warming_jobs") {
		t.Fatalf("sql = %s", db.lastSQL)
	}
}

func TestWarmingJobRepositoryCountRunningByPair(t *testing.T) {
	db := &fakeExecutor{row: fakeRow{values: []any{2}}}
	repo := NewWarmingJobRepository(db)

	count, err := repo.CountRunningByPair(context.Background(), "phone-a", "phone-b")
	if err != nil {
		t.Fatalf("CountRunningByPair() error = %v", err)
	}
	if count != 2 {
		t.Fatalf("count = %d", count)
	}
}

func TestWarmingJobRepositoryCountRunningByEvolutionServer(t *testing.T) {
	db := &fakeExecutor{row: fakeRow{values: []any{3}}}
	repo := NewWarmingJobRepository(db)

	count, err := repo.CountRunningByEvolutionServer(context.Background(), "server-id")
	if err != nil {
		t.Fatalf("CountRunningByEvolutionServer() error = %v", err)
	}
	if count != 3 {
		t.Fatalf("count = %d", count)
	}
}

func TestWarmingJobRepositoryCountByStatus(t *testing.T) {
	db := &fakeExecutor{
		rows: fakeRows{values: [][]any{
			{"pending", 2},
			{"running", 1},
		}},
	}
	repo := NewWarmingJobRepository(db)

	counts, err := repo.CountByStatus(context.Background())
	if err != nil {
		t.Fatalf("CountByStatus() error = %v", err)
	}
	if counts["pending"] != 2 {
		t.Fatalf("pending = %d", counts["pending"])
	}
	if counts["running"] != 1 {
		t.Fatalf("running = %d", counts["running"])
	}
}

func TestWarmingJobRepositoryCountStaleRunning(t *testing.T) {
	db := &fakeExecutor{row: fakeRow{values: []any{4}}}
	repo := NewWarmingJobRepository(db)

	count, err := repo.CountStaleRunning(context.Background(), time.Date(2026, 5, 5, 15, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("CountStaleRunning() error = %v", err)
	}
	if count != 4 {
		t.Fatalf("count = %d", count)
	}
}

func TestWarmingJobRepositoryFailStaleRunning(t *testing.T) {
	db := &fakeExecutor{commandTag: CommandTag{RowsAffected: 3}}
	repo := NewWarmingJobRepository(db)

	affected, err := repo.FailStaleRunning(context.Background(), time.Date(2026, 5, 5, 15, 0, 0, 0, time.UTC), "cleanup")
	if err != nil {
		t.Fatalf("FailStaleRunning() error = %v", err)
	}
	if affected != 3 {
		t.Fatalf("affected = %d", affected)
	}
	if !strings.Contains(db.lastSQL, "status = 'failed'::public.execution_status") {
		t.Fatalf("sql = %s", db.lastSQL)
	}
}
