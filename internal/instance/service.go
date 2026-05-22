package instance

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"aquecedor-evolution/backend/internal/evolution"
	"aquecedor-evolution/backend/internal/repository"
)

var ErrNoEvolutionServer = errors.New("no enabled evolution server available")

type EvolutionServerStore interface {
	ListEnabled(ctx context.Context) ([]repository.EvolutionServer, error)
	GetByID(ctx context.Context, id string) (repository.EvolutionServer, error)
}

type ProxyStore interface {
	ListEnabled(ctx context.Context) ([]repository.Proxy, error)
	Upsert(ctx context.Context, params repository.CreateProxyParams) (repository.Proxy, error)
	Create(ctx context.Context, params repository.CreateProxyParams) (repository.Proxy, error)
}

type FBFlixAcquireParams struct {
	InstanceName string
	TestRunID    string
}

type FBFlixProxyProvider interface {
	AcquireProxy(ctx context.Context, params FBFlixAcquireParams) (repository.CreateProxyParams, error)
}

type InstanceStore interface {
	Create(ctx context.Context, params repository.CreateInstanceParams) (repository.Instance, error)
	GetOpenByPhoneNumberID(ctx context.Context, phoneNumberID string) (repository.Instance, error)
	GetByID(ctx context.Context, id string) (repository.Instance, error)
	List(ctx context.Context) ([]repository.Instance, error)
	UpdateClassification(ctx context.Context, id string, params repository.UpdateInstanceClassificationParams) (repository.Instance, error)
	Delete(ctx context.Context, id string) error
}

type EvolutionInstanceCreator interface {
	CreateInstance(ctx context.Context, request evolution.CreateInstanceRequest) (evolution.CreateInstanceResponse, error)
	RestartInstance(ctx context.Context, instanceName string) error
	DeleteInstance(ctx context.Context, instanceName string) error
	ConnectInstance(ctx context.Context, instanceName, number string) (evolution.ConnectInstanceResponse, error)
	ConnectionState(ctx context.Context, instanceName string) (evolution.ConnectionStateResponse, error)
}

type EvolutionFactory interface {
	New(server repository.EvolutionServer) EvolutionInstanceCreator
	NewWithAPIKey(server repository.EvolutionServer, apiKey string) EvolutionInstanceCreator
}

type SecretResolver interface {
	Resolve(secretName string) string
}

type PhoneNumberStore interface {
	Create(ctx context.Context, params repository.CreatePhoneNumberParams) (repository.PhoneNumber, error)
}

type StaticSecretResolver map[string]string

func (r StaticSecretResolver) Resolve(secretName string) string {
	if value, ok := literalSecret(secretName); ok {
		return value
	}
	return r[secretName]
}

type ServiceConfig struct {
	EvolutionServers    EvolutionServerStore
	Proxies             ProxyStore
	Instances           InstanceStore
	EvolutionFactory    EvolutionFactory
	SecretResolver      SecretResolver
	WebhookURL          string
	PhoneNumbers        PhoneNumberStore
	FBFlixProxyProvider FBFlixProxyProvider
}

type Service struct {
	evolutionServers    EvolutionServerStore
	proxies             ProxyStore
	instances           InstanceStore
	evolutionFactory    EvolutionFactory
	secretResolver      SecretResolver
	webhookURL          string
	phoneNumbers        PhoneNumberStore
	fbflixProxyProvider FBFlixProxyProvider
}

type CreateParams struct {
	PhoneNumberID  string
	PhoneE164      string
	InstanceName   string
	Classification string
	TestRunID      string
	ManualProxy    *ManualProxyInput
}

type ManualProxyInput struct {
	Host     string
	Port     int
	Protocol string
	Username *string
	Password *string
}

func NewService(cfg ServiceConfig) Service {
	return Service{
		evolutionServers:    cfg.EvolutionServers,
		proxies:             cfg.Proxies,
		instances:           cfg.Instances,
		evolutionFactory:    cfg.EvolutionFactory,
		secretResolver:      cfg.SecretResolver,
		webhookURL:          strings.TrimRight(cfg.WebhookURL, "/"),
		phoneNumbers:        cfg.PhoneNumbers,
		fbflixProxyProvider: cfg.FBFlixProxyProvider,
	}
}

