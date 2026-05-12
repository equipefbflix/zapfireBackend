package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockFBFlixSyncService struct {
	syncCalled bool
	count      int
	err        error
}

func (m *mockFBFlixSyncService) Sync(ctx context.Context) (int, error) {
	m.syncCalled = true
	return m.count, m.err
}

func TestFBFlixSyncRoute(t *testing.T) {
	// Arrange
	syncService := &mockFBFlixSyncService{count: 5}
	server := NewServer(ServerConfig{
		FBFlixSync: syncService,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/proxies/sync/fbflix", nil)
	rec := httptest.NewRecorder()

	// Act
	server.Handler().ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, syncService.syncCalled)
	assert.Contains(t, rec.Body.String(), `"count":5`)
}
