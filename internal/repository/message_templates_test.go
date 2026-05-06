package repository

import (
	"context"
	"strings"
	"testing"
)

func TestMessageTemplateRepositoryCreate(t *testing.T) {
	db := &fakeExecutor{
		row: fakeRow{values: []any{
			"template-id",
			"casual",
			"bom dia simples",
			"Bom dia, tudo certo por ai?",
			10,
			true,
			0.0,
			40.0,
			[]byte(`{"testRunId":"test-run"}`),
		}},
	}
	repo := NewMessageTemplateRepository(db)

	template, err := repo.Create(context.Background(), CreateMessageTemplateParams{
		Category:        "casual",
		Title:           "bom dia simples",
		Body:            "Bom dia, tudo certo por ai?",
		Weight:          10,
		Enabled:         true,
		MinWarmingScore: 0,
		MaxWarmingScore: 40,
		Metadata:        map[string]any{"testRunId": "test-run"},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if template.ID != "template-id" {
		t.Fatalf("ID = %q", template.ID)
	}
	if template.MaxWarmingScore != 40 {
		t.Fatalf("MaxWarmingScore = %f", template.MaxWarmingScore)
	}
	if !strings.Contains(db.lastSQL, "insert into public.message_templates") {
		t.Fatalf("sql = %s", db.lastSQL)
	}
}

func TestMessageTemplateRepositoryList(t *testing.T) {
	db := &fakeExecutor{
		rows: fakeRows{values: [][]any{
			{
				"template-id",
				"casual",
				"bom dia simples",
				"Bom dia, tudo certo por ai?",
				10,
				true,
				0.0,
				40.0,
				[]byte(`{"tone":"friendly"}`),
			},
		}},
	}
	repo := NewMessageTemplateRepository(db)

	items, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items len = %d", len(items))
	}
	if items[0].Category != "casual" {
		t.Fatalf("Category = %q", items[0].Category)
	}
	if !strings.Contains(db.lastSQL, "from public.message_templates") {
		t.Fatalf("sql = %s", db.lastSQL)
	}
}

func TestMessageTemplateRepositoryDeleteByTestRunID(t *testing.T) {
	db := &fakeExecutor{commandTag: CommandTag{RowsAffected: 2}}
	repo := NewMessageTemplateRepository(db)

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
