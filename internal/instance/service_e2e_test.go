//go:build e2e

package instance

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"aquecedor-evolution/backend/internal/config"
	"aquecedor-evolution/backend/internal/database"
	"aquecedor-evolution/backend/internal/evolution"
	"aquecedor-evolution/backend/internal/proxy/fbflix"
	"aquecedor-evolution/backend/internal/repository"
)

type emptyProxyStore struct{}

func (emptyProxyStore) ListEnabled(ctx context.Context) ([]repository.Proxy, error) {
	return nil, nil
}

func (emptyProxyStore) Upsert(ctx context.Context, params repository.CreateProxyParams) (repository.Proxy, error) {
	return repository.Proxy{}, fmt.Errorf("upsert not supported")
}

func (emptyProxyStore) Create(ctx context.Context, params repository.CreateProxyParams) (repository.Proxy, error) {
	return repository.Proxy{}, fmt.Errorf("create not supported")
}

func newRealFBFlixClient(t *testing.T) *fbflix.Client {
	t.Helper()

	fbflixToken := os.Getenv("FBFLIX_B2B_TOKEN")
	if fbflixToken == "" {
		t.Fatal("FBFLIX_B2B_TOKEN is required")
	}
	fbflixBaseURL := os.Getenv("FBFLIX_API_URL")
	if fbflixBaseURL == "" {
		fbflixBaseURL = "https://mxnlerkeygfvdnznoxld.supabase.co/functions/v1/proxyfbflix-api"
	}
	return fbflix.NewClient(fbflix.Config{
		BaseURL: fbflixBaseURL,
		Token:   fbflixToken,
		Timeout: 30 * time.Second,
	})
}

