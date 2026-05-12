package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"aquecedor-evolution/backend/internal/repository"
)

type createProxyRequest struct {
	Name               string         `json:"name"`
	Host               string         `json:"host"`
	Port               int            `json:"port"`
	Protocol           string         `json:"protocol"`
	Username           *string        `json:"username,omitempty"`
	PasswordSecretName *string        `json:"passwordSecretName,omitempty"`
	Enabled            bool           `json:"enabled"`
	MaxInstances       *int           `json:"maxInstances,omitempty"`
	TestRunID          string         `json:"testRunId,omitempty"`
	Metadata           map[string]any `json:"metadata,omitempty"`
}

type proxyResponse struct {
	ID                 string         `json:"id"`
	Name               string         `json:"name"`
	Host               string         `json:"host"`
	Port               int            `json:"port"`
	Protocol           string         `json:"protocol"`
	Username           *string        `json:"username,omitempty"`
	PasswordSecretName *string        `json:"passwordSecretName,omitempty"`
	Enabled            bool           `json:"enabled"`
	MaxInstances       *int           `json:"maxInstances,omitempty"`
	CurrentInstances   int            `json:"currentInstances"`
	Metadata           map[string]any `json:"metadata"`
}

type listProxiesResponse struct {
	Items []proxyResponse `json:"items"`
}

func (s *Server) handleCreateProxy(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Proxies == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "proxy store is not configured"})
		return
	}

	var request createProxyRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json body"})
		return
	}
	if request.Name == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "name is required"})
		return
	}
	if request.Host == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "host is required"})
		return
	}
	if request.Port <= 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "port must be greater than zero"})
		return
	}
	if request.Protocol == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "protocol is required"})
		return
	}

	metadata := request.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	if request.TestRunID != "" {
		metadata["testRunId"] = request.TestRunID
	}

	created, err := s.cfg.Proxies.Create(r.Context(), repository.CreateProxyParams{
		Name:               request.Name,
		Host:               request.Host,
		Port:               request.Port,
		Protocol:           request.Protocol,
		Username:           request.Username,
		PasswordSecretName: request.PasswordSecretName,
		Enabled:            request.Enabled,
		MaxInstances:       request.MaxInstances,
		Metadata:           metadata,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to create proxy"})
		return
	}

	writeJSON(w, http.StatusCreated, newProxyResponse(created))
}

func (s *Server) handleListProxies(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Proxies == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "proxy store is not configured"})
		return
	}

	items, err := s.cfg.Proxies.List(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to list proxies"})
		return
	}

	response := listProxiesResponse{
		Items: make([]proxyResponse, 0, len(items)),
	}
	for _, item := range items {
		response.Items = append(response.Items, newProxyResponse(item))
	}
	writeJSON(w, http.StatusOK, response)
}

func newProxyResponse(proxy repository.Proxy) proxyResponse {
	return proxyResponse{
		ID:                 proxy.ID,
		Name:               proxy.Name,
		Host:               proxy.Host,
		Port:               proxy.Port,
		Protocol:           proxy.Protocol,
		Username:           proxy.Username,
		PasswordSecretName: publicPasswordSecretName(proxy.PasswordSecretName),
		Enabled:            proxy.Enabled,
		MaxInstances:       proxy.MaxInstances,
		CurrentInstances:   proxy.CurrentInstances,
		Metadata:           proxy.Metadata,
	}
}

func publicPasswordSecretName(secretName *string) *string {
	if secretName == nil {
		return nil
	}
	if strings.HasPrefix(*secretName, "literal:") {
		redacted := "literal:***"
		return &redacted
	}
	return secretName
}
