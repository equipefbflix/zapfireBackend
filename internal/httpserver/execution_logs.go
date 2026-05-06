package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"aquecedor-evolution/backend/internal/repository"
)

type createExecutionLogRequest struct {
	WarmingJobID        *string        `json:"warmingJobId,omitempty"`
	InstanceID          *string        `json:"instanceId,omitempty"`
	ActionType          *string        `json:"actionType,omitempty"`
	Status              string         `json:"status"`
	RequestPayload      map[string]any `json:"requestPayload,omitempty"`
	ResponsePayload     map[string]any `json:"responsePayload,omitempty"`
	EvolutionMessageKey map[string]any `json:"evolutionMessageKey,omitempty"`
	RemoteJID           string         `json:"remoteJid,omitempty"`
	Error               string         `json:"error,omitempty"`
	DurationMs          *int           `json:"durationMs,omitempty"`
}

type executionLogResponse struct {
	ID                  string         `json:"id"`
	WarmingJobID        *string        `json:"warmingJobId,omitempty"`
	InstanceID          *string        `json:"instanceId,omitempty"`
	ActionType          *string        `json:"actionType,omitempty"`
	Status              string         `json:"status"`
	RequestPayload      map[string]any `json:"requestPayload"`
	ResponsePayload     map[string]any `json:"responsePayload"`
	EvolutionMessageKey map[string]any `json:"evolutionMessageKey"`
	RemoteJID           string         `json:"remoteJid"`
	Error               string         `json:"error"`
	DurationMs          *int           `json:"durationMs,omitempty"`
	CreatedAt           time.Time      `json:"createdAt"`
}

type listExecutionLogsResponse struct {
	Items []executionLogResponse `json:"items"`
}

func (s *Server) handleCreateExecutionLog(w http.ResponseWriter, r *http.Request) {
	if s.cfg.ExecutionLogs == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "execution log store is not configured"})
		return
	}

	var request createExecutionLogRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json body"})
		return
	}
	if request.Status == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "status is required"})
		return
	}

	created, err := s.cfg.ExecutionLogs.Create(r.Context(), repository.CreateExecutionLogParams{
		WarmingJobID:        request.WarmingJobID,
		InstanceID:          request.InstanceID,
		ActionType:          request.ActionType,
		Status:              request.Status,
		RequestPayload:      nonNilMap(request.RequestPayload),
		ResponsePayload:     nonNilMap(request.ResponsePayload),
		EvolutionMessageKey: nonNilMap(request.EvolutionMessageKey),
		RemoteJID:           request.RemoteJID,
		Error:               request.Error,
		DurationMs:          request.DurationMs,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to create execution log"})
		return
	}

	writeJSON(w, http.StatusCreated, newExecutionLogResponse(created))
}

func (s *Server) handleListExecutionLogs(w http.ResponseWriter, r *http.Request) {
	if s.cfg.ExecutionLogs == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "execution log store is not configured"})
		return
	}

	items, err := s.cfg.ExecutionLogs.List(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to list execution logs"})
		return
	}

	response := listExecutionLogsResponse{
		Items: make([]executionLogResponse, 0, len(items)),
	}
	for _, item := range items {
		response.Items = append(response.Items, newExecutionLogResponse(item))
	}
	writeJSON(w, http.StatusOK, response)
}

func newExecutionLogResponse(log repository.ExecutionLog) executionLogResponse {
	return executionLogResponse{
		ID:                  log.ID,
		WarmingJobID:        log.WarmingJobID,
		InstanceID:          log.InstanceID,
		ActionType:          log.ActionType,
		Status:              log.Status,
		RequestPayload:      nonNilMap(log.RequestPayload),
		ResponsePayload:     nonNilMap(log.ResponsePayload),
		EvolutionMessageKey: nonNilMap(log.EvolutionMessageKey),
		RemoteJID:           log.RemoteJID,
		Error:               log.Error,
		DurationMs:          log.DurationMs,
		CreatedAt:           log.CreatedAt,
	}
}

func nonNilMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}
