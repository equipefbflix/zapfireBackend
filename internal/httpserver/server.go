package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"aquecedor-evolution/backend/internal/auth"
	"aquecedor-evolution/backend/internal/config"
	"aquecedor-evolution/backend/internal/conversation"
	"aquecedor-evolution/backend/internal/evolution"
	"aquecedor-evolution/backend/internal/instance"
	"aquecedor-evolution/backend/internal/observability"
	"aquecedor-evolution/backend/internal/repository"
)

type ServerConfig struct {
	App                        config.AppConfig
	EvolutionServers           []config.EvolutionServerConfig
	InstanceCreator            InstanceCreator
	InstanceLookupByName       InstanceLookupByNameStore
	InstanceOperationalSummary InstanceOperationalSummaryStore
	DeviceModels               DeviceModelStore
	PhoneNumbers               PhoneNumberStore
	Proxies                    ProxyStore
	EvolutionStore             EvolutionServerStore
	MessageTemplates           MessageTemplateStore
	ConversationScripts        ConversationScriptStore
	WarmingJobs                WarmingJobStore
	WarmingJobRunner           WarmingJobRunnerService
	ExecutionLogs              ExecutionLogStore
	EvolutionEvents            EvolutionEventStore
	EvolutionSync              EvolutionSyncService
	Observability              ObservabilityService
	StaleJobCleanup            StaleJobCleanupService
	AuthVerifier               auth.Verifier
	FBFlixSync                 FBFlixSyncService
	HeaterActivator            HeaterActivator
	DailyLimitPerNumber        int
}

type Server struct {
	cfg    ServerConfig
	mux    *http.ServeMux
	events *instanceEventBroker
}

type InstanceCreator interface {
	Create(ctx context.Context, params instance.CreateParams) (repository.Instance, error)
	Restart(ctx context.Context, phoneNumberID string) error
	List(ctx context.Context) ([]repository.Instance, error)
	GetByID(ctx context.Context, id string) (repository.Instance, error)
	Connect(ctx context.Context, id string) (evolution.ConnectInstanceResponse, error)
	SyncState(ctx context.Context, id string) (repository.Instance, error)
	RestartByID(ctx context.Context, id string) error
	UpdateClassification(ctx context.Context, id string, classification string) (repository.Instance, error)
	DeleteByID(ctx context.Context, id string) error
}

type InstanceLookupByNameStore interface {
	GetByInstanceName(ctx context.Context, instanceName string) (repository.Instance, error)
}

type InstanceOperationalSummaryStore interface {
	GetOperationalSummary(ctx context.Context, id string) (instanceOperationalSummaryResponse, error)
}

type DeviceModelStore interface {
	Create(ctx context.Context, params repository.CreateDeviceModelParams) (repository.DeviceModel, error)
	List(ctx context.Context, includeDisabled bool) ([]repository.DeviceModel, error)
	Update(ctx context.Context, id string, params repository.UpdateDeviceModelParams) (repository.DeviceModel, error)
}

type PhoneNumberStore interface {
	Create(ctx context.Context, params repository.CreatePhoneNumberParams) (repository.PhoneNumber, error)
	List(ctx context.Context) ([]repository.PhoneNumber, error)
	Update(ctx context.Context, id string, params repository.UpdatePhoneNumberParams) (repository.PhoneNumber, error)
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (repository.PhoneNumber, error)
	GetDailyMessageCount(ctx context.Context, phoneNumberID string) (int, error)
}

type ProxyStore interface {
	Create(ctx context.Context, params repository.CreateProxyParams) (repository.Proxy, error)
	List(ctx context.Context) ([]repository.Proxy, error)
}

type EvolutionServerStore interface {
	Create(ctx context.Context, params repository.CreateEvolutionServerParams) (repository.EvolutionServer, error)
	List(ctx context.Context) ([]repository.EvolutionServer, error)
}

type MessageTemplateStore interface {
	Create(ctx context.Context, params repository.CreateMessageTemplateParams) (repository.MessageTemplate, error)
	List(ctx context.Context) ([]repository.MessageTemplate, error)
	Update(ctx context.Context, id string, params repository.UpdateMessageTemplateParams) (repository.MessageTemplate, error)
}

type ConversationScriptStore interface {
	Create(ctx context.Context, params conversation.CreateScriptParams) (conversation.ScriptWithSteps, error)
	List(ctx context.Context) ([]conversation.ScriptWithSteps, error)
	GetByID(ctx context.Context, id string) (conversation.ScriptWithSteps, error)
	Update(ctx context.Context, id string, params conversation.UpdateScriptParams) (conversation.ScriptWithSteps, error)
}

type WarmingJobStore interface {
	Create(ctx context.Context, params repository.CreateWarmingJobParams) (repository.WarmingJob, error)
	List(ctx context.Context) ([]repository.WarmingJob, error)
}

type WarmingJobRunnerService interface {
	Run(ctx context.Context, jobID string) (int, error)
}

type ExecutionLogStore interface {
	Create(ctx context.Context, params repository.CreateExecutionLogParams) (repository.ExecutionLog, error)
	List(ctx context.Context) ([]repository.ExecutionLog, error)
}

