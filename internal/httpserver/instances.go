package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"

	"aquecedor-evolution/backend/internal/instance"
	"aquecedor-evolution/backend/internal/repository"
)

type createInstanceRequest struct {
	PhoneNumberID string `json:"phoneNumberId"`
	PhoneE164     string `json:"phoneE164"`
	InstanceName  string `json:"instanceName"`
	TestRunID     string `json:"testRunId,omitempty"`
}

type createInstanceResponse struct {
	ID                string  `json:"id"`
	PhoneNumberID     string  `json:"phoneNumberId"`
	EvolutionServerID string  `json:"evolutionServerId"`
	ProxyID           *string `json:"proxyId,omitempty"`
	InstanceName      string  `json:"instanceName"`
	Status            string  `json:"status"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (s *Server) handleCreateInstance(w http.ResponseWriter, r *http.Request) {
	if s.cfg.InstanceCreator == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "instance creator is not configured"})
		return
	}

	var request createInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json body"})
		return
	}
	if request.PhoneNumberID == "" || request.PhoneE164 == "" || request.InstanceName == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "phoneNumberId, phoneE164 and instanceName are required"})
		return
	}

	created, err := s.cfg.InstanceCreator.Create(r.Context(), instance.CreateParams{
		PhoneNumberID: request.PhoneNumberID,
		PhoneE164:     request.PhoneE164,
		InstanceName:  request.InstanceName,
		TestRunID:     request.TestRunID,
	})
	if err != nil {
		if errors.Is(err, instance.ErrNoEvolutionServer) {
			writeJSON(w, http.StatusConflict, errorResponse{Error: "no enabled evolution server available"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to create instance"})
		return
	}

	writeJSON(w, http.StatusCreated, newCreateInstanceResponse(created))
}

func newCreateInstanceResponse(created repository.Instance) createInstanceResponse {
	return createInstanceResponse{
		ID:                created.ID,
		PhoneNumberID:     created.PhoneNumberID,
		EvolutionServerID: created.EvolutionServerID,
		ProxyID:           created.ProxyID,
		InstanceName:      created.InstanceName,
		Status:            created.Status,
	}
}
