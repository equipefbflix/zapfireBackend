package fbflix

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Config struct {
	BaseURL string
	Token   string
	Timeout time.Duration
}

type Proxy struct {
	ID       string `json:"id"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Protocol string `json:"protocol"`
	Country  string `json:"country"`
	Region   string `json:"region"`
	Status   string `json:"status"`
}

type PurchaseProxyRequest struct {
	Total      float64 `json:"total"`
	ProductID  string  `json:"product_id"`
	OrderID    string  `json:"order_id"`
	Quantity   int     `json:"quantity"`
	TargetCity string  `json:"target_city,omitempty"`
}

type PurchaseProxyResponse struct {
	Success       bool     `json:"success"`
	Quantity      int      `json:"quantity"`
	Subscriptions []string `json:"subscriptions"`
	Proxies       []Proxy  `json:"proxies"`
}

type Client struct {
	cfg        Config
	httpClient *http.Client
}

func NewClient(cfg Config) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func (c *Client) ListProxies(ctx context.Context) ([]Proxy, error) {
	var response listProxiesResponse
	if err := c.postAction(ctx, map[string]any{
		"action":    "get-proxy-list",
		"page":      1,
		"page_size": 100,
	}, &response); err != nil {
		return nil, err
	}

	return response.ProxyList(), nil
}

func (c *Client) PurchaseProxyWithBalance(ctx context.Context, purchase PurchaseProxyRequest) (PurchaseProxyResponse, error) {
	payload := map[string]any{
		"action":     "purchase-proxy-with-balance",
		"total":      purchase.Total,
		"product_id": purchase.ProductID,
		"order_id":   purchase.OrderID,
		"quantity":   purchase.Quantity,
	}
	if purchase.TargetCity != "" {
		payload["target_city"] = purchase.TargetCity
	}

	var response purchaseProxyAPIResponse
	if err := c.postAction(ctx, payload, &response); err != nil {
		return PurchaseProxyResponse{}, err
	}
	if !response.Success {
		return PurchaseProxyResponse{}, fmt.Errorf("api error: purchase proxy returned success=false")
	}

	return PurchaseProxyResponse{
		Success:       response.Success,
		Quantity:      response.Quantity,
		Subscriptions: response.Subscriptions,
		Proxies:       normalizeAPIProxies(response.Proxies),
	}, nil
}

func (c *Client) postAction(ctx context.Context, payload map[string]any, target any) error {
	url := strings.TrimRight(c.cfg.BaseURL, "/")
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.cfg.Token))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
			return fmt.Errorf("api error (status %d): %s", resp.StatusCode, errResp.Error)
		}
		return fmt.Errorf("api error: status %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if strings.Contains(contentType, "text/html") || bytes.HasPrefix(bytes.TrimSpace(responseBody), []byte("<!doctype html")) {
		return fmt.Errorf("api error: expected json response, got html")
	}
	if err := json.Unmarshal(responseBody, target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

type listProxiesResponse struct {
	Results []proxyAPIItem `json:"results"`
	Data    []proxyAPIItem `json:"data"`
	Proxies []proxyAPIItem `json:"proxies"`
	Legacy  []Proxy
}

func (r *listProxiesResponse) UnmarshalJSON(data []byte) error {
	var legacy []Proxy
	if err := json.Unmarshal(data, &legacy); err == nil {
		r.Legacy = legacy
		return nil
	}

	var payload struct {
		Results []proxyAPIItem `json:"results"`
		Data    []proxyAPIItem `json:"data"`
		Proxies []proxyAPIItem `json:"proxies"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	r.Results = payload.Results
	r.Data = payload.Data
	r.Proxies = payload.Proxies
	return nil
}

func (r listProxiesResponse) ProxyList() []Proxy {
	if len(r.Legacy) > 0 {
		return r.Legacy
	}
	if len(r.Results) > 0 {
		return normalizeAPIProxies(r.Results)
	}
	if len(r.Data) > 0 {
		return normalizeAPIProxies(r.Data)
	}
	return normalizeAPIProxies(r.Proxies)
}

type purchaseProxyAPIResponse struct {
	Success       bool           `json:"success"`
	Quantity      int            `json:"quantity"`
	Subscriptions []string       `json:"subscriptions"`
	Proxies       []proxyAPIItem `json:"proxies"`
}

type proxyAPIItem struct {
	ID              string `json:"id"`
	Host            string `json:"host"`
	ProxyAddress    string `json:"proxy_address"`
	IP              string `json:"ip"`
	Port            int    `json:"port"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	Protocol        string `json:"protocol"`
	Country         string `json:"country"`
	CountryCode     string `json:"country_code"`
	Region          string `json:"region"`
	CityName        string `json:"city_name"`
	Status          string `json:"status"`
	Valid           *bool  `json:"valid"`
	HTTPPort        int    `json:"http_port"`
	SOCKS5Port      int    `json:"socks5_port"`
	ProxyHost       string `json:"proxy_host"`
	ProxyAddressAlt string `json:"proxyAddress"`
}

func normalizeAPIProxies(items []proxyAPIItem) []Proxy {
	proxies := make([]Proxy, 0, len(items))
	for _, item := range items {
		host := firstNonEmpty(item.Host, item.ProxyAddress, item.IP, item.ProxyHost, item.ProxyAddressAlt)
		port := item.Port
		if port == 0 {
			port = item.HTTPPort
		}
		if host == "" || port == 0 {
			continue
		}

		protocol := strings.ToLower(firstNonEmpty(item.Protocol, "http"))
		status := item.Status
		if status == "" {
			status = "active"
			if item.Valid != nil && !*item.Valid {
				status = "inactive"
			}
		}

		proxies = append(proxies, Proxy{
			ID:       item.ID,
			Host:     host,
			Port:     port,
			Username: item.Username,
			Password: item.Password,
			Protocol: protocol,
			Country:  firstNonEmpty(item.Country, item.CountryCode),
			Region:   firstNonEmpty(item.Region, item.CityName),
			Status:   status,
		})
	}
	return proxies
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
