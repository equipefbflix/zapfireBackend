package repository

import (
	"context"
	"strings"
	"testing"
)

func TestPhoneNumberRepositoryCreate(t *testing.T) {
	db := &fakeExecutor{
		row: fakeRow{values: []any{
			"phone-id",
			"5511999999999",
			"chip teste",
			"new",
			0.0,
			[]byte(`{"testRunId":"test-run"}`),
		}},
	}
	repo := NewPhoneNumberRepository(db)

	phone, err := repo.Create(context.Background(), CreatePhoneNumberParams{
		PhoneE164: "5511999999999",
		Label:     "chip teste",
		Metadata: map[string]any{
			"testRunId": "test-run",
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if phone.ID != "phone-id" {
		t.Fatalf("ID = %q", phone.ID)
	}
	if phone.PhoneE164 != "5511999999999" {
		t.Fatalf("PhoneE164 = %q", phone.PhoneE164)
	}
	if phone.Metadata["testRunId"] != "test-run" {
		t.Fatalf("testRunId = %v", phone.Metadata["testRunId"])
	}
	if !strings.Contains(db.lastSQL, "insert into public.phone_numbers") {
		t.Fatalf("sql = %s", db.lastSQL)
	}
}

func TestPhoneNumberRepositoryDeleteByTestRunID(t *testing.T) {
	db := &fakeExecutor{commandTag: CommandTag{RowsAffected: 2}}
	repo := NewPhoneNumberRepository(db)

	deleted, err := repo.DeleteByTestRunID(context.Background(), "test-run")
	if err != nil {
		t.Fatalf("DeleteByTestRunID() error = %v", err)
	}

	if deleted != 2 {
		t.Fatalf("deleted = %d", deleted)
	}
	if !strings.Contains(db.lastSQL, "metadata ->> 'testRunId'") {
		t.Fatalf("sql = %s", db.lastSQL)
	}
}

func TestPhoneNumberRepositoryList(t *testing.T) {
	db := &fakeExecutor{
		rows: fakeRows{values: [][]any{
			{"phone-id", "5511999999999", "chip", "new", 0.0, []byte(`{}`)},
		}},
	}
	repo := NewPhoneNumberRepository(db)

	items, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items len = %d", len(items))
	}
	if items[0].ID != "phone-id" {
		t.Fatalf("ID = %q", items[0].ID)
	}
}

func TestPhoneNumberRepositoryUpdateWarmingState(t *testing.T) {
	db := &fakeExecutor{commandTag: CommandTag{RowsAffected: 1}}
	repo := NewPhoneNumberRepository(db)

	if err := repo.UpdateWarmingState(context.Background(), "phone-id", 82.5, "warm"); err != nil {
		t.Fatalf("UpdateWarmingState() error = %v", err)
	}
	if !strings.Contains(db.lastSQL, "update public.phone_numbers") {
		t.Fatalf("sql = %s", db.lastSQL)
	}
}

func TestPhoneNumberRepositoryUpdate(t *testing.T) {
	db := &fakeExecutor{
		row: fakeRow{values: []any{
			"phone-id",
			"5511999999999",
			"updated label",
			"active",
			10.5,
			[]byte(`{"updated":true}`),
		}},
	}
	repo := NewPhoneNumberRepository(db)

	phone, err := repo.Update(context.Background(), "phone-id", UpdatePhoneNumberParams{
		Label:  stringPointer("updated label"),
		Status: stringPointer("active"),
		Metadata: map[string]any{
			"updated": true,
		},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if phone.Label != "updated label" {
		t.Fatalf("Label = %q", phone.Label)
	}
	if phone.Status != "active" {
		t.Fatalf("Status = %q", phone.Status)
	}
	if !strings.Contains(db.lastSQL, "update public.phone_numbers") {
		t.Fatalf("sql = %s", db.lastSQL)
	}
}

func TestPhoneNumberRepositoryDelete(t *testing.T) {
	db := &fakeExecutor{commandTag: CommandTag{RowsAffected: 1}}
	repo := NewPhoneNumberRepository(db)

	if err := repo.Delete(context.Background(), "phone-id"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !strings.Contains(db.lastSQL, "delete from public.phone_numbers") {
		t.Fatalf("sql = %s", db.lastSQL)
	}
}

func stringPointer(s string) *string {
	return &s
}
