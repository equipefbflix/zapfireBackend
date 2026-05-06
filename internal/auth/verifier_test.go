package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestSupabaseVerifierVerify(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	kid := "test-kid"
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]string{
				{
					"kty": "RSA",
					"kid": kid,
					"use": "sig",
					"alg": "RS256",
					"n":   base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.PublicKey.E)).Bytes()),
				},
			},
		})
	}))
	defer jwksServer.Close()

	issuer := "https://project-ref.supabase.co/auth/v1"
	tokenString := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":        issuer,
		"aud":        "authenticated",
		"sub":        "user-123",
		"email":      "user@example.com",
		"role":       "authenticated",
		"session_id": "session-123",
		"exp":        time.Now().Add(5 * time.Minute).Unix(),
		"iat":        time.Now().Add(-1 * time.Minute).Unix(),
	})
	tokenString.Header["kid"] = kid

	signedToken, err := tokenString.SignedString(privateKey)
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}

	verifier := NewSupabaseVerifier(SupabaseVerifierConfig{
		Issuer:  issuer,
		JWKSURL: jwksServer.URL,
		Client:  jwksServer.Client(),
	})

	user, err := verifier.Verify(context.Background(), signedToken)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}

	if user.ID != "user-123" {
		t.Fatalf("user.ID = %q", user.ID)
	}
	if user.Email != "user@example.com" {
		t.Fatalf("user.Email = %q", user.Email)
	}
	if user.Role != "authenticated" {
		t.Fatalf("user.Role = %q", user.Role)
	}
}
