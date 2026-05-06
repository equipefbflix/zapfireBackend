package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestExecutionLogRepositoryCreate(t *testing.T) {
	jobID := "job-id"
	instanceID := "instance-id"
	actionType := "send_text"
	durationMs := 120
	createdAt := time.Date(2026, 5, 4, 15, 0, 0, 0, time.UTC)
	db := &fakeExecutor{
		row: fakeRow{values: []any{
			"log-id",
			jobID,
			instanceID,
			actionType,
			"success",
			[]byte(`{"text":"Bom dia"}`),
			[]byte(`{"messageId":"abc"}`),
			[]byte(`{"id":"abc"}`),
			"5511999999999@s.whatsapp.net",
			"",
			durationMs,
			createdAt,
		}},
	}
	repo := NewExecutionLogRepository(db)

	log, err := repo.Create(context.Background(), CreateExecutionLogParams{
		WarmingJobID:        &jobID,
		InstanceID:          &instanceID,
		ActionType:          &actionType,
		Status:              "success",
		RequestPayload:      map[string]any{"text": "Bom dia"},
		ResponsePayload:     map[string]any{"messageId": "abc"},
		EvolutionMessageKey: map[string]any{"id": "abc"},
		RemoteJID:           "5511999999999@s.whatsapp.net",
		Error:               "",
		DurationMs:          &durationMs,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if log.ID != "log-id" {
		t.Fatalf("ID = %q", log.ID)
	}
	if log.RequestPayload["text"] != "Bom dia" {
		t.Fatalf("RequestPayload text = %v", log.RequestPayload["text"])
	}
	if !strings.Contains(db.lastSQL, "insert into public.execution_logs") {
		t.Fatalf("sql = %s", db.lastSQL)
	}
}

func TestExecutionLogRepositoryList(t *testing.T) {
	createdAt := time.Date(2026, 5, 4, 15, 0, 0, 0, time.UTC)
	db := &fakeExecutor{
		rows: fakeRows{values: [][]any{
			{
				"log-id",
				nil,
				nil,
				"send_presence",
				"success",
				[]byte(`{"presence":"composing"}`),
				[]byte(`{}`),
				[]byte(`{}`),
				"",
				"",
				nil,
				createdAt,
			},
		}},
	}
	repo := NewExecutionLogRepository(db)

	items, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items len = %d", len(items))
	}
	if items[0].ActionType == nil || *items[0].ActionType != "send_presence" {
		t.Fatalf("ActionType = %v", items[0].ActionType)
	}
	if !strings.Contains(db.lastSQL, "from public.execution_logs") {
		t.Fatalf("sql = %s", db.lastSQL)
	}
}

func TestExecutionLogRepositoryExistsSuccessfulStep(t *testing.T) {
	db := &fakeExecutor{
		row: fakeRow{values: []any{true}},
	}
	repo := NewExecutionLogRepository(db)

	exists, err := repo.ExistsSuccessfulStep(context.Background(), "job-id", "step-id")
	if err != nil {
		t.Fatalf("ExistsSuccessfulStep() error = %v", err)
	}
	if !exists {
		t.Fatal("exists = false")
	}
	if db.lastArgs[0] != "job-id" {
		t.Fatalf("job id arg = %v", db.lastArgs[0])
	}
	if db.lastArgs[1] != "step-id" {
		t.Fatalf("step id arg = %v", db.lastArgs[1])
	}
	if !strings.Contains(db.lastSQL, "request_payload ->> 'stepId' = $2") {
		t.Fatalf("sql = %s", db.lastSQL)
	}
}

func TestExecutionLogRepositoryFindPhoneNumberIDByMessageID(t *testing.T) {
	db := &fakeExecutor{
		row: fakeRow{values: []any{"phone-id"}},
	}
	repo := NewExecutionLogRepository(db)

	phoneNumberID, err := repo.FindPhoneNumberIDByMessageID(context.Background(), "message-id")
	if err != nil {
		t.Fatalf("FindPhoneNumberIDByMessageID() error = %v", err)
	}
	if phoneNumberID != "phone-id" {
		t.Fatalf("phoneNumberID = %q", phoneNumberID)
	}
	if db.lastArgs[0] != "message-id" {
		t.Fatalf("message id arg = %v", db.lastArgs[0])
	}
}

func TestExecutionLogRepositoryFindPhoneNumberIDByMessageIDNotFound(t *testing.T) {
	db := &fakeExecutor{row: fakeRow{err: pgx.ErrNoRows}}
	repo := NewExecutionLogRepository(db)

	phoneNumberID, err := repo.FindPhoneNumberIDByMessageID(context.Background(), "missing-message-id")
	if err != nil {
		t.Fatalf("FindPhoneNumberIDByMessageID() error = %v", err)
	}
	if phoneNumberID != "" {
		t.Fatalf("phoneNumberID = %q", phoneNumberID)
	}
}

func TestExecutionLogRepositoryCountFailuresSince(t *testing.T) {
	db := &fakeExecutor{
		row: fakeRow{values: []any{5}},
	}
	repo := NewExecutionLogRepository(db)

	count, err := repo.CountFailuresSince(context.Background(), time.Date(2026, 5, 5, 15, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("CountFailuresSince() error = %v", err)
	}
	if count != 5 {
		t.Fatalf("count = %d", count)
	}
	if !strings.Contains(db.lastSQL, "status = 'failed'") {
		t.Fatalf("sql = %s", db.lastSQL)
	}
}