func (s Service) Create(ctx context.Context, params CreateParams) (repository.Instance, error) {
	servers, err := s.evolutionServers.ListEnabled(ctx)
	if err != nil {
		return repository.Instance{}, fmt.Errorf("list evolution servers: %w", err)
	}
	server, ok := s.selectEvolutionServer(servers)
	if !ok {
		return repository.Instance{}, ErrNoEvolutionServer
	}

	selectedProxy, err := s.resolveProxy(ctx, params)
	if err != nil {
		return repository.Instance{}, err
	}

	creator := s.evolutionFactory.New(server)
	baseName := strings.TrimSpace(params.InstanceName)
	if baseName == "" {
		baseName = "instancia"
	}
	baseName = fmt.Sprintf("%s_%d", baseName, time.Now().UnixMilli())
	var (
		response  evolution.CreateInstanceResponse
		createErr error
		request   evolution.CreateInstanceRequest
	)
	finalInstanceName := baseName
	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {
			finalInstanceName = fmt.Sprintf("%s_%d", baseName, attempt+1)
		}
		request = s.newCreateInstanceRequest(finalInstanceName, selectedProxy)
		response, createErr = creator.CreateInstance(ctx, request)
		if createErr == nil {
			break
		}
		slog.Info("instance create attempt failed", "attempt", attempt, "instanceName", finalInstanceName, "error", createErr, "isDuplicate", isEvolutionDuplicateInstanceError(createErr))
		if !isEvolutionDuplicateInstanceError(createErr) {
			return repository.Instance{}, fmt.Errorf("create evolution instance: %w", createErr)
		}
	}
	if createErr != nil {
		return repository.Instance{}, fmt.Errorf("create evolution instance: %w", createErr)
	}

	var proxyID *string
	if selectedProxy != nil {
		proxyID = &selectedProxy.ID
	}

	instanceToken := response.Hash.APIKey
	if strings.TrimSpace(instanceToken) == "" {
		instanceToken = request.Token
	}
	instanceAPIKeySecretName := stringPtr("literal:" + instanceToken)

	metadata := map[string]any{}
	if params.TestRunID != "" {
		metadata["testRunId"] = params.TestRunID
	}
	if classification := normalizeClassification(params.Classification); classification != "" {
		metadata["classification"] = classification
	}

	evolutionInstanceID := response.Instance.ID
	if strings.TrimSpace(evolutionInstanceID) == "" {
		evolutionInstanceID = response.Instance.InstanceName
	}

	phoneNumberID := params.PhoneNumberID
	if strings.TrimSpace(phoneNumberID) == "" && s.phoneNumbers != nil {
		fakeE164 := "0000000000000_" + finalInstanceName
		phone, err := s.phoneNumbers.Create(ctx, repository.CreatePhoneNumberParams{
			PhoneE164: fakeE164,
			Label:     finalInstanceName,
			Metadata:  metadata,
		})
		if err == nil {
			phoneNumberID = phone.ID
		}
	}

	instance, err := s.instances.Create(ctx, repository.CreateInstanceParams{
		PhoneNumberID:            phoneNumberID,
		EvolutionServerID:        server.ID,
		ProxyID:                  proxyID,
		InstanceName:             finalInstanceName,
		EvolutionInstanceID:      stringPtr(evolutionInstanceID),
		InstanceAPIKeySecretName: instanceAPIKeySecretName,
		Status:                   "created",
		Metadata:                 metadata,
	})
	if err != nil {
		return repository.Instance{}, fmt.Errorf("persist instance: %w", err)
	}

	return instance, nil
}

func (s Service) resolveProxy(ctx context.Context, params CreateParams) (*repository.Proxy, error) {
	if params.ManualProxy != nil {
		return s.persistManualProxy(ctx, *params.ManualProxy, params.TestRunID)
	}
	if s.fbflixProxyProvider != nil {
		proxy, err := s.acquireFBFlixProxy(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("acquire fbflix proxy: %w", err)
		}
		return proxy, nil
	}
	proxies, err := s.proxies.ListEnabled(ctx)
	if err != nil {
		return nil, fmt.Errorf("list proxies: %w", err)
	}
	selectedProxy, ok := s.selectProxy(proxies)
	if !ok {
		return nil, nil
	}
	return selectedProxy, nil
}

