package httpserver

import (
	"encoding/json"
	"net/http"

	"aquecedor-evolution/backend/internal/repository"
)

type createPhoneNumberRequest struct {
	PhoneE164 string         `json:"phoneE164"`
	Label     string         `json:"label"`
	TestRunID string         `json:"testRunId,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type phoneNumberResponse struct {
	ID           string         `json:"id"`
	PhoneE164    string         `json:"phoneE164"`
	Label        string         `json:"label"`
	Status       string         `json:"status"`
	WarmingScore float64        `json:"warmingScore"`
	Metadata     map[string]any `json:"metadata"`
}

type listPhoneNumbersResponse struct {
	Items []phoneNumberResponse `json:"items"`
}

func (s *Server) handleCreatePhoneNumber(w http.ResponseWriter, r *http.Request) {
	if s.cfg.PhoneNumbers == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "phone number store is not configured"})
		return
	}

	var request createPhoneNumberRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json body"})
		return
	}
	if request.PhoneE164 == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "phoneE164 is required"})
		return
	}

	metadata := request.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	if request.TestRunID != "" {
		metadata["testRunId"] = request.TestRunID
	}

	created, err := s.cfg.PhoneNumbers.Create(r.Context(), repository.CreatePhoneNumberParams{
		PhoneE164: request.PhoneE164,
		Label:     request.Label,
		Metadata:  metadata,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to create phone number"})
		return
	}

	writeJSON(w, http.StatusCreated, newPhoneNumberResponse(created))
}

func (s *Server) handleListPhoneNumbers(w http.ResponseWriter, r *http.Request) {
	if s.cfg.PhoneNumbers == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "phone number store is not configured"})
		return
	}

	items, err := s.cfg.PhoneNumbers.List(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to list phone numbers"})
		return
	}

	response := listPhoneNumbersResponse{
		Items: make([]phoneNumberResponse, 0, len(items)),
	}
	for _, item := range items {
		response.Items = append(response.Items, newPhoneNumberResponse(item))
	}
	writeJSON(w, http.StatusOK, response)
}

func newPhoneNumberResponse(phone repository.PhoneNumber) phoneNumberResponse {
	return phoneNumberResponse{
		ID:           phone.ID,
		PhoneE164:    phone.PhoneE164,
		Label:        phone.Label,
		Status:       phone.Status,
		WarmingScore: phone.WarmingScore,
		Metadata:     phone.Metadata,
	}
}
