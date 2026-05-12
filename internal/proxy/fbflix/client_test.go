package fbflix

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func test_list_proxies_success(t *testing.T) {
	// Arrange
	response, err := os.ReadFile("testdata/proxies_response.json")
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, "get-proxy-list", payload["action"])
		assert.Equal(t, float64(1), payload["page"])
		assert.Equal(t, float64(100), payload["page_size"])
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(response)
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL: server.URL,
		Token:   "test-token",
	})

	// Act
	proxies, err := client.ListProxies(context.Background())

	// Assert
	assert.NoError(t, err)
	assert.Len(t, proxies, 2)
	assert.Equal(t, "1.2.3.4", proxies[0].Host)
	assert.Equal(t, 8080, proxies[0].Port)
	assert.Equal(t, "http", proxies[0].Protocol)
	assert.Equal(t, "5.6.7.8", proxies[1].Host)
	assert.Equal(t, "Fortaleza", proxies[1].Region)
	assert.Equal(t, "inactive", proxies[1].Status)
}

func test_list_proxies_unauthorized(t *testing.T) {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "unauthorized"}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL: server.URL,
		Token:   "invalid-token",
	})

	// Act
	proxies, err := client.ListProxies(context.Background())

	// Assert
	assert.Error(t, err)
	assert.Nil(t, proxies)
	assert.Contains(t, err.Error(), "unauthorized")
}

func test_purchase_proxy_with_balance_success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, "purchase-proxy-with-balance", payload["action"])
		assert.Equal(t, "product-id", payload["product_id"])
		assert.Equal(t, "order-id", payload["order_id"])
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"success": true,
			"quantity": 1,
			"subscriptions": ["WS-ABCD123"],
			"proxies": [
				{"ip": "185.242.1.2", "port": 8080, "username": "user123", "password": "pass_secure"}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL: server.URL,
		Token:   "test-token",
	})

	response, err := client.PurchaseProxyWithBalance(context.Background(), PurchaseProxyRequest{
		Total:     9.9,
		ProductID: "product-id",
		OrderID:   "order-id",
		Quantity:  1,
	})

	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, []string{"WS-ABCD123"}, response.Subscriptions)
	require.Len(t, response.Proxies, 1)
	assert.Equal(t, "185.242.1.2", response.Proxies[0].Host)
	assert.Equal(t, "http", response.Proxies[0].Protocol)
}

func TestClient(t *testing.T) {
	t.Run("ListProxiesSuccess", test_list_proxies_success)
	t.Run("ListProxiesUnauthorized", test_list_proxies_unauthorized)
	t.Run("PurchaseProxyWithBalanceSuccess", test_purchase_proxy_with_balance_success)
}