func (s Service) persistManualProxy(ctx context.Context, manual ManualProxyInput, testRunID string) (*repository.Proxy, error) {
	params := repository.CreateProxyParams{
		Name:     fmt.Sprintf("Manual %s:%d", manual.Host, manual.Port),
		Host:     strings.TrimSpace(manual.Host),
		Port:     manual.Port,
		Protocol: strings.ToLower(strings.TrimSpace(manual.Protocol)),
		Enabled:  true,
		Metadata: map[string]any{
			"source": "manual",
			"origin": "manual",
		},
	}
	if testRunID != "" {
		params.Metadata["testRunId"] = testRunID
	}
	if manual.Username != nil && strings.TrimSpace(*manual.Username) != "" {
		params.Username = manual.Username
	}
	if manual.Password != nil {
		passwordSecretName := "literal:" + *manual.Password
		params.PasswordSecretName = &passwordSecretName
	}
	proxy, err := s.proxies.Upsert(ctx, params)
	if err != nil {
		proxy, fallbackErr := s.createProxyWithoutUpsert(ctx, params, err)
		if fallbackErr != nil {
			return nil, fmt.Errorf("upsert manual proxy: %w", err)
		}
		return &proxy, nil
	}
	return &proxy, nil
}

func (s Service) acquireFBFlixProxy(ctx context.Context, params CreateParams) (*repository.Proxy, error) {
	proxyParams, err := s.fbflixProxyProvider.AcquireProxy(ctx, FBFlixAcquireParams{
		InstanceName: params.InstanceName,
		TestRunID:    params.TestRunID,
	})
	if err != nil {
		return nil, err
	}
	if proxyParams.Metadata == nil {
		proxyParams.Metadata = map[string]any{}
	}
	proxyParams.Metadata["origin"] = "automatic"
	if params.TestRunID != "" {
		proxyParams.Metadata["testRunId"] = params.TestRunID
	}
	proxy, err := s.proxies.Upsert(ctx, proxyParams)
	if err != nil {
		proxy, fallbackErr := s.createProxyWithoutUpsert(ctx, proxyParams, err)
		if fallbackErr != nil {
			return nil, fmt.Errorf("upsert fbflix proxy: %w", err)
		}
		return &proxy, nil
	}
	return &proxy, nil
}

func (s Service) createProxyWithoutUpsert(ctx context.Context, params repository.CreateProxyParams, upsertErr error) (repository.Proxy, error) {
	if !strings.Contains(strings.ToLower(upsertErr.Error()), "no unique or exclusion constraint") {
		return repository.Proxy{}, upsertErr
	}
	return s.proxies.Create(ctx, params)
}

func (s Service) UpdateClassification(ctx context.Context, id string, classification string) (repository.Instance, error) {
	classification = normalizeClassification(classification)
	if classification == "" {
		return repository.Instance{}, fmt.Errorf("invalid classification")
	}
	updated, err := s.instances.UpdateClassification(ctx, id, repository.UpdateInstanceClassificationParams{
		Classification: classification,
	})
	if err != nil {
		return repository.Instance{}, fmt.Errorf("update instance classification: %w", err)
	}
	return updated, nil
}

func (s Service) newCreateInstanceRequest(instanceName string, selectedProxy *repository.Proxy) evolution.CreateInstanceRequest {
	request := evolution.CreateInstanceRequest{
		InstanceName:    instanceName,
		Token:           newInstanceToken(instanceName),
		RejectCall:      true,
		GroupsIgnore:    true,
		AlwaysOnline:    true,
		ReadMessages:    true,
		ReadStatus:      true,
		SyncFullHistory: false,
	}
	if selectedProxy != nil {
		request.ProxyHost = selectedProxy.Host
		request.ProxyPort = strconv.Itoa(selectedProxy.Port)
		request.ProxyProtocol = selectedProxy.Protocol
		if selectedProxy.Username != nil {
			request.ProxyUsername = *selectedProxy.Username
		}
		if selectedProxy.PasswordSecretName != nil && s.secretResolver != nil {
			request.ProxyPassword = s.secretResolver.Resolve(*selectedProxy.PasswordSecretName)
		}
	}
	if s.webhookURL != "" {
		request.Webhook = &evolution.WebhookConfig{
			URL:      s.webhookURL,
			ByEvents: true,
			Base64:   false,
			Events: []string{
				"MESSAGES_UPSERT",
				"MESSAGES_UPDATE",
				"CONNECTION_UPDATE",
				"QRCODE_UPDATED",
			},
		}
	}
	return request
}

