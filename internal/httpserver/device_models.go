package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"

	"aquecedor-evolution/backend/internal/auth"
	"aquecedor-evolution/backend/internal/repository"
)

type createDeviceModelRequest struct {
	Name             string         `json:"name"`
	OS               string         `json:"os"`
	SystemLabel      string         `json:"systemLabel"`
	VersionLabel     string         `json:"versionLabel"`
	ImageURL         string         `json:"imageUrl"`
	TechnicalProfile map[string]any `json:"technicalProfile,omitempty"`
	SortOrder        int            `json:"sortOrder"`
	Enabled          bool           `json:"enabled"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

type updateDeviceModelRequest struct {
	Name             *string        `json:"name"`
	OS               *string        `json:"os"`
	SystemLabel      *string        `json:"systemLabel"`
	VersionLabel     *string        `json:"versionLabel"`
	ImageURL         *string        `json:"imageUrl"`
	TechnicalProfile map[string]any `json:"technicalProfile,omitempty"`
	SortOrder        *int           `json:"sortOrder"`
	Enabled          *bool          `json:"enabled"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

type deviceModelResponse struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	OS               string         `json:"os"`
	SystemLabel      string         `json:"systemLabel"`
	VersionLabel     string         `json:"versionLabel"`
	ImageURL         string         `json:"imageUrl"`
	TechnicalProfile map[string]any `json:"technicalProfile,omitempty"`
	SortOrder        int            `json:"sortOrder"`
	Enabled          bool           `json:"enabled"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

type listDeviceModelsResponse struct {
	Items []deviceModelResponse `json:"items"`
}

func (s *Server) handleCreateDeviceModel(w http.ResponseWriter, r *http.Request) {
	if s.cfg.DeviceModels == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "device model store is not configured"})
		return
	}

	var request createDeviceModelRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json body"})
		return
	}
	if request.Name == "" || request.OS == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "name and os are required"})
		return
	}

	metadata := mergeTechnicalProfile(request.Metadata, request.TechnicalProfile)
	created, err := s.cfg.DeviceModels.Create(r.Context(), repository.CreateDeviceModelParams{
		Name:         request.Name,
		OS:           request.OS,
		SystemLabel:  request.SystemLabel,
		VersionLabel: request.VersionLabel,
		ImageURL:     request.ImageURL,
		SortOrder:    request.SortOrder,
		Enabled:      request.Enabled,
		Metadata:     metadata,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to create device model"})
		return
	}
	writeJSON(w, http.StatusCreated, newDeviceModelResponse(created))
}

func (s *Server) handleListDeviceModels(w http.ResponseWriter, r *http.Request) {
	if s.cfg.DeviceModels == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "device model store is not configured"})
		return
	}

	includeDisabled, _ := strconv.ParseBool(r.URL.Query().Get("includeDisabled"))
	if _, ok := auth.UserFromContext(r.Context()); !ok {
		includeDisabled = false
	}
	items, err := s.cfg.DeviceModels.List(r.Context(), includeDisabled)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to list device models"})
		return
	}
	response := listDeviceModelsResponse{Items: make([]deviceModelResponse, 0, len(items))}
	for _, item := range items {
		response.Items = append(response.Items, newDeviceModelResponse(item))
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleUpdateDeviceModel(w http.ResponseWriter, r *http.Request) {
	if s.cfg.DeviceModels == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "device model store is not configured"})
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "id is required"})
		return
	}

	var request updateDeviceModelRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json body"})
		return
	}

	metadata := mergeTechnicalProfile(request.Metadata, request.TechnicalProfile)
	updated, err := s.cfg.DeviceModels.Update(r.Context(), id, repository.UpdateDeviceModelParams{
		Name:         request.Name,
		OS:           request.OS,
		SystemLabel:  request.SystemLabel,
		VersionLabel: request.VersionLabel,
		ImageURL:     request.ImageURL,
		SortOrder:    request.SortOrder,
		Enabled:      request.Enabled,
		Metadata:     metadata,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to update device model"})
		return
	}

	writeJSON(w, http.StatusOK, newDeviceModelResponse(updated))
}

func newDeviceModelResponse(model repository.DeviceModel) deviceModelResponse {
	return deviceModelResponse{
		ID:               model.ID,
		Name:             model.Name,
		OS:               model.OS,
		SystemLabel:      model.SystemLabel,
		VersionLabel:     model.VersionLabel,
		ImageURL:         model.ImageURL,
		TechnicalProfile: extractTechnicalProfile(model.Metadata),
		SortOrder:        model.SortOrder,
		Enabled:          model.Enabled,
		Metadata:         model.Metadata,
	}
}

func mergeTechnicalProfile(metadata map[string]any, technicalProfile map[string]any) map[string]any {
	merged := map[string]any{}
	for key, value := range metadata {
		merged[key] = value
	}
	if len(technicalProfile) > 0 {
		merged["technicalProfile"] = technicalProfile
	}
	return merged
}

func extractTechnicalProfile(metadata map[string]any) map[string]any {
	if metadata == nil {
		return nil
	}
	value, ok := metadata["technicalProfile"]
	if !ok {
		return nil
	}
	profile, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return profile
}
