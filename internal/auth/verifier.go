package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type User struct {
	ID        string
	Email     string
	Role      string
	SessionID string
}

type Verifier interface {
	Verify(ctx context.Context, token string) (User, error)
}

type contextKey string

const userContextKey contextKey = "auth.user"

func WithUser(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func UserFromContext(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(userContextKey).(User)
	return user, ok
}

type SupabaseVerifierConfig struct {
	Issuer         string
	JWKSURL        string
	Client         *http.Client
	RefreshTimeout time.Duration
}

type SupabaseVerifier struct {
	issuer         string
	jwksURL        string
	client         *http.Client
	refreshTimeout time.Duration

	mu      sync.RWMutex
	keySet  map[string]any
	loaded  bool
}

type supabaseClaims struct {
	Email     string `json:"email"`
	Role      string `json:"role"`
	SessionID string `json:"session_id"`
	jwt.RegisteredClaims
}

type jwksResponse struct {
	Keys []jsonWebKey `json:"keys"`
}

type jsonWebKey struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Crv string `json:"crv"`
	N   string `json:"n"`
	E   string `json:"e"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

func NewSupabaseVerifier(cfg SupabaseVerifierConfig) *SupabaseVerifier {
	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	refreshTimeout := cfg.RefreshTimeout
	if refreshTimeout <= 0 {
		refreshTimeout = 10 * time.Second
	}

	return &SupabaseVerifier{
		issuer:         strings.TrimSpace(cfg.Issuer),
		jwksURL:        strings.TrimSpace(cfg.JWKSURL),
		client:         client,
		refreshTimeout: refreshTimeout,
	}
}

func (v *SupabaseVerifier) Verify(ctx context.Context, token string) (User, error) {
	if strings.TrimSpace(token) == "" {
		return User{}, fmt.Errorf("missing bearer token")
	}

	parsedToken, err := jwt.ParseWithClaims(token, &supabaseClaims{}, func(parsedToken *jwt.Token) (any, error) {
		kid, _ := parsedToken.Header["kid"].(string)
		if kid == "" {
			return nil, fmt.Errorf("missing kid header")
		}

		if key, ok := v.lookupKey(kid); ok {
			return key, nil
		}
		if err := v.refreshKeys(ctx); err != nil {
			return nil, err
		}
		key, ok := v.lookupKey(kid)
		if !ok {
			return nil, fmt.Errorf("signing key %q not found", kid)
		}
		return key, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg(), jwt.SigningMethodES256.Alg()}), jwt.WithIssuer(v.issuer))
	if err != nil {
		return User{}, err
	}

	claims, ok := parsedToken.Claims.(*supabaseClaims)
	if !ok || !parsedToken.Valid {
		return User{}, fmt.Errorf("invalid token claims")
	}
	if !containsAudience(claims.Audience, "authenticated") {
		return User{}, fmt.Errorf("invalid token audience")
	}
	if strings.TrimSpace(claims.Subject) == "" {
		return User{}, fmt.Errorf("missing token subject")
	}

	return User{
		ID:        claims.Subject,
		Email:     claims.Email,
		Role:      claims.Role,
		SessionID: claims.SessionID,
	}, nil
}

func containsAudience(audiences []string, expected string) bool {
	for _, audience := range audiences {
		if audience == expected {
			return true
		}
	}
	return false
}

func (v *SupabaseVerifier) lookupKey(kid string) (any, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if !v.loaded {
		return nil, false
	}
	key, ok := v.keySet[kid]
	return key, ok
}

func (v *SupabaseVerifier) refreshKeys(ctx context.Context) error {
	refreshCtx, cancel := context.WithTimeout(ctx, v.refreshTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(refreshCtx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return err
	}

	resp, err := v.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks fetch failed: status %d", resp.StatusCode)
	}

	var body jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return err
	}

	keySet := make(map[string]any, len(body.Keys))
	for _, key := range body.Keys {
		if key.Kid == "" {
			continue
		}
		publicKey, err := publicKeyFromJWK(key)
		if err != nil {
			return err
		}
		if publicKey == nil {
			continue
		}
		keySet[key.Kid] = publicKey
	}
	if len(keySet) == 0 {
		return fmt.Errorf("jwks did not contain any supported signing keys")
	}

	v.mu.Lock()
	v.keySet = keySet
	v.loaded = true
	v.mu.Unlock()
	return nil
}

func publicKeyFromJWK(key jsonWebKey) (any, error) {
	switch key.Kty {
	case "RSA":
		if key.N == "" || key.E == "" {
			return nil, nil
		}
		return rsaPublicKeyFromJWK(key)
	case "EC":
		if key.Crv != "P-256" || key.X == "" || key.Y == "" {
			return nil, nil
		}
		return ecdsaPublicKeyFromJWK(key)
	default:
		return nil, nil
	}
}

func rsaPublicKeyFromJWK(key jsonWebKey) (*rsa.PublicKey, error) {
	modulusBytes, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		return nil, fmt.Errorf("decode jwk modulus: %w", err)
	}
	exponentBytes, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil {
		return nil, fmt.Errorf("decode jwk exponent: %w", err)
	}

	modulus := new(big.Int).SetBytes(modulusBytes)
	exponent := new(big.Int).SetBytes(exponentBytes)
	if !exponent.IsInt64() {
		return nil, fmt.Errorf("invalid jwk exponent")
	}

	return &rsa.PublicKey{
		N: modulus,
		E: int(exponent.Int64()),
	}, nil
}

func ecdsaPublicKeyFromJWK(key jsonWebKey) (*ecdsa.PublicKey, error) {
	xBytes, err := base64.RawURLEncoding.DecodeString(key.X)
	if err != nil {
		return nil, fmt.Errorf("decode jwk x: %w", err)
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(key.Y)
	if err != nil {
		return nil, fmt.Errorf("decode jwk y: %w", err)
	}

	curve := elliptic.P256()
	x := new(big.Int).SetBytes(xBytes)
	y := new(big.Int).SetBytes(yBytes)
	if !curve.IsOnCurve(x, y) {
		return nil, fmt.Errorf("ecdsa key is not on curve")
	}

	return &ecdsa.PublicKey{
		Curve: curve,
		X:     x,
		Y:     y,
	}, nil
}
