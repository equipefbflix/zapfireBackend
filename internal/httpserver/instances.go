package httpserver

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"aquecedor-evolution/backend/internal/instance"
	"aquecedor-evolution/backend/internal/repository"
)

type createInstanceRequest struct {
	PhoneNumberID  string              `json:"phoneNumberId"`
	PhoneE164      string              `json:"phoneE164"`
	InstanceName   string              `json:"instanceName"`
	Classification string              `json:"classification,omitempty"`
	TestRunID      string              `json:"testRunId,omitempty"`
	ManualProxy    *manualProxyRequest `json:"manualProxy,omitempty"`
}

type manualProxyRequest struct {
	Host     string  `json:"host"`
	Port     int     `json:"port"`
	Protocol string  `json:"protocol"`
	Username *string `json:"username,omitempty"`
	Password *string `json:"password,omitempty"`
}

type createInstanceResponse struct {
	ID                  string         `json:"id"`
	PhoneNumberID       string         `json:"phoneNumberId,omitempty"`
	EvolutionServerID   string         `json:"evolutionServerId"`
	ProxyID             *string        `json:"proxyId,omitempty"`
	InstanceName        string         `json:"instanceName"`
	EvolutionInstanceID *string        `json:"evolutionInstanceId,omitempty"`
	Status              string         `json:"status"`
	Classification      string         `json:"classification"`
	Metadata            map[string]any `json:"metadata,omitempty"`
}

type updateInstanceClassificationRequest struct {
	Classification string `json:"classification"`
}

type connectInstanceResponse struct {
	PairingCode string `json:"pairingCode,omitempty"`
	Code        string `json:"code,omitempty"`
	Count       int    `json:"count,omitempty"`
}

type instanceOperationalSummaryResponse struct {
	InstanceID         string  `json:"instanceId"`
	PhoneNumberID      string  `json:"phoneNumberId"`
	InstanceName       string  `json:"instanceName"`
	Status             string  `json:"status"`
	ConnectionStatus   string  `json:"connectionStatus"`
	WarmingScore       float64 `json:"warmingScore"`
	DailyMessageCount  int     `json:"dailyMessageCount"`
	DailyLimit         int     `json:"dailyLimit"`
	LastConnectionSync string  `json:"lastConnectionSync,omitempty"`
}

type listInstancesResponse struct {
	Items []createInstanceResponse `json:"items"`
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
	if request.InstanceName == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "instanceName is required"})
		return
	}
	manualProxy, err := validateAndBuildManualProxy(request.ManualProxy)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	created, err := s.cfg.InstanceCreator.Create(r.Context(), instance.CreateParams{
		PhoneNumberID:  request.PhoneNumberID,
		PhoneE164:      request.PhoneE164,
		InstanceName:   request.InstanceName,
		Classification: request.Classification,
		TestRunID:      request.TestRunID,
		ManualProxy:    manualProxy,
	})
	if err != nil {
		if errors.Is(err, instance.ErrNoEvolutionServer) {
			writeJSON(w, http.StatusConflict, errorResponse{Error: "no enabled evolution server available"})
			return
		}
		slog.Error("failed to create instance", "error", err, "instanceName", request.InstanceName)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to create instance: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, newCreateInstanceResponse(created))
}

func validateAndBuildManualProxy(proxy *manualProxyRequest) (*instance.ManualProxyInput, error) {
	if proxy == nil {
		return nil, nil
	}
	host := strings.TrimSpace(proxy.Host)
	if host == "" {
		return nil, errors.New("manualProxy.host is required")
	}
	if proxy.Port <= 0 || proxy.Port > 65535 {
		return nil, errors.New("manualProxy.port must be between 1 and 65535")
	}
	protocol := strings.ToLower(strings.TrimSpace(proxy.Protocol))
	switch protocol {
	case "http", "https", "socks5":
	default:
		return nil, errors.New("manualProxy.protocol must be one of http, https or socks5")
	}
	return &instance.ManualProxyInput{
		Host:     host,
		Port:     proxy.Port,
		Protocol: protocol,
		Username: proxy.Username,
		Password: proxy.Password,
	}, nil
}

