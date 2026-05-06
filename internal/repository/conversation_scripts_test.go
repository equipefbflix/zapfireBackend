package repository

import (
	"context"
	"strings"
	"testing"
)

func TestConversationScriptRepositoryCreate(t *testing.T) {
	db := &fakeExecutor{
		row: fakeRow{values: []any{
			"script-id",
			"conversa_basica_manha",
			"casual",
			true,
			10,
			0.0,
			40.0,
		}},
	}
	repo := NewConversationScriptRepository(db)

	script, err := repo.Create(context.Background(), CreateConversationScriptParams{
		Name:            "conversa_basica_manha",
		Category:        "casual",
		Enabled:         true,
		Weight:          10,
		MinWarmingScore: 0,
		MaxWarmingScore: 40,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if script.ID != "script-id" {
		t.Fatalf("ID = %q", script.ID)
	}
	if !strings.Contains(db.lastSQL, "insert into public.conversation_scripts") {
		t.Fatalf("sql = %s", db.lastSQL)
	}
}

func TestConversationScriptRepositoryList(t *testing.T) {
	db := &fakeExecutor{
		rows: fakeRows{values: [][]any{
			{"script-id", "conversa_basica_manha", "casual", true, 10, 0.0, 40.0},
		}},
	}
	repo := NewConversationScriptRepository(db)

	items, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items len = %d", len(items))
	}
	if items[0].Name != "conversa_basica_manha" {
		t.Fatalf("Name = %q", items[0].Name)
	}
	if !strings.Contains(db.lastSQL, "from public.conversation_scripts") {
		t.Fatalf("sql = %s", db.lastSQL)
	}
}

func TestConversationScriptRepositoryDeleteByName(t *testing.T) {
	db := &fakeExecutor{commandTag: CommandTag{RowsAffected: 1}}
	repo := NewConversationScriptRepository(db)

	deleted, err := repo.DeleteByName(context.Background(), "conversa_basica_manha")
	if err != nil {
		t.Fatalf("DeleteByName() error = %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d", deleted)
	}
	if !strings.Contains(db.lastSQL, "where name = $1") {
		t.Fatalf("sql = %s", db.lastSQL)
	}
}

func TestConversationStepRepositoryCreate(t *testing.T) {
	templateID := "template-id"
	db := &fakeExecutor{
		row: fakeRow{values: []any{
			"step-id",
			"script-id",
			1,
			"a",
			"send_text",
			templateID,
			[]byte(`{"text":"hello"}`),
			5,
			30,
		}},
	}
	repo := NewConversationStepRepository(db)

	step, err := repo.Create(context.Background(), CreateConversationStepParams{
		ScriptID:        "script-id",
		StepOrder:       1,
		SenderRole:      "a",
		ActionType:      "send_text",
		TemplateID:      &templateID,
		Payload:         map[string]any{"text": "hello"},
		MinDelaySeconds: 5,
		MaxDelaySeconds: 30,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if step.ID != "step-id" {
		t.Fatalf("ID = %q", step.ID)
	}
	if step.TemplateID == nil || *step.TemplateID != "template-id" {
		t.Fatalf("TemplateID = %v", step.TemplateID)
	}
	if !strings.Contains(db.lastSQL, "insert into public.conversation_steps") {
		t.Fatalf("sql = %s", db.lastSQL)
	}
}

func TestConversationStepRepositoryListByScriptID(t *testing.T) {
	db := &fakeExecutor{
		rows: fakeRows{values: [][]any{
			{"step-id", "script-id", 1, "a", "send_presence", nil, []byte(`{"presence":"composing"}`), 1, 3},
		}},
	}
	repo := NewConversationStepRepository(db)

	items, err := repo.ListByScriptID(context.Background(), "script-id")
	if err != nil {
		t.Fatalf("ListByScriptID() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items len = %d", len(items))
	}
	if items[0].ActionType != "send_presence" {
		t.Fatalf("ActionType = %q", items[0].ActionType)
	}
	if !strings.Contains(db.lastSQL, "where script_id = $1") {
		t.Fatalf("sql = %s", db.lastSQL)
	}
}
