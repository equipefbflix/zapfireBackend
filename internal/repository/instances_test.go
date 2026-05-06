package repository

import (
	"context"
	"strings"
	"testing"
)

func TestInstanceRepositoryCreate(t *testing.T) {
	db := &fakeExecutor{
		row: fakeRow{values: []any{
			"instance-id",
			"phone-id",
			"server-id",
			"proxy-id",
			"chip_5511999999999",
			"evo-instance-id",
			"INSTANCE_API_KEY_SECRET",
			"created",
			[]byte(`{"testRunId":"test-run"}`),
		}},
	}
	repo := NewInstanceRepository(db)

	instance, err := repo.Create(context.Background(), CreateInstanceParams{
		PhoneNumberID:            "phone-id",
		EvolutionServerID:        "server-id",
		ProxyID:                  stringPtr("proxy-id"),
		InstanceName:             "chip_5511999999999",
		EvolutionInstanceID:      stringPtr("evo-instance-id"),
		InstanceAPIKeySecretName: stringPtr("INSTANCE_API_KEY_SECRET"),
		Status:                   "created",
		Metadata:                 map[string]any{"testRunId": "test-run"},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if instance.ID != "instance-id" {
		t.Fatalf("ID = %q", instance.ID)
	}
	if instance.ProxyID == nil || *instance.ProxyID != "proxy-id" {
		t.Fatalf("ProxyID = %v", instance.ProxyID)
	}
	if instance.Metadata["testRunId"] != "test-run" {
		t.Fatalf("testRunId = %v", instance.Metadata["testRunId"])
	}
}

func TestInstanceRepositoryGetOpenByPhoneNumberID(t *testing.T) {
	db := &fakeExecutor{
		row: fakeRow{values: []any{
			"instance-id",
			"phone-id",
			"server-id",
			nil,
			"chip-phone",
			"evo-instance-id",
			"INSTANCE_API_KEY",
			"open",
			[]byte(`{"testRunId":"test-run"}`),
		}},
	}
	repo := NewInstanceRepository(db)

	instance, err := repo.GetOpenByPhoneNumberID(context.Background(), "phone-id")
	if err != nil {
		t.Fatalf("GetOpenByPhoneNumberID() error = %v", err)
	}
	if instance.ID != "instance-id" {
		t.Fatalf("ID = %q", instance.ID)
	}
	if db.lastArgs[0] != "phone-id" {
		t.Fatalf("phone id arg = %v", db.lastArgs[0])
	}
	if !strings.Contains(db.lastSQL, "status = 'open'") {
		t.Fatalf("sql = %s", db.lastSQL)
	}
}

func TestInstanceRepositoryDeleteByTestRunID(t *testing.T) {
	db := &fakeExecutor{commandTag: CommandTag{RowsAffected: 1}}
	repo := NewInstanceRepository(db)

	deleted, err := repo.DeleteByTestRunID(context.Background(), "test-run")
	if err != nil {
		t.Fatalf("DeleteByTestRunID() error = %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d", deleted)
	}
}

func TestInstanceRepositoryGetByInstanceName(t *testing.T) {
	db := &fakeExecutor{
		row: fakeRow{values: []any{
			"instance-id",
			"phone-id",
			"server-id",
			nil,
			"chip-phone",
			"evo-instance-id",
			"INSTANCE_API_KEY",
			"open",
			[]byte(`{"testRunId":"test-run"}`),
		}},
	}
	repo := NewInstanceRepository(db)

	instance, err := repo.GetByInstanceName(context.Background(), "chip-phone")
	if err != nil {
		t.Fatalf("GetByInstanceName() error = %v", err)
	}
	if instance.ID != "instance-id" {
		t.Fatalf("ID = %q", instance.ID)
	}
	if db.lastArgs[0] != "chip-phone" {
		t.Fatalf("instance name arg = %v", db.lastArgs[0])
	}
}
