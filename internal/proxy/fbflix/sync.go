package fbflix

import (
	"context"
	"fmt"

	"aquecedor-evolution/backend/internal/repository"
)

type ProxyClient interface {
	ListProxies(ctx context.Context) ([]Proxy, error)
}

type ProxyStore interface {
	Upsert(ctx context.Context, params repository.CreateProxyParams) (repository.Proxy, error)
}

type SyncService struct {
	client ProxyClient
	repo   ProxyStore
}

func NewSyncService(client ProxyClient, repo ProxyStore) *SyncService {
	return &SyncService{
		client: client,
		repo:   repo,
	}
}

func (s *SyncService) Sync(ctx context.Context) (int, error) {
	proxies, err := s.client.ListProxies(ctx)
	if err != nil {
		return 0, fmt.Errorf("list fbflix proxies: %w", err)
	}

	count := 0
	for _, p := range proxies {
		params := repository.CreateProxyParams{
			Name:     fmt.Sprintf("FBFlix %s", p.ID),
			Host:     p.Host,
			Port:     p.Port,
			Protocol: p.Protocol,
			Enabled:  p.Status == "active",
			Metadata: map[string]any{
				"source":   "fbflix",
				"fbflixId": p.ID,
				"country":  p.Country,
				"region":   p.Region,
			},
		}

		if p.Username != "" {
			params.Username = &p.Username
		}
		if p.Password != "" {
			passwordSecretName := "literal:" + p.Password
			params.PasswordSecretName = &passwordSecretName
		}

		_, err := s.repo.Upsert(ctx, params)
		if err != nil {
			// Log error but continue with other proxies
			continue
		}
		count++
	}

	return count, nil
}