func TestServiceCreateRealEvolutionInstanceWithFBFlixProxyE2E(t *testing.T) {
	if os.Getenv("ENABLE_REAL_TESTS") != "true" {
		t.Skip("set ENABLE_REAL_TESTS=true to run real Evolution E2E tests")
	}

	baseURL := firstEnv("EVOLUTION_TEST_BASE_URL", "SERVER_URL")
	apiKey := firstEnv("EVOLUTION_TEST_API_KEY", "AUTHENTICATION_API_KEY")
	if baseURL == "" {
		t.Fatal("EVOLUTION_TEST_BASE_URL or SERVER_URL is required")
	}
	if apiKey == "" {
		t.Fatal("EVOLUTION_TEST_API_KEY or AUTHENTICATION_API_KEY is required")
	}

	testRunID := os.Getenv("TEST_RUN_ID")
	if testRunID == "" {
		testRunID = "instance-fbflix-e2e-" + time.Now().UTC().Format("20060102T150405")
	}
	instanceName := "codex_fbflix_e2e_" + time.Now().UTC().Format("20060102t150405")
	phoneE164 := "+55119" + time.Now().UTC().Format("150405") + "98"
	phoneNumber := phoneE164[1:]

	dbConfig, err := config.LoadDatabaseConfig()
	if err != nil {
		t.Fatalf("LoadDatabaseConfig() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	pool, err := database.Open(ctx, dbConfig)
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	defer pool.Close()

	executor := repository.NewPgxExecutor(pool)
	phones := repository.NewPhoneNumberRepository(executor)
	servers := repository.NewEvolutionServerRepository(executor)
	proxies := repository.NewProxyRepository(executor)
	instances := repository.NewInstanceRepository(executor)

	evolutionClient := evolution.NewClient(evolution.Config{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Timeout: 60 * time.Second,
	})

	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cleanupCancel()

		_ = evolutionClient.DeleteInstance(cleanupCtx, instanceName)
		_, _ = instances.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = phones.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = proxies.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = servers.DeleteByTestRunID(cleanupCtx, testRunID)
	}()

	fbflixClient := newRealFBFlixClient(t)
	realProxies, err := fbflixClient.ListProxies(ctx)
	if err != nil {
		t.Fatalf("fbflix.ListProxies() error = %v", err)
	}
	if len(realProxies) == 0 {
		t.Fatal("fbflix.ListProxies() returned zero proxies")
	}
	realProxy := realProxies[0]
	if realProxy.Password == "" {
		t.Fatal("fbflix proxy password is empty")
	}
	passwordSecretName := "literal:" + realProxy.Password

	phone, err := phones.Create(ctx, repository.CreatePhoneNumberParams{
		PhoneE164: phoneE164,
		Label:     "e2e_fbflix_instance_phone_" + testRunID,
		Metadata:  map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("phones.Create() error = %v", err)
	}

	server, err := servers.Create(ctx, repository.CreateEvolutionServerParams{
		Name:              "000_fbflix_e2e_" + testRunID,
		BaseURL:           baseURL,
		APIKeySecretName:  "EVOLUTION_TEST_API_KEY",
		Enabled:           true,
		Weight:            1,
		MaxConcurrentJobs: 1,
		Metadata:          map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("servers.Create() error = %v", err)
	}
	_ = server

	proxy, err := proxies.Create(ctx, repository.CreateProxyParams{
		Name:               "000_fbflix_e2e_" + testRunID,
		Host:               realProxy.Host,
		Port:               realProxy.Port,
		Protocol:           realProxy.Protocol,
		Username:           &realProxy.Username,
		PasswordSecretName: &passwordSecretName,
		Enabled:            true,
		MaxInstances:       nil,
		Metadata: map[string]any{
			"testRunId": testRunID,
			"source":    "fbflix",
			"fbflixId":  realProxy.ID,
		},
	})
	if err != nil {
		t.Fatalf("proxies.Create() error = %v", err)
	}

	t.Setenv("EVOLUTION_TEST_API_KEY", apiKey)
	service := NewService(ServiceConfig{
		EvolutionServers: servers,
		Proxies:          proxies,
		Instances:        instances,
		EvolutionFactory: EvolutionClientFactory{
			SecretResolver: EnvSecretResolver{},
			Timeout:        60 * time.Second,
		},
		SecretResolver: EnvSecretResolver{},
	})

	created, err := service.Create(ctx, CreateParams{
		PhoneNumberID: phone.ID,
		PhoneE164:     phoneNumber,
		InstanceName:  instanceName,
		TestRunID:     testRunID,
	})
	if err != nil {
		t.Fatalf("service.Create() error = %v", err)
	}
	if created.ProxyID == nil || *created.ProxyID != proxy.ID {
		t.Fatalf("created.ProxyID = %v, want %s", created.ProxyID, proxy.ID)
	}

	items, err := evolutionClient.FetchInstances(ctx, instanceName)
	if err != nil {
		t.Fatalf("FetchInstances() error = %v", err)
	}
	if len(items) == 0 {
		t.Fatal("FetchInstances() returned zero items for created instance")
	}
}

func TestServiceCreateRealEvolutionInstanceWithManualProxyE2E(t *testing.T) {
	if os.Getenv("ENABLE_REAL_TESTS") != "true" {
		t.Skip("set ENABLE_REAL_TESTS=true to run real Evolution E2E tests")
	}

	baseURL := firstEnv("EVOLUTION_TEST_BASE_URL", "SERVER_URL")
	apiKey := firstEnv("EVOLUTION_TEST_API_KEY", "AUTHENTICATION_API_KEY")
	if baseURL == "" {
		t.Fatal("EVOLUTION_TEST_BASE_URL or SERVER_URL is required")
	}
	if apiKey == "" {
		t.Fatal("EVOLUTION_TEST_API_KEY or AUTHENTICATION_API_KEY is required")
	}

	testRunID := os.Getenv("TEST_RUN_ID")
	if testRunID == "" {
		testRunID = "instance-manual-proxy-e2e-" + time.Now().UTC().Format("20060102T150405")
	}
	instanceName := "codex_manual_proxy_e2e_" + time.Now().UTC().Format("20060102t150405")
	cleanupInstanceName := instanceName
	phoneE164 := "+55119" + time.Now().UTC().Format("150405") + "96"
	phoneNumber := phoneE164[1:]

	dbConfig, err := config.LoadDatabaseConfig()
	if err != nil {
		t.Fatalf("LoadDatabaseConfig() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	pool, err := database.Open(ctx, dbConfig)
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	defer pool.Close()

	executor := repository.NewPgxExecutor(pool)
	phones := repository.NewPhoneNumberRepository(executor)
	servers := repository.NewEvolutionServerRepository(executor)
	proxies := repository.NewProxyRepository(executor)
	instances := repository.NewInstanceRepository(executor)

	evolutionClient := evolution.NewClient(evolution.Config{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Timeout: 60 * time.Second,
	})

	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cleanupCancel()

		_ = evolutionClient.DeleteInstance(cleanupCtx, cleanupInstanceName)
		_, _ = instances.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = phones.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = proxies.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = servers.DeleteByTestRunID(cleanupCtx, testRunID)
	}()

	fbflixClient := newRealFBFlixClient(t)
	realProxies, err := fbflixClient.ListProxies(ctx)
	if err != nil {
		t.Fatalf("fbflix.ListProxies() error = %v", err)
	}
	if len(realProxies) == 0 {
		t.Fatal("fbflix.ListProxies() returned zero proxies")
	}
	realProxy := realProxies[0]
	if realProxy.Password == "" {
		t.Fatal("fbflix proxy password is empty")
	}

	phone, err := phones.Create(ctx, repository.CreatePhoneNumberParams{
		PhoneE164: phoneE164,
		Label:     "e2e_manual_proxy_phone_" + testRunID,
		Metadata:  map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("phones.Create() error = %v", err)
	}

	_, err = servers.Create(ctx, repository.CreateEvolutionServerParams{
		Name:              "000_manual_proxy_e2e_" + testRunID,
		BaseURL:           baseURL,
		APIKeySecretName:  "EVOLUTION_TEST_API_KEY",
		Enabled:           true,
		Weight:            1,
		MaxConcurrentJobs: 1,
		Metadata:          map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("servers.Create() error = %v", err)
	}

	t.Setenv("EVOLUTION_TEST_API_KEY", apiKey)
	service := NewService(ServiceConfig{
		EvolutionServers: servers,
		Proxies:          proxies,
		Instances:        instances,
		EvolutionFactory: EvolutionClientFactory{
			SecretResolver: EnvSecretResolver{},
			Timeout:        60 * time.Second,
		},
		SecretResolver: EnvSecretResolver{},
	})

	created, err := service.Create(ctx, CreateParams{
		PhoneNumberID: phone.ID,
		PhoneE164:     phoneNumber,
		InstanceName:  instanceName,
		TestRunID:     testRunID,
		ManualProxy: &ManualProxyInput{
			Host:     realProxy.Host,
			Port:     realProxy.Port,
			Protocol: realProxy.Protocol,
			Username: &realProxy.Username,
			Password: &realProxy.Password,
		},
	})
	if err != nil {
		t.Fatalf("service.Create() error = %v", err)
	}
	if created.ProxyID == nil {
		t.Fatal("created.ProxyID is nil")
	}
	cleanupInstanceName = created.InstanceName

	persistedProxy, err := proxies.List(ctx)
	if err != nil {
		t.Fatalf("proxies.List() error = %v", err)
	}
	found := false
	for _, proxy := range persistedProxy {
		if proxy.ID == *created.ProxyID {
			found = true
			if proxy.Host != realProxy.Host || proxy.Port != realProxy.Port {
				t.Fatalf("persisted proxy = %s:%d, want %s:%d", proxy.Host, proxy.Port, realProxy.Host, realProxy.Port)
			}
			break
		}
	}
	if !found {
		t.Fatalf("proxy %s not found in repository", *created.ProxyID)
	}

	items, err := evolutionClient.FetchInstances(ctx, created.InstanceName)
	if err != nil {
		t.Fatalf("FetchInstances() error = %v", err)
	}
	if len(items) == 0 {
		t.Fatal("FetchInstances() returned zero items for created instance")
	}
}

func TestServiceCreateRealEvolutionInstanceWithFBFlixFallbackE2E(t *testing.T) {
	if os.Getenv("ENABLE_REAL_TESTS") != "true" {
		t.Skip("set ENABLE_REAL_TESTS=true to run real Evolution E2E tests")
	}

	baseURL := firstEnv("EVOLUTION_TEST_BASE_URL", "SERVER_URL")
	apiKey := firstEnv("EVOLUTION_TEST_API_KEY", "AUTHENTICATION_API_KEY")
	if baseURL == "" {
		t.Fatal("EVOLUTION_TEST_BASE_URL or SERVER_URL is required")
	}
	if apiKey == "" {
		t.Fatal("EVOLUTION_TEST_API_KEY or AUTHENTICATION_API_KEY is required")
	}

	testRunID := os.Getenv("TEST_RUN_ID")
	if testRunID == "" {
		testRunID = "instance-fbflix-fallback-e2e-" + time.Now().UTC().Format("20060102T150405")
	}
	instanceName := "codex_fbflix_fallback_e2e_" + time.Now().UTC().Format("20060102t150405")
	cleanupInstanceName := instanceName
	phoneE164 := "+55119" + time.Now().UTC().Format("150405") + "95"
	phoneNumber := phoneE164[1:]

	dbConfig, err := config.LoadDatabaseConfig()
	if err != nil {
		t.Fatalf("LoadDatabaseConfig() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	pool, err := database.Open(ctx, dbConfig)
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	defer pool.Close()

	executor := repository.NewPgxExecutor(pool)
	phones := repository.NewPhoneNumberRepository(executor)
	servers := repository.NewEvolutionServerRepository(executor)
	proxies := repository.NewProxyRepository(executor)
	instances := repository.NewInstanceRepository(executor)

	evolutionClient := evolution.NewClient(evolution.Config{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Timeout: 60 * time.Second,
	})

	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cleanupCancel()

		_ = evolutionClient.DeleteInstance(cleanupCtx, cleanupInstanceName)
		_, _ = instances.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = phones.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = proxies.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = servers.DeleteByTestRunID(cleanupCtx, testRunID)
	}()

	phone, err := phones.Create(ctx, repository.CreatePhoneNumberParams{
		PhoneE164: phoneE164,
		Label:     "e2e_fbflix_fallback_phone_" + testRunID,
		Metadata:  map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("phones.Create() error = %v", err)
	}

	_, err = servers.Create(ctx, repository.CreateEvolutionServerParams{
		Name:              "000_fbflix_fallback_e2e_" + testRunID,
		BaseURL:           baseURL,
		APIKeySecretName:  "EVOLUTION_TEST_API_KEY",
		Enabled:           true,
		Weight:            1,
		MaxConcurrentJobs: 1,
		Metadata:          map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("servers.Create() error = %v", err)
	}

	fbflixClient := newRealFBFlixClient(t)

	t.Setenv("EVOLUTION_TEST_API_KEY", apiKey)
	service := NewService(ServiceConfig{
		EvolutionServers: servers,
		Proxies:          proxies,
		Instances:        instances,
		EvolutionFactory: EvolutionClientFactory{
			SecretResolver: EnvSecretResolver{},
			Timeout:        60 * time.Second,
		},
		SecretResolver:      EnvSecretResolver{},
		FBFlixProxyProvider: NewFBFlixProvider(fbflixClient),
	})

	created, err := service.Create(ctx, CreateParams{
		PhoneNumberID: phone.ID,
		PhoneE164:     phoneNumber,
		InstanceName:  instanceName,
		TestRunID:     testRunID,
	})
	if err != nil {
		t.Fatalf("service.Create() error = %v", err)
	}
	if created.ProxyID == nil {
		t.Fatal("created.ProxyID is nil")
	}
	cleanupInstanceName = created.InstanceName

	persistedProxy, err := proxies.List(ctx)
	if err != nil {
		t.Fatalf("proxies.List() error = %v", err)
	}
	found := false
	for _, proxy := range persistedProxy {
		if proxy.ID == *created.ProxyID {
			found = true
			if proxy.Metadata["source"] != "fbflix" {
				t.Fatalf("proxy.Metadata[source] = %v", proxy.Metadata["source"])
			}
			break
		}
	}
	if !found {
		t.Fatalf("proxy %s not found in repository", *created.ProxyID)
	}

	items, err := evolutionClient.FetchInstances(ctx, created.InstanceName)
	if err != nil {
		t.Fatalf("FetchInstances() error = %v", err)
	}
	if len(items) == 0 {
		t.Fatal("FetchInstances() returned zero items for created instance")
	}
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return ""
}

func TestServiceCreateRealEvolutionInstanceE2E(t *testing.T) {
	if os.Getenv("ENABLE_REAL_TESTS") != "true" {
		t.Skip("set ENABLE_REAL_TESTS=true to run real Evolution E2E tests")
	}

	baseURL := os.Getenv("EVOLUTION_TEST_BASE_URL")
	apiKey := os.Getenv("EVOLUTION_TEST_API_KEY")
	if baseURL == "" {
		t.Fatal("EVOLUTION_TEST_BASE_URL is required")
	}
	if apiKey == "" {
		t.Fatal("EVOLUTION_TEST_API_KEY is required")
	}

	testRunID := os.Getenv("TEST_RUN_ID")
	if testRunID == "" {
		testRunID = "instance-e2e-" + time.Now().UTC().Format("20060102T150405")
	}
	instanceName := "codex_e2e_" + time.Now().UTC().Format("20060102t150405")

	dbConfig, err := config.LoadDatabaseConfig()
	if err != nil {
		t.Fatalf("LoadDatabaseConfig() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := database.Open(ctx, dbConfig)
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	defer pool.Close()

	executor := repository.NewPgxExecutor(pool)
	phones := repository.NewPhoneNumberRepository(executor)
	servers := repository.NewEvolutionServerRepository(executor)
	instances := repository.NewInstanceRepository(executor)

	client := evolution.NewClient(evolution.Config{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Timeout: 60 * time.Second,
	})

	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cleanupCancel()

		_, _ = instances.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = phones.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = servers.DeleteByTestRunID(cleanupCtx, testRunID)
		_ = client.DeleteInstance(cleanupCtx, instanceName)
	}()

	phone, err := phones.Create(ctx, repository.CreatePhoneNumberParams{
		PhoneE164: "+5511999999999",
		Label:     "e2e_instance_phone_" + testRunID,
		Metadata:  map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("phones.Create() error = %v", err)
	}

	server, err := servers.Create(ctx, repository.CreateEvolutionServerParams{
		Name:              "000_e2e_" + testRunID,
		BaseURL:           baseURL,
		APIKeySecretName:  "EVOLUTION_TEST_API_KEY",
		Enabled:           true,
		Weight:            1,
		MaxConcurrentJobs: 1,
		Metadata:          map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("servers.Create() error = %v", err)
	}

	service := NewService(ServiceConfig{
		EvolutionServers: servers,
		Proxies:          emptyProxyStore{},
		Instances:        instances,
		EvolutionFactory: EvolutionClientFactory{
			SecretResolver: EnvSecretResolver{},
			Timeout:        60 * time.Second,
		},
		SecretResolver: EnvSecretResolver{},
	})

	created, err := service.Create(ctx, CreateParams{
		PhoneNumberID: phone.ID,
		PhoneE164:     "5511999999999",
		InstanceName:  instanceName,
		TestRunID:     testRunID,
	})
	if err != nil {
		t.Fatalf("service.Create() error = %v", err)
	}
	if created.ID == "" {
		t.Fatal("created.ID is empty")
	}
	if created.EvolutionServerID != server.ID {
		t.Fatalf("created.EvolutionServerID = %q", created.EvolutionServerID)
	}

	items, err := client.FetchInstances(ctx, instanceName)
	if err != nil {
		t.Fatalf("FetchInstances() error = %v", err)
	}
	if len(items) == 0 {
		t.Fatal("FetchInstances() returned zero items for created instance")
	}

	if err := client.DeleteInstance(ctx, instanceName); err != nil {
		t.Fatalf("DeleteInstance() error = %v", err)
	}

	time.Sleep(2 * time.Second)

	items, err = client.FetchInstances(ctx, instanceName)
	if err != nil {
		httpErr, ok := err.(*evolution.HTTPError)
		if !ok || httpErr.StatusCode != 404 {
			t.Fatalf("FetchInstances() after delete error = %v", err)
		}
		return
	}
	if len(items) != 0 {
		t.Fatalf("instance still exists after delete: %s", fmt.Sprintf("%+v", items))
	}
}

func TestServiceSyncStateRealEvolutionInstanceE2E(t *testing.T) {
	if os.Getenv("ENABLE_REAL_TESTS") != "true" {
		t.Skip("set ENABLE_REAL_TESTS=true to run real Evolution E2E tests")
	}

	baseURL := firstEnv("EVOLUTION_TEST_BASE_URL", "SERVER_URL")
	apiKey := firstEnv("EVOLUTION_TEST_API_KEY", "AUTHENTICATION_API_KEY")
	if baseURL == "" {
		t.Fatal("EVOLUTION_TEST_BASE_URL or SERVER_URL is required")
	}
	if apiKey == "" {
		t.Fatal("EVOLUTION_TEST_API_KEY or AUTHENTICATION_API_KEY is required")
	}

	testRunID := os.Getenv("TEST_RUN_ID")
	if testRunID == "" {
		testRunID = "instance-sync-e2e-" + time.Now().UTC().Format("20060102T150405")
	}
	instanceName := "codex_sync_e2e_" + time.Now().UTC().Format("20060102t150405")
	phoneE164 := "+55119" + time.Now().UTC().Format("150405") + "97"

	dbConfig, err := config.LoadDatabaseConfig()
	if err != nil {
		t.Fatalf("LoadDatabaseConfig() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	pool, err := database.Open(ctx, dbConfig)
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	defer pool.Close()

	executor := repository.NewPgxExecutor(pool)
	phones := repository.NewPhoneNumberRepository(executor)
	servers := repository.NewEvolutionServerRepository(executor)
	instances := repository.NewInstanceRepository(executor)

	client := evolution.NewClient(evolution.Config{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Timeout: 60 * time.Second,
	})

	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cleanupCancel()

		_ = client.DeleteInstance(cleanupCtx, instanceName)
		_, _ = instances.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = phones.DeleteByTestRunID(cleanupCtx, testRunID)
		_, _ = servers.DeleteByTestRunID(cleanupCtx, testRunID)
	}()

	phone, err := phones.Create(ctx, repository.CreatePhoneNumberParams{
		PhoneE164: phoneE164,
		Label:     "e2e_sync_instance_phone_" + testRunID,
		Metadata:  map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("phones.Create() error = %v", err)
	}

	_, err = servers.Create(ctx, repository.CreateEvolutionServerParams{
		Name:              "000_sync_e2e_" + testRunID,
		BaseURL:           baseURL,
		APIKeySecretName:  "EVOLUTION_TEST_API_KEY",
		Enabled:           true,
		Weight:            1,
		MaxConcurrentJobs: 1,
		Metadata:          map[string]any{"testRunId": testRunID},
	})
	if err != nil {
		t.Fatalf("servers.Create() error = %v", err)
	}

	t.Setenv("EVOLUTION_TEST_API_KEY", apiKey)
	service := NewService(ServiceConfig{
		EvolutionServers: servers,
		Proxies:          emptyProxyStore{},
		Instances:        instances,
		EvolutionFactory: EvolutionClientFactory{
			SecretResolver: EnvSecretResolver{},
			Timeout:        60 * time.Second,
		},
		SecretResolver: EnvSecretResolver{},
	})

	created, err := service.Create(ctx, CreateParams{
		PhoneNumberID: phone.ID,
		PhoneE164:     phone.PhoneE164[1:],
		InstanceName:  instanceName,
		TestRunID:     testRunID,
	})
	if err != nil {
		t.Fatalf("service.Create() error = %v", err)
	}

	synced, err := service.SyncState(ctx, created.ID)
	if err != nil {
		t.Fatalf("service.SyncState() error = %v", err)
	}
	if synced.ID != created.ID {
		t.Fatalf("synced.ID = %q, want %q", synced.ID, created.ID)
	}
	if synced.Status == "" {
		t.Fatal("synced.Status is empty")
	}

	refetched, err := instances.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("instances.GetByID() error = %v", err)
	}
	if refetched.Status != synced.Status {
		t.Fatalf("refetched.Status = %q, want %q", refetched.Status, synced.Status)
	}
}
