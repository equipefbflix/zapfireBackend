package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"aquecedor-evolution/backend/internal/auth"
	"aquecedor-evolution/backend/internal/config"
	"aquecedor-evolution/backend/internal/conversation"
	"aquecedor-evolution/backend/internal/instance"
	"aquecedor-evolution/backend/internal/observability"
	"aquecedor-evolution/backend/internal/repository"
)

type ServerConfig struct {
	App                 config.AppConfig
	EvolutionServers    []config.EvolutionServerConfig
	InstanceCreator     InstanceCreator
	PhoneNumbers        PhoneNumberStore
	Proxies             ProxyStore
	EvolutionStore      EvolutionServerStore
	MessageTemplates    MessageTemplateStore
	ConversationScripts ConversationScriptStore
	WarmingJobs         WarmingJobStore
	WarmingJobRunner    WarmingJobRunnerService
	ExecutionLogs       ExecutionLogStore
	EvolutionEvents     EvolutionEventStore
	EvolutionSync       EvolutionSyncService
	Observability       ObservabilityService
	StaleJobCleanup     StaleJobCleanupService
	AuthVerifier        auth.Verifier
}

type Server struct {
	cfg ServerConfig
	mux *http.ServeMux
}

type InstanceCreator interface {
	Create(ctx context.Context, params instance.CreateParams) (repository.Instance, error)
}

type PhoneNumberStore interface {
	Create(ctx context.Context, params repository.CreatePhoneNumberParams) (repository.PhoneNumber, error)
	List(ctx context.Context) ([]repository.PhoneNumber, error)
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
}

type ConversationScriptStore interface {
	Create(ctx context.Context, params conversation.CreateScriptParams) (conversation.ScriptWithSteps, error)
	List(ctx context.Context) ([]conversation.ScriptWithSteps, error)
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
		cfg: cfg,
		mux: http.NewServeMux(),
	}
	server.routes()
	return server
}

func (s *Server) Handler() http.Handler {
	return loggingMiddleware(authMiddleware(s.cfg.App, s.cfg.AuthVerifier, s.mux))
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	s.mux.HandleFunc("POST /api/v1/instances", s.handleCreateInstance)
	s.mux.HandleFunc("POST /api/v1/phone-numbers", s.handleCreatePhoneNumber)
	s.mux.HandleFunc("GET /api/v1/phone-numbers", s.handleListPhoneNumbers)
	s.mux.HandleFunc("POST /api/v1/proxies", s.handleCreateProxy)
	s.mux.HandleFunc("GET /api/v1/proxies", s.handleListProxies)
	s.mux.HandleFunc("POST /api/v1/evolution-servers", s.handleCreateEvolutionServer)
	s.mux.HandleFunc("GET /api/v1/evolution-servers", s.handleListEvolutionServers)
	s.mux.HandleFunc("POST /api/v1/message-templates", s.handleCreateMessageTemplate)
	s.mux.HandleFunc("GET /api/v1/message-templates", s.handleListMessageTemplates)
	s.mux.HandleFunc("POST /api/v1/conversation-scripts", s.handleCreateConversationScript)
	s.mux.HandleFunc("GET /api/v1/conversation-scripts", s.handleListConversationScripts)
	s.mux.HandleFunc("POST /api/v1/warming-jobs", s.handleCreateWarmingJob)
	s.mux.HandleFunc("GET /api/v1/warming-jobs", s.handleListWarmingJobs)
	s.mux.HandleFunc("POST /api/v1/warming-jobs/{id}/run-now", s.handleRunWarmingJobNow)
	s.mux.HandleFunc("POST /api/v1/warming-jobs/stale-cleanup", s.handleStaleCleanup)
	s.mux.HandleFunc("POST /api/v1/execution-logs", s.handleCreateExecutionLog)
	s.mux.HandleFunc("GET /api/v1/execution-logs", s.handleListExecutionLogs)
	s.mux.HandleFunc("GET /api/v1/metrics", s.handleMetrics)
	s.mux.HandleFunc("POST /api/v1/webhooks/evolution", s.handleEvolutionWebhook)
	s.mux.HandleFunc("POST /api/v1/webhooks/evolution/", s.handleEvolutionWebhook)
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

func authMiddleware(app config.AppConfig, verifier auth.Verifier, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !app.AuthEnabled || isPublicRoute(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		token, ok := bearerTokenFromHeader(r.Header.Get("Authorization"))
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
	case "/health", "/api/v1/health":
		return true
	}
	return path == "/api/v1/webhooks/evolution" || strings.HasPrefix(path, "/api/v1/webhooks/evolution/")
}

func bearerTokenFromHeader(header string) (string, bool) {
	parts := strings.SplitN(strings.TrimSpace(header), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return "", false
	}
	return strings.TrimSpace(parts[1]), true
}