func (s *Server) handleUpdateInstanceClassification(w http.ResponseWriter, r *http.Request) {
	if s.cfg.InstanceCreator == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "instance creator is not configured"})
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "id is required"})
		return
	}

	var request updateInstanceClassificationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json body"})
		return
	}
	if request.Classification == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "classification is required"})
		return
	}

	item, err := s.cfg.InstanceCreator.UpdateClassification(r.Context(), id, request.Classification)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to update instance classification"})
		return
	}
	writeJSON(w, http.StatusOK, newCreateInstanceResponse(item))
}

func (s *Server) handleListInstances(w http.ResponseWriter, r *http.Request) {
	if s.cfg.InstanceCreator == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "instance creator is not configured"})
		return
	}
	items, err := s.cfg.InstanceCreator.List(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to list instances"})
		return
	}
	response := listInstancesResponse{Items: make([]createInstanceResponse, 0, len(items))}
	for _, item := range items {
		response.Items = append(response.Items, newCreateInstanceResponse(item))
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetInstance(w http.ResponseWriter, r *http.Request) {
	if s.cfg.InstanceCreator == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "instance creator is not configured"})
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "id is required"})
		return
	}
	item, err := s.cfg.InstanceCreator.GetByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to get instance"})
		return
	}
	writeJSON(w, http.StatusOK, newCreateInstanceResponse(item))
}

func (s *Server) handleGetInstanceOperationalSummary(w http.ResponseWriter, r *http.Request) {
	if s.cfg.InstanceOperationalSummary == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "instance operational summary store is not configured"})
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "id is required"})
		return
	}
	item, err := s.cfg.InstanceOperationalSummary.GetOperationalSummary(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to get instance operational summary"})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleConnectInstance(w http.ResponseWriter, r *http.Request) {
	if s.cfg.InstanceCreator == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "instance creator is not configured"})
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "id is required"})
		return
	}
	response, err := s.cfg.InstanceCreator.Connect(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to connect instance"})
		return
	}
	writeJSON(w, http.StatusOK, connectInstanceResponse{
		PairingCode: response.PairingCode,
		Code:        response.Code,
		Count:       response.Count,
	})
}

func (s *Server) handleSyncInstanceState(w http.ResponseWriter, r *http.Request) {
	if s.cfg.InstanceCreator == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "instance creator is not configured"})
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "id is required"})
		return
	}
	item, err := s.cfg.InstanceCreator.SyncState(r.Context(), id)
	if err != nil {
		slog.Error("failed to sync instance state", "error", err, "instanceId", id)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to sync instance state: " + err.Error()})
		return
	}
	s.events.Publish(instanceEventMessage{
		InstanceID:       item.ID,
		InstanceName:     item.InstanceName,
		Status:           item.Status,
		ConnectionStatus: item.Status,
	})
	writeJSON(w, http.StatusOK, newCreateInstanceResponse(item))
}

func (s *Server) handleRestartInstance(w http.ResponseWriter, r *http.Request) {
	if s.cfg.InstanceCreator == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "instance creator is not configured"})
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "id is required"})
		return
	}
	if err := s.cfg.InstanceCreator.RestartByID(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to restart instance"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "restart initiated"})
}

func (s *Server) handleDeleteInstance(w http.ResponseWriter, r *http.Request) {
	if s.cfg.InstanceCreator == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "instance creator is not configured"})
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "id is required"})
		return
	}
	if err := s.cfg.InstanceCreator.DeleteByID(r.Context(), id); err != nil {
		slog.Error("failed to delete instance", "error", err, "instanceId", id)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to delete instance: " + err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func newCreateInstanceResponse(created repository.Instance) createInstanceResponse {
	return createInstanceResponse{
		ID:                  created.ID,
		PhoneNumberID:       created.PhoneNumberID,
		EvolutionServerID:   created.EvolutionServerID,
		ProxyID:             created.ProxyID,
		InstanceName:        created.InstanceName,
		EvolutionInstanceID: created.EvolutionInstanceID,
		Status:              created.Status,
		Classification:      instanceClassification(created),
		Metadata:            created.Metadata,
	}
}

func instanceClassification(item repository.Instance) string {
	if item.Metadata != nil {
		if raw, ok := item.Metadata["classification"].(string); ok {
			switch raw {
			case "internal", "external":
				return raw
			}
		}
	}
	return "external"
}