type EvolutionEventStore interface {
	Create(ctx context.Context, params repository.CreateEvolutionEventParams) (repository.EvolutionEvent, error)
}

type EvolutionSyncService interface {
	Sync(ctx context.Context, event repository.EvolutionEvent) error
}

type ObservabilityService interface {
	Snapshot(ctx context.Context) (observability.Snapshot, error)
}

type StaleJobCleanupService interface {
	Cleanup(ctx context.Context) (int64, error)
}

type FBFlixSyncService interface {
	Sync(ctx context.Context) (int, error)
}

type HeaterActivator interface {
	Activate(ctx context.Context, targetPhoneID string) error
}

type HealthResponse struct {
	Status           string                  `json:"status"`
	AppEnv           string                  `json:"appEnv"`
	EvolutionServers []HealthEvolutionServer `json:"evolutionServers"`
	Supabase         HealthDependencyStatus  `json:"supabase"`
	Metrics          *observability.Snapshot `json:"metrics,omitempty"`
}

type HealthEvolutionServer struct {
	Name    string `json:"name"`
	BaseURL string `json:"baseUrl"`
	Enabled bool   `json:"enabled"`
}

type HealthDependencyStatus struct {
	Status string `json:"status"`
}

type StaleCleanupResponse struct {
	Affected int64 `json:"affected"`
}

func NewServer(cfg ServerConfig) *Server {
	server := &Server{
		cfg:    cfg,
		mux:    http.NewServeMux(),
		events: newInstanceEventBroker(),
	}
	server.routes()
	return server
}

func (s *Server) Handler() http.Handler {
	return loggingMiddleware(corsMiddleware(authMiddleware(s.cfg.App, s.cfg.AuthVerifier, s.mux)))
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/v1/instances", s.handleListInstances)
	s.mux.HandleFunc("POST /api/v1/instances", s.handleCreateInstance)
	s.mux.HandleFunc("GET /api/v1/instances/{id}", s.handleGetInstance)
	s.mux.HandleFunc("GET /api/v1/instances/{id}/operational-summary", s.handleGetInstanceOperationalSummary)
	s.mux.HandleFunc("GET /api/v1/instances/events", s.handleInstanceEvents)
	s.mux.HandleFunc("POST /api/v1/instances/{id}/connect", s.handleConnectInstance)
	s.mux.HandleFunc("POST /api/v1/instances/{id}/restart", s.handleRestartInstance)
	s.mux.HandleFunc("PATCH /api/v1/instances/{id}/classification", s.handleUpdateInstanceClassification)
	s.mux.HandleFunc("DELETE /api/v1/instances/{id}", s.handleDeleteInstance)
	s.mux.HandleFunc("POST /api/v1/instances/{id}/sync-state", s.handleSyncInstanceState)
	s.mux.HandleFunc("GET /api/v1/device-models", s.handleListDeviceModels)
	s.mux.HandleFunc("POST /api/v1/device-models", s.handleCreateDeviceModel)
	s.mux.HandleFunc("PATCH /api/v1/device-models/{id}", s.handleUpdateDeviceModel)
	s.mux.HandleFunc("POST /api/v1/phone-numbers", s.handleCreatePhoneNumber)
	s.mux.HandleFunc("GET /api/v1/phone-numbers", s.handleListPhoneNumbers)
	s.mux.HandleFunc("PATCH /api/v1/phone-numbers/{id}", s.handleUpdatePhoneNumber)
	s.mux.HandleFunc("DELETE /api/v1/phone-numbers/{id}", s.handleDeletePhoneNumber)
	s.mux.HandleFunc("POST /api/v1/phone-numbers/{id}/restart", s.handleRestartPhoneNumberInstance)
	s.mux.HandleFunc("GET /api/v1/phone-numbers/{id}/daily-limit", s.handleGetDailyLimit)
	s.mux.HandleFunc("POST /api/v1/proxies", s.handleCreateProxy)
	s.mux.HandleFunc("GET /api/v1/proxies", s.handleListProxies)
	s.mux.HandleFunc("POST /api/v1/proxies/test", s.handleTestProxy)
	s.mux.HandleFunc("POST /api/v1/proxies/sync/fbflix", s.handleFBFlixSync)
	s.mux.HandleFunc("POST /api/v1/evolution-servers", s.handleCreateEvolutionServer)
	s.mux.HandleFunc("GET /api/v1/evolution-servers", s.handleListEvolutionServers)
	s.mux.HandleFunc("POST /api/v1/message-templates", s.handleCreateMessageTemplate)
	s.mux.HandleFunc("GET /api/v1/message-templates", s.handleListMessageTemplates)
	s.mux.HandleFunc("PATCH /api/v1/message-templates/{id}", s.handleUpdateMessageTemplate)
	s.mux.HandleFunc("POST /api/v1/conversation-scripts", s.handleCreateConversationScript)
	s.mux.HandleFunc("GET /api/v1/conversation-scripts", s.handleListConversationScripts)
	s.mux.HandleFunc("GET /api/v1/conversation-scripts/{id}", s.handleGetConversationScript)
	s.mux.HandleFunc("PATCH /api/v1/conversation-scripts/{id}", s.handleUpdateConversationScript)
	s.mux.HandleFunc("POST /api/v1/warming-jobs", s.handleCreateWarmingJob)
	s.mux.HandleFunc("GET /api/v1/warming-jobs", s.handleListWarmingJobs)
	s.mux.HandleFunc("POST /api/v1/warming-jobs/{id}/run-now", s.handleRunWarmingJobNow)
	s.mux.HandleFunc("POST /api/v1/warming-jobs/stale-cleanup", s.handleStaleCleanup)
	s.mux.HandleFunc("POST /api/v1/execution-logs", s.handleCreateExecutionLog)
	s.mux.HandleFunc("GET /api/v1/execution-logs", s.handleListExecutionLogs)
	s.mux.HandleFunc("GET /api/v1/metrics", s.handleMetrics)
	s.mux.HandleFunc("POST /api/v1/webhooks/evolution", s.handleEvolutionWebhook)
	s.mux.HandleFunc("POST /api/v1/webhooks/evolution/", s.handleEvolutionWebhook)
	s.mux.HandleFunc("POST /api/webhook/evolution", s.handleEvolutionWebhook)
	s.mux.HandleFunc("POST /api/webhook/evolution/", s.handleEvolutionWebhook)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	servers := make([]HealthEvolutionServer, 0, len(s.cfg.EvolutionServers))
	for _, evo := range s.cfg.EvolutionServers {
		servers = append(servers, HealthEvolutionServer{
			Name:    evo.Name,
			BaseURL: evo.BaseURL,
			Enabled: evo.Enabled,
		})
	}

	var metrics *observability.Snapshot
	status := "ok"
	supabaseStatus := "not_configured"
	if s.cfg.Observability != nil {
		snapshot, err := s.cfg.Observability.Snapshot(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to build health snapshot"})
			return
		}
		metrics = &snapshot
		supabaseStatus = "healthy"
		if snapshot.StaleRunningJobs > 0 {
			status = "degraded"
		}
	}

	writeJSON(w, http.StatusOK, HealthResponse{
		Status:           status,
		AppEnv:           s.cfg.App.Env,
		EvolutionServers: servers,
		Supabase: HealthDependencyStatus{
			Status: supabaseStatus,
		},
		Metrics: metrics,
	})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Observability == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "observability is not configured"})
		return
	}

	snapshot, err := s.cfg.Observability.Snapshot(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to build metrics snapshot"})
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (s *Server) handleStaleCleanup(w http.ResponseWriter, r *http.Request) {
	if s.cfg.StaleJobCleanup == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "stale cleanup is not configured"})
		return
	}

	affected, err := s.cfg.StaleJobCleanup.Cleanup(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to cleanup stale jobs"})
		return
	}
	writeJSON(w, http.StatusOK, StaleCleanupResponse{Affected: affected})
}

