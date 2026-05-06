package httpserver

import (
	"encoding/json"
	"net/http"

	"aquecedor-evolution/backend/internal/conversation"
	"aquecedor-evolution/backend/internal/repository"
)

type createConversationScriptRequest struct {
	Name            string                          `json:"name"`
	Category        string                          `json:"category"`
	Enabled         bool                            `json:"enabled"`
	Weight          int                             `json:"weight"`
	MinWarmingScore float64                         `json:"minWarmingScore"`
	MaxWarmingScore float64                         `json:"maxWarmingScore"`
	Steps           []createConversationStepRequest `json:"steps"`
}

type createConversationStepRequest struct {
	StepOrder       int            `json:"stepOrder"`
	SenderRole      string         `json:"senderRole"`
	ActionType      string         `json:"actionType"`
	TemplateID      *string        `json:"templateId,omitempty"`
	Payload         map[string]any `json:"payload,omitempty"`
	MinDelaySeconds int            `json:"minDelaySeconds"`
	MaxDelaySeconds int            `json:"maxDelaySeconds"`
}

type conversationScriptResponse struct {
	ID              string                     `json:"id"`
	Name            string                     `json:"name"`
	Category        string                     `json:"category"`
	Enabled         bool                       `json:"enabled"`
	Weight          int                        `json:"weight"`
	MinWarmingScore float64                    `json:"minWarmingScore"`
	MaxWarmingScore float64                    `json:"maxWarmingScore"`
	Steps           []conversationStepResponse `json:"steps"`
}

type conversationStepResponse struct {
	ID              string         `json:"id"`
	ScriptID        string         `json:"scriptId"`
	StepOrder       int            `json:"stepOrder"`
	SenderRole      string         `json:"senderRole"`
	ActionType      string         `json:"actionType"`
	TemplateID      *string        `json:"templateId,omitempty"`
	Payload         map[string]any `json:"payload"`
	MinDelaySeconds int            `json:"minDelaySeconds"`
	MaxDelaySeconds int            `json:"maxDelaySeconds"`
}

type listConversationScriptsResponse struct {
	Items []conversationScriptResponse `json:"items"`
}

func (s *Server) handleCreateConversationScript(w http.ResponseWriter, r *http.Request) {
	if s.cfg.ConversationScripts == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "conversation script store is not configured"})
		return
	}

	var request createConversationScriptRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json body"})
		return
	}
	if request.Name == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "name is required"})
		return
	}
	if request.Category == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "category is required"})
		return
	}
	if len(request.Steps) == 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "steps are required"})
		return
	}

	steps := make([]conversation.CreateStepParams, 0, len(request.Steps))
	for _, step := range request.Steps {
		if step.StepOrder <= 0 {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "stepOrder must be greater than zero"})
			return
		}
		if step.SenderRole == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "senderRole is required"})
			return
		}
		if step.ActionType == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "actionType is required"})
			return
		}
		payload := step.Payload
		if payload == nil {
			payload = map[string]any{}
		}
		steps = append(steps, conversation.CreateStepParams{
			StepOrder:       step.StepOrder,
			SenderRole:      step.SenderRole,
			ActionType:      step.ActionType,
			TemplateID:      step.TemplateID,
			Payload:         payload,
			MinDelaySeconds: step.MinDelaySeconds,
			MaxDelaySeconds: step.MaxDelaySeconds,
		})
	}

	created, err := s.cfg.ConversationScripts.Create(r.Context(), conversation.CreateScriptParams{
		Name:            request.Name,
		Category:        request.Category,
		Enabled:         request.Enabled,
		Weight:          request.Weight,
		MinWarmingScore: request.MinWarmingScore,
		MaxWarmingScore: request.MaxWarmingScore,
		Steps:           steps,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to create conversation script"})
		return
	}

	writeJSON(w, http.StatusCreated, newConversationScriptResponse(created))
}

func (s *Server) handleListConversationScripts(w http.ResponseWriter, r *http.Request) {
	if s.cfg.ConversationScripts == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "conversation script store is not configured"})
		return
	}

	items, err := s.cfg.ConversationScripts.List(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to list conversation scripts"})
		return
	}

	response := listConversationScriptsResponse{
		Items: make([]conversationScriptResponse, 0, len(items)),
	}
	for _, item := range items {
		response.Items = append(response.Items, newConversationScriptResponse(item))
	}
	writeJSON(w, http.StatusOK, response)
}

func newConversationScriptResponse(item conversation.ScriptWithSteps) conversationScriptResponse {
	steps := make([]conversationStepResponse, 0, len(item.Steps))
	for _, step := range item.Steps {
		steps = append(steps, newConversationStepResponse(step))
	}
	return conversationScriptResponse{
		ID:              item.Script.ID,
		Name:            item.Script.Name,
		Category:        item.Script.Category,
		Enabled:         item.Script.Enabled,
		Weight:          item.Script.Weight,
		MinWarmingScore: item.Script.MinWarmingScore,
		MaxWarmingScore: item.Script.MaxWarmingScore,
		Steps:           steps,
	}
}

func newConversationStepResponse(step repository.ConversationStep) conversationStepResponse {
	return conversationStepResponse{
		ID:              step.ID,
		ScriptID:        step.ScriptID,
		StepOrder:       step.StepOrder,
		SenderRole:      step.SenderRole,
		ActionType:      step.ActionType,
		TemplateID:      step.TemplateID,
		Payload:         step.Payload,
		MinDelaySeconds: step.MinDelaySeconds,
		MaxDelaySeconds: step.MaxDelaySeconds,
	}
}
