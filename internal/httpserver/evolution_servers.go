package httpserver

import (
	"encoding/json"
	"net/http"

	"aquecedor-evolution/backend/internal/repository"
)

type createEvolutionServerRequest struct {
	Name              string         `json:"name"`
	BaseURL           string         `json:"baseUrl"`
	APIKeySecretName  string         `json:"apiKeySecretName"`
	Enabled           bool           `json:"enabled"`
	Weight            int            `json:"weight"`
	MaxConcurrentJobs int            `json:"maxConcurrentJobs"`
	TestRunID         string         `json:"testRunId,omitempty"`
	Metadata          map[string]any `json:"metadata,omitempty"`
}

type evolutionServerResponse struct {
	ID                string         `json:"id"`
	Name              string         `json:"name"`
	BaseURL           string         `json:"baseUrl"`
	APIKeySecretName  string         `json:"apiKeySecretName"`
	Enabled           bool           `json:"enabled"`
	Weight            int            `json:"weight"`
	MaxConcurrentJobs int            `json:"maxConcurrentJobs"`
	HealthStatus      string         `json:"healthStatus"`
	Metadata          map[string]any `json:"metadata"`
}

type listEvolutionServersResponse struct {
	Items []evolutionServerResponse `json:"items"`
}

func (s *Server) handleCreateEvolutionServer(w http.ResponseWriter, r *http.Request) {
	if s.cfg.EvolutionStore == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "evolution server store is not configured"})
		return
	}

	var request createEvolutionServerRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json body"})
		return
	}
	if request.Name == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "name is required"})
		return
	}
	if request.BaseURL == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "baseUrl is required"})
		return
	}
	if request.APIKeySecretName == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "apiKeySecretName is required"})
		return
	}
	if request.Weight <= 0 {
		request.Weight = 1
	}
	if request.MaxConcurrentJobs <= 0 {
		request.MaxConcurrentJobs = 1
	}

	metadata := request.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	if request.TestRunID != "" {
		metadata["testRunId"] = request.TestRunID
	}

	created, err := s.cfg.EvolutionStore.Create(r.Context(), repository.CreateEvolutionServerParams{
		Name:              request.Name,
		BaseURL:           request.BaseURL,
		APIKeySecretName:  request.APIKeySecretName,
		Enabled:           request.Enabled,
		Weight:            request.Weight,
		MaxConcurrentJobs: request.MaxConcurrentJobs,
		Metadata:          metadata,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to create evolution server"})
		return
	}

	writeJSON(w, http.StatusCreated, newEvolutionServerResponse(created))
}

func (s *Server) handleListEvolutionServers(w http.ResponseWriter, r *http.Request) {
	if s.cfg.EvolutionStore == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "evolution server store is not configured"})
		return
	}

	items, err := s.cfg.EvolutionStore.List(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to list evolution servers"})
		return
	}

	response := listEvolutionServersResponse{
		Items: make([]evolutionServerResponse, 0, len(items)),
	}
	for _, item := range items {
		response.Items = append(response.Items, newEvolutionServerResponse(item))
	}
	writeJSON(w, http.StatusOK, response)
}

func newEvolutionServerResponse(server repository.EvolutionServer) evolutionServerResponse {
	return evolutionServerResponse{
		ID:                server.ID,
		Name:              server.Name,
		BaseURL:           server.BaseURL,
		APIKeySecretName:  server.APIKeySecretName,
		Enabled:           server.Enabled,
		Weight:            server.Weight,
		MaxConcurrentJobs: server.MaxConcurrentJobs,
		HealthStatus:      server.HealthStatus,
		Metadata:          server.Metadata,
	}
}
