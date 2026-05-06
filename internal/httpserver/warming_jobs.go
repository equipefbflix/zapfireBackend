package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"aquecedor-evolution/backend/internal/repository"
)

type createWarmingJobRequest struct {
	ScriptID    *string        `json:"scriptId,omitempty"`
	PhoneAID    string         `json:"phoneAId"`
	PhoneBID    string         `json:"phoneBId"`
	ScheduledAt string         `json:"scheduledAt"`
	TestRunID   string         `json:"testRunId,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type warmingJobResponse struct {
	ID               string         `json:"id"`
	ScriptID         *string        `json:"scriptId,omitempty"`
	PhoneAID         string         `json:"phoneAId"`
	PhoneBID         string         `json:"phoneBId"`
	Status           string         `json:"status"`
	ScheduledAt      time.Time      `json:"scheduledAt"`
	CurrentStepOrder int            `json:"currentStepOrder"`
	Error            string         `json:"error"`
	Metadata         map[string]any `json:"metadata"`
}

type listWarmingJobsResponse struct {
	Items []warmingJobResponse `json:"items"`
}

type runWarmingJobNowResponse struct {
	JobID          string `json:"jobId"`
	ExecutedSteps  int    `json:"executedSteps"`
}

func (s *Server) handleCreateWarmingJob(w http.ResponseWriter, r *http.Request) {
	if s.cfg.WarmingJobs == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "warming job store is not configured"})
		return
	}

	var request createWarmingJobRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json body"})
		return
	}
	if request.PhoneAID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "phoneAId is required"})
		return
	}
	if request.PhoneBID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "phoneBId is required"})
		return
	}
	if request.PhoneAID == request.PhoneBID {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "phoneAId and phoneBId must be different"})
		return
	}
	if request.ScheduledAt == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "scheduledAt is required"})
		return
	}
	scheduledAt, err := time.Parse(time.RFC3339, request.ScheduledAt)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "scheduledAt must be RFC3339"})
		return
	}

	metadata := request.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	if request.TestRunID != "" {
		metadata["testRunId"] = request.TestRunID
	}

	created, err := s.cfg.WarmingJobs.Create(r.Context(), repository.CreateWarmingJobParams{
		ScriptID:    request.ScriptID,
		PhoneAID:    request.PhoneAID,
		PhoneBID:    request.PhoneBID,
		ScheduledAt: scheduledAt,
		Metadata:    metadata,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to create warming job"})
		return
	}

	writeJSON(w, http.StatusCreated, newWarmingJobResponse(created))
}

func (s *Server) handleListWarmingJobs(w http.ResponseWriter, r *http.Request) {
	if s.cfg.WarmingJobs == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "warming job store is not configured"})
		return
	}

	items, err := s.cfg.WarmingJobs.List(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to list warming jobs"})
		return
	}

	response := listWarmingJobsResponse{
		Items: make([]warmingJobResponse, 0, len(items)),
	}
	for _, item := range items {
		response.Items = append(response.Items, newWarmingJobResponse(item))
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleRunWarmingJobNow(w http.ResponseWriter, r *http.Request) {
	if s.cfg.WarmingJobRunner == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "warming job runner is not configured"})
		return
	}

	jobID := r.PathValue("id")
	if jobID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "job id is required"})
		return
	}

	executed, err := s.cfg.WarmingJobRunner.Run(r.Context(), jobID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to run warming job"})
		return
	}

	writeJSON(w, http.StatusOK, runWarmingJobNowResponse{
		JobID:         jobID,
		ExecutedSteps: executed,
	})
}

func newWarmingJobResponse(job repository.WarmingJob) warmingJobResponse {
	return warmingJobResponse{
		ID:               job.ID,
		ScriptID:         job.ScriptID,
		PhoneAID:         job.PhoneAID,
		PhoneBID:         job.PhoneBID,
		Status:           job.Status,
		ScheduledAt:      job.ScheduledAt,
		CurrentStepOrder: job.CurrentStepOrder,
		Error:            job.Error,
		Metadata:         job.Metadata,
	}
}