func (s Service) selectEvolutionServer(servers []repository.EvolutionServer) (repository.EvolutionServer, bool) {
	for _, server := range servers {
		if hasTestRunID(server.Metadata) {
			continue
		}
		if strings.TrimSpace(server.BaseURL) == "" {
			continue
		}
		if s.secretResolver != nil && strings.TrimSpace(server.APIKeySecretName) != "" {
			if strings.TrimSpace(s.secretResolver.Resolve(server.APIKeySecretName)) == "" {
				continue
			}
		}
		return server, true
	}
	return repository.EvolutionServer{}, false
}

func (s Service) selectProxy(proxies []repository.Proxy) (*repository.Proxy, bool) {
	for _, proxy := range proxies {
		if hasTestRunID(proxy.Metadata) {
			continue
		}
		if strings.TrimSpace(proxy.Host) == "" || proxy.Port <= 0 {
			continue
		}
		if proxy.PasswordSecretName != nil && s.secretResolver != nil {
			if strings.TrimSpace(s.secretResolver.Resolve(*proxy.PasswordSecretName)) == "" {
				continue
			}
		}
		selected := proxy
		return &selected, true
	}
	return nil, false
}

func hasTestRunID(metadata map[string]any) bool {
	if metadata == nil {
		return false
	}
	value, ok := metadata["testRunId"]
	if !ok {
		return false
	}
	text, ok := value.(string)
	return ok && strings.TrimSpace(text) != ""
}

func (s Service) Restart(ctx context.Context, phoneNumberID string) error {
	inst, err := s.instances.GetOpenByPhoneNumberID(ctx, phoneNumberID)
	if err != nil {
		return fmt.Errorf("find open instance: %w", err)
	}

	server, err := s.evolutionServers.GetByID(ctx, inst.EvolutionServerID)
	if err != nil {
		return fmt.Errorf("get evolution server: %w", err)
	}

	creator := s.evolutionFactory.NewWithAPIKey(server, s.instanceAPIKey(inst))
	return creator.RestartInstance(ctx, inst.InstanceName)
}

func (s Service) List(ctx context.Context) ([]repository.Instance, error) {
	return s.instances.List(ctx)
}

func (s Service) GetByID(ctx context.Context, id string) (repository.Instance, error) {
	return s.instances.GetByID(ctx, id)
}

func (s Service) Connect(ctx context.Context, id string) (evolution.ConnectInstanceResponse, error) {
	inst, err := s.instances.GetByID(ctx, id)
	if err != nil {
		return evolution.ConnectInstanceResponse{}, fmt.Errorf("get instance: %w", err)
	}
	server, err := s.evolutionServers.GetByID(ctx, inst.EvolutionServerID)
	if err != nil {
		return evolution.ConnectInstanceResponse{}, fmt.Errorf("get evolution server: %w", err)
	}
	creator := s.evolutionFactory.NewWithAPIKey(server, s.instanceAPIKey(inst))
	response, err := creator.ConnectInstance(ctx, inst.InstanceName, "")
	if err != nil {
		return evolution.ConnectInstanceResponse{}, fmt.Errorf("connect evolution instance: %w", err)
	}
	return response, nil
}

