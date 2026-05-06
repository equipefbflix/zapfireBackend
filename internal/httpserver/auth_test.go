package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"aquecedor-evolution/backend/internal/auth"
	"aquecedor-evolution/backend/internal/config"
)

type fakeHTTPAuthVerifier struct {
	user auth.User
	err  error
}

func (v fakeHTTPAuthVerifier) Verify(ctx context.Context, token string) (auth.User, error) {
	if token == "" {
		return auth.User{}, errors.New("missing token")
	}
	if v.err != nil {
		return auth.User{}, v.err
	}
	return v.user, nil
}

func TestProtectedRouteRequiresBearerToken(t *testing.T) {
	server := NewServer(ServerConfig{
		App: config.AppConfig{
			AuthEnabled: true,
		},
		AuthVerifier: fakeHTTPAuthVerifier{},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/phone-numbers", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestProtectedRouteRejectsInvalidBearerToken(t *testing.T) {
	server := NewServer(ServerConfig{
		App: config.AppConfig{
			AuthEnabled: true,
		},
		AuthVerifier: fakeHTTPAuthVerifier{err: errors.New("invalid token")},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/phone-numbers", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestProtectedRouteAllowsValidBearerToken(t *testing.T) {
	server := NewServer(ServerConfig{
		App: config.AppConfig{
			AuthEnabled: true,
		},
		AuthVerifier: fakeHTTPAuthVerifier{user: auth.User{
			ID:    "user-123",
			Email: "user@example.com",
			Role:  "authenticated",
		}},
		PhoneNumbers: &fakePhoneNumberStore{},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/phone-numbers", nil)
	req.Header.Set("Authorization", "Bearer good-token")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHealthRouteBypassesAuth(t *testing.T) {
	server := NewServer(ServerConfig{
		App: config.AppConfig{
			AuthEnabled: true,
		},
		AuthVerifier: fakeHTTPAuthVerifier{err: errors.New("should not be called")},
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestEvolutionWebhookBypassesAuth(t *testing.T) {
	server := NewServer(ServerConfig{
		App: config.AppConfig{
			AuthEnabled: true,
		},
		AuthVerifier:    fakeHTTPAuthVerifier{err: errors.New("should not be called")},
		EvolutionEvents: &fakeHTTPEvolutionEventStore{},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/evolution", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code == http.StatusUnauthorized {
		t.Fatalf("status = %d, want non-auth failure", rec.Code)
	}
}
