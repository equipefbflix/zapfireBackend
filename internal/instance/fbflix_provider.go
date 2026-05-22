package instance

import (
	"context"
	"fmt"

	"aquecedor-evolution/backend/internal/proxy/fbflix"
	"aquecedor-evolution/backend/internal/repository"
)

type FBFlixListClient interface {
	ListProxies(ctx context.Context) ([]fbflix.Proxy, error)
}

type FBFlixProvider struct {
	client FBFlixListClient
}

func NewFBFlixProvider(client FBFlixListClient) *FBFlixProvider {
	return &FBFlixProvider{client: client}
}

func (p *FBFlixProvider) AcquireProxy(ctx context.Context, params FBFlixAcquireParams) (repository.CreateProxyParams, error) {
	proxies, err := p.client.ListProxies(ctx)
	if err != nil {
		return repository.CreateProxyParams{}, fmt.Errorf("list proxies: %w", err)
	}
	for _, item := range proxies {
		if item.Host == "" || item.Port == 0 {
			continue
		}
		proxyParams := repository.CreateProxyParams{
			Name:     fmt.Sprintf("FBFlix %s", item.ID),
			Host:     item.Host,
			Port:     item.Port,
			Protocol: item.Protocol,
			Enabled:  item.Status == "" || item.Status == "active",
			Metadata: map[string]any{
				"source":   "fbflix",
				"fbflixId": item.ID,
				"country":  item.Country,
				"region":   item.Region,
			},
		}
		if item.Username != "" {
			proxyParams.Username = &item.Username
		}
		if item.Password != "" {
			passwordSecretName := "literal:" + item.Password
			proxyParams.PasswordSecretName = &passwordSecretName
		}
		return proxyParams, nil
	}
	return repository.CreateProxyParams{}, fmt.Errorf("no valid proxy returned by fbflix")
}