func (s Service) SyncState(ctx context.Context, id string) (repository.Instance, error) {
	inst, err := s.instances.GetByID(ctx, id)
	if err != nil {
		return repository.Instance{}, fmt.Errorf("get instance: %w", err)
	}
	server, err := s.evolutionServers.GetByID(ctx, inst.EvolutionServerID)
	if err != nil {
		return repository.Instance{}, fmt.Errorf("get evolution server: %w", err)
	}
	creator := s.evolutionFactory.NewWithAPIKey(server, s.instanceAPIKey(inst))
	state, err := creator.ConnectionState(ctx, inst.InstanceName)
	if err != nil {
		return repository.Instance{}, fmt.Errorf("fetch evolution connection state: %w", err)
	}
	slog.Info("evolution state fetched", "instanceName", inst.InstanceName, "state", state.Instance.State, "connected", state.Instance.Connected)
	if updater, ok := s.instances.(interface {
		UpdateConnectionStateByName(ctx context.Context, instanceName string, status string, lastError string) error
	}); ok {
		if err := updater.UpdateConnectionStateByName(ctx, inst.InstanceName, mapConnectionState(state.Instance.State), ""); err != nil {
			return repository.Instance{}, fmt.Errorf("update instance connection state: %w", err)
		}
	}
	return s.instances.GetByID(ctx, id)
}

func (s Service) RestartByID(ctx context.Context, id string) error {
	inst, err := s.instances.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get instance: %w", err)
	}
	server, err := s.evolutionServers.GetByID(ctx, inst.EvolutionServerID)
	if err != nil {
		return fmt.Errorf("get evolution server: %w", err)
	}
	creator := s.evolutionFactory.NewWithAPIKey(server, s.instanceAPIKey(inst))
	return creator.RestartInstance(ctx, inst.InstanceName)
}

func (s Service) DeleteByID(ctx context.Context, id string) error {
	inst, err := s.instances.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get instance: %w", err)
	}
	server, err := s.evolutionServers.GetByID(ctx, inst.EvolutionServerID)
	if err != nil {
		return fmt.Errorf("get evolution server: %w", err)
	}
	creator := s.evolutionFactory.New(server)
	target := inst.InstanceName
	if inst.EvolutionInstanceID != nil && *inst.EvolutionInstanceID != "" {
		target = *inst.EvolutionInstanceID
	}
	if err := creator.DeleteInstance(ctx, target); err != nil {
		if strings.Contains(err.Error(), "invalid UUID format") {
			state, stateErr := creator.ConnectionState(ctx, inst.InstanceName)
			if stateErr == nil && state.Instance.ID != "" {
				if err := creator.DeleteInstance(ctx, state.Instance.ID); err != nil {
					slog.Warn("failed to delete instance from evolution api by uuid", "error", err, "instanceName", inst.InstanceName, "uuid", state.Instance.ID)
				}
			}
		} else {
			slog.Warn("failed to delete instance from evolution api", "error", err, "instanceName", inst.InstanceName)
		}
	}
	if err := s.instances.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete instance from db: %w", err)
	}
	return nil
}

func (s Service) instanceAPIKey(inst repository.Instance) string {
	if inst.InstanceAPIKeySecretName != nil && s.secretResolver != nil {
		if value := strings.TrimSpace(s.secretResolver.Resolve(*inst.InstanceAPIKeySecretName)); value != "" {
			return value
		}
	}
	return ""
}

func newInstanceToken(instanceName string) string {
	random := make([]byte, 12)
	if _, err := rand.Read(random); err != nil {
		return "inst_" + sanitizeTokenPart(instanceName)
	}
	return "inst_" + sanitizeTokenPart(instanceName) + "_" + hex.EncodeToString(random)
}

func sanitizeTokenPart(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "_")
	value = strings.ReplaceAll(value, "-", "_")
	builder := strings.Builder{}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			builder.WriteRune(r)
		}
	}
	if builder.Len() == 0 {
		return "instance"
	}
	return builder.String()
}

func stringPtr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func mapConnectionState(state string) string {
	switch strings.ToLower(state) {
	case "open", "opened":
		return "open"
	case "close", "closed", "disconnected":
		return "close"
	case "connecting":
		return "connecting"
	case "failed", "error":
		return "failed"
	case "paused":
		return "paused"
	default:
		return strings.ToLower(state)
	}
}

func isEvolutionDuplicateInstanceError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	if strings.Contains(errStr, "instance already exists") {
		return true
	}
	var httpErr *evolution.HTTPError
	if errors.As(err, &httpErr) && strings.Contains(strings.ToLower(httpErr.Body), "instance already exists") {
		return true
	}
	return false
}

func normalizeClassification(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "internal":
		return "internal"
	case "external":
		return "external"
	default:
		return ""
	}
}
