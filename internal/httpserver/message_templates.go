package httpserver

import (
	"encoding/json"
	"net/http"

	"aquecedor-evolution/backend/internal/repository"
)

type createMessageTemplateRequest struct {
	Category        string         `json:"category"`
	Title           string         `json:"title"`
	Body            string         `json:"body"`
	Weight          int            `json:"weight"`
	Enabled         bool           `json:"enabled"`
	MinWarmingScore float64        `json:"minWarmingScore"`
	MaxWarmingScore float64        `json:"maxWarmingScore"`
	TestRunID       string         `json:"testRunId,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

type messageTemplateResponse struct {
	ID              string         `json:"id"`
	Category        string         `json:"category"`
	Title           string         `json:"title"`
	Body            string         `json:"body"`
	Weight          int            `json:"weight"`
	Enabled         bool           `json:"enabled"`
	MinWarmingScore float64        `json:"minWarmingScore"`
	MaxWarmingScore float64        `json:"maxWarmingScore"`
	Metadata        map[string]any `json:"metadata"`
}

type listMessageTemplatesResponse struct {
	Items []messageTemplateResponse `json:"items"`
}

type updateMessageTemplateRequest struct {
	Category        *string        `json:"category"`
	Title           *string        `json:"title"`
	Body            *string        `json:"body"`
	Weight          *int           `json:"weight"`
	Enabled         *bool          `json:"enabled"`
	MinWarmingScore *float64       `json:"minWarmingScore"`
	MaxWarmingScore *float64       `json:"maxWarmingScore"`
	Metadata        map[string]any `json:"metadata"`
}

func (s *Server) handleCreateMessageTemplate(w http.ResponseWriter, r *http.Request) {
	if s.cfg.MessageTemplates == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "message template store is not configured"})
		return
	}

	var request createMessageTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json body"})
		return
	}
	if request.Category == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "category is required"})
		return
	}
	if request.Title == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "title is required"})
		return
	}
	if request.Body == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "body is required"})
		return
	}
	if request.Weight <= 0 {
		request.Weight = 1
	}
	if request.MaxWarmingScore <= 0 {
		request.MaxWarmingScore = 100
	}

	metadata := request.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	if request.TestRunID != "" {
		metadata["testRunId"] = request.TestRunID
	}

	created, err := s.cfg.MessageTemplates.Create(r.Context(), repository.CreateMessageTemplateParams{
		Category:        request.Category,
		Title:           request.Title,
		Body:            request.Body,
		Weight:          request.Weight,
		Enabled:         request.Enabled,
		MinWarmingScore: request.MinWarmingScore,
		MaxWarmingScore: request.MaxWarmingScore,
		Metadata:        metadata,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to create message template"})
		return
	}

	writeJSON(w, http.StatusCreated, newMessageTemplateResponse(created))
}

func (s *Server) handleListMessageTemplates(w http.ResponseWriter, r *http.Request) {
	if s.cfg.MessageTemplates == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "message template store is not configured"})
		return
	}

	items, err := s.cfg.MessageTemplates.List(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to list message templates"})
		return
	}

	response := listMessageTemplatesResponse{
		Items: make([]messageTemplateResponse, 0, len(items)),
	}
	for _, item := range items {
		response.Items = append(response.Items, newMessageTemplateResponse(item))
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleUpdateMessageTemplate(w http.ResponseWriter, r *http.Request) {
	if s.cfg.MessageTemplates == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "message template store is not configured"})
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "id is required"})
		return
	}
	var request updateMessageTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json body"})
		return
	}
	updated, err := s.cfg.MessageTemplates.Update(r.Context(), id, repository.UpdateMessageTemplateParams{
		Category:        request.Category,
		Title:           request.Title,
		Body:            request.Body,
		Weight:          request.Weight,
		Enabled:         request.Enabled,
		MinWarmingScore: request.MinWarmingScore,
		MaxWarmingScore: request.MaxWarmingScore,
		Metadata:        request.Metadata,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to update message template"})
		return
	}
	writeJSON(w, http.StatusOK, newMessageTemplateResponse(updated))
}

func newMessageTemplateResponse(template repository.MessageTemplate) messageTemplateResponse {
	return messageTemplateResponse{
		ID:              template.ID,
		Category:        template.Category,
		Title:           template.Title,
		Body:            template.Body,
		Weight:          template.Weight,
		Enabled:         template.Enabled,
		MinWarmingScore: template.MinWarmingScore,
		MaxWarmingScore: template.MaxWarmingScore,
		Metadata:        template.Metadata,
	}
}
