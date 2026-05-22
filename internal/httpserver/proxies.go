package httpserver

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"aquecedor-evolution/backend/internal/repository"
)

type testProxyRequest struct {
	Host     string  `json:"host"`
	Port     int     `json:"port"`
	Protocol string  `json:"protocol"`
	Username *string `json:"username,omitempty"`
	Password *string `json:"password,omitempty"`
}

type testProxyResponse struct {
	Success bool   `json:"success"`
	Latency string `json:"latency,omitempty"`
	IP      string `json:"ip,omitempty"`
	Error   string `json:"error,omitempty"`
}

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

func (s *Server) handleTestProxy(w http.ResponseWriter, r *http.Request) {
	var req testProxyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, testProxyResponse{Success: false, Error: "invalid json body"})
		return
	}
	if req.Host == "" {
		writeJSON(w, http.StatusBadRequest, testProxyResponse{Success: false, Error: "host is required"})
		return
	}
	if req.Port <= 0 {
		writeJSON(w, http.StatusBadRequest, testProxyResponse{Success: false, Error: "port is required"})
		return
	}
	if req.Protocol == "" {
		req.Protocol = "socks5"
	}

	proxyURL := fmt.Sprintf("%s://%s", req.Protocol, req.Host)
	if req.Port > 0 {
		proxyURL = fmt.Sprintf("%s://%s:%d", req.Protocol, req.Host, req.Port)
	}
	if req.Username != nil && *req.Username != "" && req.Password != nil && *req.Password != "" {
		proxyURL = fmt.Sprintf("%s://%s:%s@%s:%d", req.Protocol, url.QueryEscape(*req.Username), url.QueryEscape(*req.Password), req.Host, req.Port)
	}

	proxyParsed, err := url.Parse(proxyURL)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, testProxyResponse{Success: false, Error: "invalid proxy url: " + err.Error()})
		return
	}

	start := time.Now()
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyParsed),
		},
	}

	resp, err := client.Get("https://api.ipify.org?format=json")
	latency := time.Since(start)
	if err != nil {
		slog.Warn("proxy test failed", "host", req.Host, "port", req.Port, "protocol", req.Protocol, "error", err)
		writeJSON(w, http.StatusOK, testProxyResponse{
			Success: false,
			Latency: latency.Round(time.Millisecond).String(),
			Error:   err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	var result struct {
		IP string `json:"ip"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		writeJSON(w, http.StatusOK, testProxyResponse{
			Success: true,
			Latency: latency.Round(time.Millisecond).String(),
		})
		return
	}

	writeJSON(w, http.StatusOK, testProxyResponse{
		Success: true,
		Latency: latency.Round(time.Millisecond).String(),
		IP:      result.IP,
	})
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
