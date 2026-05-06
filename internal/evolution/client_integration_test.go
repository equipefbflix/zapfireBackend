//go:build integration

package evolution

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestEvolutionFetchInstancesReal(t *testing.T) {
	if os.Getenv("ENABLE_REAL_TESTS") != "true" {
		t.Skip("set ENABLE_REAL_TESTS=true to run real Evolution integration tests")
	}

	baseURL := os.Getenv("EVOLUTION_TEST_BASE_URL")
	apiKey := os.Getenv("EVOLUTION_TEST_API_KEY")
	if baseURL == "" {
		t.Fatal("EVOLUTION_TEST_BASE_URL is required")
	}
	if apiKey == "" {
		t.Fatal("EVOLUTION_TEST_API_KEY is required")
	}

	client := NewClient(Config{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Timeout: 30 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, err := client.FetchInstances(ctx, ""); err != nil {
		t.Fatalf("FetchInstances() real error = %v", err)
	}
}
