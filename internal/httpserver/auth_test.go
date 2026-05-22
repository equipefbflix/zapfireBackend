package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"aquecedor-evolution/backend/internal/auth"
	"aquecedor-evolution/backend/internal/config"
)

type flushRecorder struct {
	*httptest.ResponseRecorder
}

func (r flushRecorder) Flush() {}

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
	rec := flushRecorder{ResponseRecorder: httptest.NewRecorder()}

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestProtectedRouteAllowsCORSPreflightWithoutBearerToken(t *testing.T) {
	server := NewServer(ServerConfig{
		App: config.AppConfig{
			AuthEnabled: true,
		},
		AuthVerifier: fakeHTTPAuthVerifier{err: errors.New("should not be called")},
	})

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/instances", nil)
	req.Header.Set("Origin", "http://127.0.0.1:5173")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "authorization,content-type")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:5173" {
		t.Fatalf("allow origin = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Fatal("allow methods header is empty")
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Fatal("allow headers header is empty")
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

func TestLegacyEvolutionWebhookBypassesAuth(t *testing.T) {
	server := NewServer(ServerConfig{
		App: config.AppConfig{
			AuthEnabled: true,
		},
		AuthVerifier:    fakeHTTPAuthVerifier{err: errors.New("should not be called")},
		EvolutionEvents: &fakeHTTPEvolutionEventStore{},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/webhook/evolution", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code == http.StatusUnauthorized {
		t.Fatalf("status = %d, want non-auth failure", rec.Code)
	}
}

func TestInstanceEventsRouteRequiresAccessTokenWhenAuthEnabled(t *testing.T) {
	server := NewServer(ServerConfig{
		App: config.AppConfig{
			AuthEnabled: true,
		},
		AuthVerifier: fakeHTTPAuthVerifier{},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances/events", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestInstanceEventsRouteAllowsAccessTokenQueryString(t *testing.T) {
	server := NewServer(ServerConfig{
		App: config.AppConfig{
			AuthEnabled: true,
		},
		AuthVerifier: fakeHTTPAuthVerifier{user: auth.User{
			ID:    "user-123",
			Email: "user@example.com",
			Role:  "authenticated",
		}},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances/events?access_token=good-token", nil).WithContext(ctx)
	rec := flushRecorder{ResponseRecorder: httptest.NewRecorder()}

	done := make(chan struct{})
	go func() {
		server.Handler().ServeHTTP(rec, req)
		close(done)
	}()
	time.Sleep(10 * time.Millisecond)
	cancel()
	<-done

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
