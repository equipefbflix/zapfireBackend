package fbflix

import (
	"context"
	"testing"

	"aquecedor-evolution/backend/internal/repository"
	"github.com/stretchr/testify/assert"
)

type mockClient struct {
	proxies []Proxy
	err     error
}

func (m *mockClient) ListProxies(ctx context.Context) ([]Proxy, error) {
	return m.proxies, m.err
}

type mockRepo struct {
	upserted []repository.CreateProxyParams
	err      error
}

func (m *mockRepo) Upsert(ctx context.Context, params repository.CreateProxyParams) (repository.Proxy, error) {
	m.upserted = append(m.upserted, params)
	return repository.Proxy{}, m.err
}

func TestSyncService(t *testing.T) {
	// Arrange
	client := &mockClient{
		proxies: []Proxy{
			{ID: "d-1", Host: "1.2.3.4", Port: 8080, Protocol: "http", Username: "u1", Password: "p1", Status: "active"},
		},
	}
	repo := &mockRepo{}
	service := NewSyncService(client, repo)

	// Act
	count, err := service.Sync(context.Background())

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Len(t, repo.upserted, 1)
	assert.Equal(t, "1.2.3.4", repo.upserted[0].Host)
	assert.NotNil(t, repo.upserted[0].PasswordSecretName)
	assert.Equal(t, "literal:p1", *repo.upserted[0].PasswordSecretName)
	assert.Equal(t, "fbflix", repo.upserted[0].Metadata["source"])
}