func (s *Server) handleFBFlixSync(w http.ResponseWriter, r *http.Request) {
	if s.cfg.FBFlixSync == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "FBFlix sync is not configured"})
		return
	}

	count, err := s.cfg.FBFlixSync.Sync(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: fmt.Sprintf("sync failed: %v", err)})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"count": count})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		slog.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.status,
			"durationMs", time.Since(startedAt).Milliseconds(),
		)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin == "" {
			origin = "*"
		}

		headers := w.Header()
		headers.Set("Access-Control-Allow-Origin", origin)
		headers.Set("Vary", "Origin")
		headers.Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		headers.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func authMiddleware(app config.AppConfig, verifier auth.Verifier, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions || !app.AuthEnabled || isPublicRoute(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		token, ok := tokenForRequest(r)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "missing bearer token"})
			return
		}
		if verifier == nil {
			writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "auth verifier is not configured"})
			return
		}

		user, err := verifier.Verify(r.Context(), token)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "invalid bearer token"})
			return
		}

		next.ServeHTTP(w, r.WithContext(auth.WithUser(r.Context(), user)))
	})
}

func isPublicRoute(path string) bool {
	switch path {
	case "/health", "/api/v1/health", "/api/v1/device-models":
		return true
	}
	return isEvolutionWebhookRoute(path)
}

func isEvolutionWebhookRoute(path string) bool {
	return path == "/api/v1/webhooks/evolution" ||
		strings.HasPrefix(path, "/api/v1/webhooks/evolution/") ||
		path == "/api/webhook/evolution" ||
		strings.HasPrefix(path, "/api/webhook/evolution/")
}

func bearerTokenFromHeader(header string) (string, bool) {
	parts := strings.SplitN(strings.TrimSpace(header), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return "", false
	}
	return strings.TrimSpace(parts[1]), true
}

func tokenForRequest(r *http.Request) (string, bool) {
	if token, ok := bearerTokenFromHeader(r.Header.Get("Authorization")); ok {
		return token, true
	}
	if r.URL.Path == "/api/v1/instances/events" {
		token := strings.TrimSpace(r.URL.Query().Get("access_token"))
		if token != "" {
			return token, true
		}
	}
	return "", false
}
