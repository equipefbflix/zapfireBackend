package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"aquecedor-evolution/backend/internal/auth"
	"aquecedor-evolution/backend/internal/config"
	"aquecedor-evolution/backend/internal/conversation"
	"aquecedor-evolution/backend/internal/conversationloop"
	"aquecedor-evolution/backend/internal/database"
	"aquecedor-evolution/backend/internal/evolutionsync"
	"aquecedor-evolution/backend/internal/httpserver"
	"aquecedor-evolution/backend/internal/instance"
	"aquecedor-evolution/backend/internal/observability"
	"aquecedor-evolution/backend/internal/planner"
	"aquecedor-evolution/backend/internal/proxy/fbflix"
	"aquecedor-evolution/backend/internal/repository"
	"aquecedor-evolution/backend/internal/runner"
	"aquecedor-evolution/backend/internal/warmingscore"
)

func main() {
	if err := run(); err != nil {
		slog.Error("api stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	appConfig, err := config.LoadAppConfig()
	if err != nil {
		return err
	}

	evolutionServers, err := config.LoadEvolutionServers()
	if err != nil {
		return err
	}

	serverConfig := httpserver.ServerConfig{
		App:              appConfig,
		EvolutionServers: evolutionServers,
	}
	if appConfig.AuthEnabled {
		serverConfig.AuthVerifier = auth.NewSupabaseVerifier(auth.SupabaseVerifierConfig{
			Issuer:  appConfig.SupabaseURL + "/auth/v1",
			JWKSURL: appConfig.SupabaseURL + "/auth/v1/.well-known/jwks.json",
		})
	}

	if os.Getenv("DATABASE_URL") != "" {
		databaseConfig, err := config.LoadDatabaseConfig()
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), databaseConfig.ConnectTimeout)
		defer cancel()

		pool, err := database.Open(ctx, databaseConfig)
		if err != nil {
			return err
		}
		defer pool.Close()

		executor := repository.NewPgxExecutor(pool)
		phoneNumbers := repository.NewPhoneNumberRepository(executor)
		proxies := repository.NewProxyRepository(executor)
		evolutionServersRepo := repository.NewEvolutionServerRepository(executor)
		messageTemplates := repository.NewMessageTemplateRepository(executor)
		conversationScripts := repository.NewConversationScriptRepository(executor)
		conversationSteps := repository.NewConversationStepRepository(executor)
		warmingJobs := repository.NewWarmingJobRepository(executor)
		executionLogs := repository.NewExecutionLogRepository(executor)
		evolutionEvents := repository.NewEvolutionEventRepository(executor)
		instances := repository.NewInstanceRepository(executor)
		observabilityConfig := config.LoadObservabilityConfig()
		serverConfig.InstanceCreator = instance.NewService(instance.ServiceConfig{
			EvolutionServers: evolutionServersRepo,
			Proxies:          proxies,
			Instances:        instances,
			EvolutionFactory: instance.EvolutionClientFactory{
				SecretResolver: instance.EnvSecretResolver{},
				Timeout:        appConfig.EvolutionTimeout,
			},
			SecretResolver: instance.EnvSecretResolver{},
			WebhookURL:     strings.TrimRight(appConfig.PublicURL, "/") + "/api/v1/webhooks/evolution",
		})
		plannerConfig := config.LoadPlannerConfig()
		instanceExecutors := runner.NewInstanceExecutorFactory(
			evolutionServersRepo,
			instance.EnvSecretResolver{},
			runner.DefaultStepClientFactory{Timeout: appConfig.EvolutionTimeout},
		)
		concurrencyGate := runner.NewMaxConcurrencyGate(
			warmingJobs,
			plannerConfig.MaxRunningJobsPerPair,
			plannerConfig.MaxRunningJobsPerEvolutionServer,
		)
		serverConfig.WarmingJobRunner = runner.NewWarmingJobRunner(
			warmingJobs,
			conversationSteps,
			instances,
			instanceExecutors,
			executionLogs,
			concurrencyGate,
		)
		serverConfig.PhoneNumbers = phoneNumbers
		serverConfig.Proxies = proxies
		serverConfig.EvolutionStore = evolutionServersRepo
		serverConfig.MessageTemplates = messageTemplates
		serverConfig.ConversationScripts = conversation.NewService(conversationScripts, conversationSteps)
		serverConfig.WarmingJobs = warmingJobs
		serverConfig.ExecutionLogs = executionLogs
		serverConfig.EvolutionEvents = evolutionEvents
		scoreService := warmingscore.NewService(config.LoadWarmingConfig(), executionLogs, phoneNumbers)
		jobPlanner := planner.NewService(plannerConfig, conversationScripts, warmingJobs, nil)
		inboundLoop := conversationloop.NewService(plannerConfig, instances, phoneNumbers, jobPlanner, warmingJobs, warmingJobs, nil)
		syncService := evolutionsync.NewService(instances, executionLogs, evolutionEvents, scoreService, inboundLoop)
		serverConfig.EvolutionSync = syncService
		serverConfig.Observability = observability.NewService(observabilityConfig, warmingJobs, executionLogs, evolutionEvents)
		serverConfig.StaleJobCleanup = observability.NewStaleCleanupService(observabilityConfig, warmingJobs)
		fbflixToken := os.Getenv("FBFLIX_B2B_TOKEN")
		if fbflixToken != "" {
			fbflixBaseURL := os.Getenv("FBFLIX_API_URL")
			if fbflixBaseURL == "" {
				fbflixBaseURL = "https://mxnlerkeygfvdnznoxld.supabase.co/functions/v1/proxyfbflix-api"
			}
			fbflixClient := fbflix.NewClient(fbflix.Config{
				BaseURL: fbflixBaseURL,
				Token:   fbflixToken,
				Timeout: appConfig.EvolutionTimeout,
			})
			serverConfig.FBFlixSync = fbflix.NewSyncService(fbflixClient, proxies)
		}
	}

	server := httpserver.NewServer(serverConfig)

	httpServer := &http.Server{
		Addr:              ":" + appConfig.Port,
		Handler:           server.Handler(),
		ReadHeaderTimeout: config.DefaultReadHeaderTimeout,
	}

	slog.Info("api listening", "addr", httpServer.Addr, "env", appConfig.Env)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}
