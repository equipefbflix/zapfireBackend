package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"aquecedor-evolution/backend/internal/conversation"
	"aquecedor-evolution/backend/internal/repository"
)

type fakeHTTPConversationScriptStore struct {
	createParams conversation.CreateScriptParams
	updateParams conversation.UpdateScriptParams
	items        []conversation.ScriptWithSteps
}

func (s *fakeHTTPConversationScriptStore) Create(ctx context.Context, params conversation.CreateScriptParams) (conversation.ScriptWithSteps, error) {
	s.createParams = params
	script := repository.ConversationScript{
		ID:              "script-id",
		Name:            params.Name,
		Category:        params.Category,
		Enabled:         params.Enabled,
		Weight:          params.Weight,
		MinWarmingScore: params.MinWarmingScore,
		MaxWarmingScore: params.MaxWarmingScore,
	}
	steps := make([]repository.ConversationStep, 0, len(params.Steps))
	for _, step := range params.Steps {
		steps = append(steps, repository.ConversationStep{
			ID:              "step-id",
			ScriptID:        "script-id",
			StepOrder:       step.StepOrder,
			SenderRole:      step.SenderRole,
			ActionType:      step.ActionType,
			TemplateID:      step.TemplateID,
			Payload:         step.Payload,
			MinDelaySeconds: step.MinDelaySeconds,
			MaxDelaySeconds: step.MaxDelaySeconds,
		})
	}
	return conversation.ScriptWithSteps{Script: script, Steps: steps}, nil
}

func (s *fakeHTTPConversationScriptStore) List(ctx context.Context) ([]conversation.ScriptWithSteps, error) {
	return s.items, nil
}

func (s *fakeHTTPConversationScriptStore) GetByID(ctx context.Context, id string) (conversation.ScriptWithSteps, error) {
	if len(s.items) > 0 {
		return s.items[0], nil
	}
	return conversation.ScriptWithSteps{
		Script: repository.ConversationScript{
			ID:              id,
			Name:            "script-existente",
			Category:        "casual",
			Enabled:         true,
			Weight:          1,
			MinWarmingScore: 0,
			MaxWarmingScore: 100,
		},
		Steps: []repository.ConversationStep{},
	}, nil
}

func (s *fakeHTTPConversationScriptStore) Update(ctx context.Context, id string, params conversation.UpdateScriptParams) (conversation.ScriptWithSteps, error) {
	s.updateParams = params
	return conversation.ScriptWithSteps{
		Script: repository.ConversationScript{
			ID:              id,
			Name:            params.Name,
			Category:        params.Category,
			Enabled:         params.Enabled,
			Weight:          params.Weight,
			MinWarmingScore: params.MinWarmingScore,
			MaxWarmingScore: params.MaxWarmingScore,
		},
		Steps: []repository.ConversationStep{
			{
				ID:              "step-1",
				ScriptID:        id,
				StepOrder:       1,
				SenderRole:      "a",
				ActionType:      "send_text",
				Payload:         map[string]any{"text": "ola"},
				MinDelaySeconds: 1,
				MaxDelaySeconds: 3,
			},
		},
	}, nil
}

func TestCreateConversationScriptRoute(t *testing.T) {
	store := &fakeHTTPConversationScriptStore{}
	server := NewServer(ServerConfig{ConversationScripts: store})
	body := []byte(`{
		"name": "conversa_basica_manha",
		"category": "casual",
		"enabled": true,
		"weight": 10,
		"minWarmingScore": 0,
		"maxWarmingScore": 40,
		"steps": [
			{
				"stepOrder": 1,
				"senderRole": "a",
				"actionType": "send_presence",
				"payload": {"presence":"composing"},
				"minDelaySeconds": 1,
				"maxDelaySeconds": 3
			}
		]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversation-scripts", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if len(store.createParams.Steps) != 1 {
		t.Fatalf("steps len = %d", len(store.createParams.Steps))
	}
	if store.createParams.Steps[0].Payload["presence"] != "composing" {
		t.Fatalf("presence = %v", store.createParams.Steps[0].Payload["presence"])
	}

	var response conversationScriptResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.ID != "script-id" {
		t.Fatalf("ID = %q", response.ID)
	}
	if len(response.Steps) != 1 {
		t.Fatalf("steps len = %d", len(response.Steps))
	}
}

func TestCreateConversationScriptRouteRequiresSteps(t *testing.T) {
	server := NewServer(ServerConfig{ConversationScripts: &fakeHTTPConversationScriptStore{}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversation-scripts", bytes.NewReader([]byte(`{
		"name": "conversa_basica_manha",
		"category": "casual"
	}`)))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestListConversationScriptsRoute(t *testing.T) {
	server := NewServer(ServerConfig{ConversationScripts: &fakeHTTPConversationScriptStore{items: []conversation.ScriptWithSteps{
		{
			Script: repository.ConversationScript{
				ID:              "script-id",
				Name:            "conversa_basica_manha",
				Category:        "casual",
				Enabled:         true,
				Weight:          10,
				MinWarmingScore: 0,
				MaxWarmingScore: 40,
			},
			Steps: []repository.ConversationStep{
				{
					ID:              "step-id",
					ScriptID:        "script-id",
					StepOrder:       1,
					SenderRole:      "a",
					ActionType:      "send_presence",
					Payload:         map[string]any{"presence": "composing"},
					MinDelaySeconds: 1,
					MaxDelaySeconds: 3,
				},
			},
		},
	}}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversation-scripts", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	var response listConversationScriptsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(response.Items) != 1 {
		t.Fatalf("items len = %d", len(response.Items))
	}
	if len(response.Items[0].Steps) != 1 {
		t.Fatalf("steps len = %d", len(response.Items[0].Steps))
	}
}

func TestGetConversationScriptRoute(t *testing.T) {
	server := NewServer(ServerConfig{ConversationScripts: &fakeHTTPConversationScriptStore{items: []conversation.ScriptWithSteps{
		{
			Script: repository.ConversationScript{
				ID:              "script-id",
				Name:            "conversa_basica_manha",
				Category:        "casual",
				Enabled:         true,
				Weight:          10,
				MinWarmingScore: 0,
				MaxWarmingScore: 40,
			},
			Steps: []repository.ConversationStep{},
		},
	}}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversation-scripts/script-id", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateConversationScriptRoute(t *testing.T) {
	store := &fakeHTTPConversationScriptStore{}
	server := NewServer(ServerConfig{ConversationScripts: store})
	body := []byte(`{
		"name": "conversa_editada",
		"category": "reactive",
		"enabled": true,
		"weight": 5,
		"minWarmingScore": 0,
		"maxWarmingScore": 100,
		"steps": [
			{
				"stepOrder": 1,
				"senderRole": "a",
				"actionType": "send_text",
				"payload": {"text":"ola"},
				"minDelaySeconds": 1,
				"maxDelaySeconds": 3
			}
		]
	}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/conversation-scripts/script-id", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if store.updateParams.Name != "conversa_editada" {
		t.Fatalf("Name = %q", store.updateParams.Name)
	}
	if len(store.updateParams.Steps) != 1 {
		t.Fatalf("steps len = %d", len(store.updateParams.Steps))
	}
}
