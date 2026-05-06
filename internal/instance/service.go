package instance

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"aquecedor-evolution/backend/internal/evolution"
	"aquecedor-evolution/backend/internal/repository"
)

var ErrNoEvolutionServer = errors.New("no enabled evolution server available")

type EvolutionServerStore interface {
	ListEnabled(ctx context.Context) ([]repository.EvolutionServer, error)
}

type ProxyStore interface {
	ListEnabled(ctx context.Context) ([]repository.Proxy, error)
}

type InstanceStore interface {
	Create(ctx context.Context, params repository.CreateInstanceParams) (repository.Instance, error)
}

type EvolutionInstanceCreator interface {
	CreateInstance(ctx context.Context, request evolution.CreateInstanceRequest) (evolution.CreateInstanceResponse, error)
}

type EvolutionFactory interface {
	New(server repository.EvolutionServer) EvolutionInstanceCreator
}

type SecretResolver interface {
	Resolve(secretName string) string
}

type StaticSecretResolver map[string]string

func (r StaticSecretResolver) Resolve(secretName string) string {
	return r[secretName]
}

type ServiceConfig struct {
	EvolutionServers EvolutionServerStore
	Proxies          ProxyStore
	Instances        InstanceStore
	EvolutionFactory EvolutionFactory
	SecretResolver   SecretResolver
	WebhookURL       string
}

type Service struct {
	evolutionServers EvolutionServerStore
	proxies          ProxyStore
	instances        InstanceStore
	evolutionFactory EvolutionFactory
	secretResolver   SecretResolver
	webhookURL       string
}

type CreateParams struct {
	PhoneNumberID string
	PhoneE164     string
	InstanceName  string
	TestRunID     string
}

func NewService(cfg ServiceConfig) Service {
	return Service{
		evolutionServers: cfg.EvolutionServers,
		proxies:          cfg.Proxies,
		instances:        cfg.Instances,
		evolutionFactory: cfg.EvolutionFactory,
		secretResolver:   cfg.SecretResolver,
		webhookURL:       strings.TrimRight(cfg.WebhookURL, "/"),
	}
}

func (s Service) Create(ctx context.Context, params CreateParams) (repository.Instance, error) {
	servers, err := s.evolutionServers.ListEnabled(ctx)
	if err != nil {
		return repository.Instance{}, fmt.Errorf("list evolution servers: %w", err)
	}
	if len(servers) == 0 {
		return repository.Instance{}, ErrNoEvolutionServer
	}
	server := servers[0]

	proxies, err := s.proxies.ListEnabled(ctx)
	if err != nil {
		return repository.Instance{}, fmt.Errorf("list proxies: %w", err)
	}

	var selectedProxy *repository.Proxy
	if len(proxies) > 0 {
		selectedProxy = &proxies[0]
	}

	request := evolution.CreateInstanceRequest{
		InstanceName:    params.InstanceName,
		Integration:     "WHATSAPP-BAILEYS",
		QRCode:          true,
		Number:          params.PhoneE164,
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

	creator := s.evolutionFactory.New(server)
	response, err := creator.CreateInstance(ctx, request)
	if err != nil {
		return repository.Instance{}, fmt.Errorf("create evolution instance: %w", err)
	}

	var proxyID *string
	if selectedProxy != nil {
		proxyID = &selectedProxy.ID
	}

	var instanceAPIKeySecretName *string
	if response.Hash.APIKey != "" {
		secretName := "EVOLUTION_INSTANCE_" + params.InstanceName + "_API_KEY"
		instanceAPIKeySecretName = &secretName
	}

	metadata := map[string]any{}
	if params.TestRunID != "" {
		metadata["testRunId"] = params.TestRunID
	}

	instance, err := s.instances.Create(ctx, repository.CreateInstanceParams{
		PhoneNumberID:            params.PhoneNumberID,
		EvolutionServerID:        server.ID,
		ProxyID:                  proxyID,
		InstanceName:             params.InstanceName,
		EvolutionInstanceID:      &response.Instance.InstanceName,
		InstanceAPIKeySecretName: instanceAPIKeySecretName,
		Status:                   "created",
		Metadata:                 metadata,
	})
	if err != nil {
		return repository.Instance{}, fmt.Errorf("persist instance: %w", err)
	}

	return instance, nil
}
