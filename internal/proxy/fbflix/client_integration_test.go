//go:build integration

package fbflix

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestClientListProxiesRealAPI(t *testing.T) {
	if os.Getenv("ENABLE_REAL_TESTS") != "true" {
		t.Skip("set ENABLE_REAL_TESTS=true to run real FBFlix integration tests")
	}

	token := os.Getenv("FBFLIX_B2B_TOKEN")
	if token == "" {
		t.Fatal("FBFLIX_B2B_TOKEN is required")
	}

	baseURL := os.Getenv("FBFLIX_API_URL")
	if baseURL == "" {
		baseURL = "https://mxnlerkeygfvdnznoxld.supabase.co/functions/v1/proxyfbflix-api"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := NewClient(Config{
		BaseURL: baseURL,
		Token:   token,
		Timeout: 30 * time.Second,
	})

	proxies, err := client.ListProxies(ctx)
	if err != nil {
		t.Fatalf("ListProxies() real API error = %v", err)
	}
	if len(proxies) == 0 {
		t.Fatal("ListProxies() returned zero proxies from real API")
	}

	first := proxies[0]
	if first.Host == "" {
		t.Fatal("first proxy host is empty")
	}
	if first.Port == 0 {
		t.Fatal("first proxy port is zero")
	}
	if first.Username == "" {
		t.Fatal("first proxy username is empty")
	}
	if first.Password == "" {
		t.Fatal("first proxy password is empty")
	}
}
