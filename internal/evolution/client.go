package evolution

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Config struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("evolution api returned status %d: %s", e.StatusCode, e.Body)
}

func NewClient(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:  cfg.APIKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

type InstanceSummary struct {
	Name             string `json:"name"`
	InstanceName     string `json:"instanceName"`
	ConnectionStatus string `json:"connectionStatus"`
}

type CreateInstanceRequest struct {
	InstanceName    string         `json:"instanceName"`
	Integration     string         `json:"integration"`
	Token           string         `json:"token,omitempty"`
	QRCode          bool           `json:"qrcode"`
	Number          string         `json:"number,omitempty"`
	RejectCall      bool           `json:"rejectCall"`
	MsgCall         string         `json:"msgCall,omitempty"`
	GroupsIgnore    bool           `json:"groupsIgnore"`
	AlwaysOnline    bool           `json:"alwaysOnline"`
	ReadMessages    bool           `json:"readMessages"`
	ReadStatus      bool           `json:"readStatus"`
	SyncFullHistory bool           `json:"syncFullHistory"`
	ProxyHost       string         `json:"proxyHost,omitempty"`
	ProxyPort       string         `json:"proxyPort,omitempty"`
	ProxyProtocol   string         `json:"proxyProtocol,omitempty"`
	ProxyUsername   string         `json:"proxyUsername,omitempty"`
	ProxyPassword   string         `json:"proxyPassword,omitempty"`
	Webhook         *WebhookConfig `json:"webhook,omitempty"`
}

type WebhookConfig struct {
	URL      string   `json:"url"`
	ByEvents bool     `json:"byEvents"`
	Base64   bool     `json:"base64"`
	Events   []string `json:"events"`
}

type CreateInstanceResponse struct {
	Instance struct {
		InstanceName string `json:"instanceName"`
	} `json:"instance"`
	Hash InstanceHash `json:"hash"`
}

type InstanceHash struct {
	APIKey string
}

func (h *InstanceHash) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err == nil {
		h.APIKey = value
		return nil
	}

	var payload struct {
		APIKey string `json:"apikey"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	h.APIKey = payload.APIKey
	return nil
}

type ConnectInstanceResponse struct {
	PairingCode string `json:"pairingCode"`
	Code        string `json:"code"`
	Count       int    `json:"count"`
}

type ConnectionStateResponse struct {
	Instance struct {
		InstanceName string `json:"instanceName"`
		State        string `json:"state"`
	} `json:"instance"`
}

type SendTextRequest struct {
	Number      string      `json:"number"`
	Text        string      `json:"text"`
	Delay       int         `json:"delay,omitempty"`
	LinkPreview bool        `json:"linkPreview"`
	Quoted      *QuotedInfo `json:"quoted,omitempty"`
}

type QuotedInfo struct {
	Key     MessageKey `json:"key"`
	Message any        `json:"message,omitempty"`
}

type MessageKey struct {
	RemoteJID string `json:"remoteJid,omitempty"`
	FromMe    bool   `json:"fromMe"`
	ID        string `json:"id"`
}

type SendMessageResponse struct {
	Key           MessageKey `json:"key"`
	AcceptedAsync bool       `json:"-"`
}

type SendMediaRequest struct {
	Number    string `json:"number"`
	MediaType string `json:"mediatype"`
	MimeType  string `json:"mimetype,omitempty"`
	Caption   string `json:"caption,omitempty"`
	Media     string `json:"media"`
	FileName  string `json:"fileName,omitempty"`
	Delay     int    `json:"delay,omitempty"`
}

type SendWhatsAppAudioRequest struct {
	Number string `json:"number"`
	Audio  string `json:"audio"`
	Delay  int    `json:"delay,omitempty"`
}

type SendStatusRequest struct {
	Type        string `json:"type"`
	Content     string `json:"content"`
	Caption     string `json:"caption,omitempty"`
	Background  string `json:"backgroundColor,omitempty"`
	Font        int    `json:"font,omitempty"`
	AllContacts bool   `json:"allContacts"`
	StatusJIDList []string `json:"statusJidList,omitempty"`
	Media       string `json:"media,omitempty"`
	Delay       int    `json:"delay,omitempty"`
	LinkPreview bool   `json:"linkPreview,omitempty"`
}

type SendStickerRequest struct {
	Number  string `json:"number"`
	Sticker string `json:"sticker"`
	Delay   int    `json:"delay,omitempty"`
}

type SendPresenceRequest struct {
	Number   string `json:"number"`
	Delay    int    `json:"delay,omitempty"`
	Presence string `json:"presence"`
}

type SendReactionRequest struct {
	Key      MessageKey `json:"key"`
	Reaction string     `json:"reaction"`
}

func (c *Client) FetchInstances(ctx context.Context, instanceName string) ([]InstanceSummary, error) {
	path := "/instance/fetchInstances"
	if instanceName != "" {
		path += "?instanceName=" + url.QueryEscape(instanceName)
	}

	var response []InstanceSummary
	if err := c.do(ctx, http.MethodGet, path, nil, &response); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *Client) CreateInstance(ctx context.Context, request CreateInstanceRequest) (CreateInstanceResponse, error) {
	var response CreateInstanceResponse
	err := c.do(ctx, http.MethodPost, "/instance/create", request, &response)
	return response, err
}

func (c *Client) DeleteInstance(ctx context.Context, instanceName string) error {
	return c.do(ctx, http.MethodDelete, "/instance/delete/"+url.PathEscape(instanceName), nil, nil)
}

func (c *Client) ConnectInstance(ctx context.Context, instanceName, number string) (ConnectInstanceResponse, error) {
	path := "/instance/connect/" + url.PathEscape(instanceName)
	if number != "" {
		path += "?number=" + url.QueryEscape(number)
	}

	var response ConnectInstanceResponse
	err := c.do(ctx, http.MethodGet, path, nil, &response)
	return response, err
}

func (c *Client) ConnectionState(ctx context.Context, instanceName string) (ConnectionStateResponse, error) {
	var response ConnectionStateResponse
	err := c.do(ctx, http.MethodGet, "/instance/connectionState/"+url.PathEscape(instanceName), nil, &response)
	return response, err
}

func (c *Client) SendText(ctx context.Context, instanceName string, request SendTextRequest) (SendMessageResponse, error) {
	var response SendMessageResponse
	err := c.do(ctx, http.MethodPost, "/message/sendText/"+url.PathEscape(instanceName), request, &response)
	return response, err
}

func (c *Client) SendMedia(ctx context.Context, instanceName string, request SendMediaRequest) (SendMessageResponse, error) {
	var response SendMessageResponse
	err := c.do(ctx, http.MethodPost, "/message/sendMedia/"+url.PathEscape(instanceName), request, &response)
	return response, err
}

func (c *Client) SendWhatsAppAudio(ctx context.Context, instanceName string, request SendWhatsAppAudioRequest) (SendMessageResponse, error) {
	var response SendMessageResponse
	err := c.do(ctx, http.MethodPost, "/message/sendWhatsAppAudio/"+url.PathEscape(instanceName), request, &response)
	return response, err
}

func (c *Client) SendStatus(ctx context.Context, instanceName string, request SendStatusRequest) (SendMessageResponse, error) {
	asyncCtx, cancel := context.WithTimeout(ctx, c.statusAsyncWait())
	defer cancel()

	var response SendMessageResponse
	err := c.do(asyncCtx, http.MethodPost, "/message/sendStatus/"+url.PathEscape(instanceName), request, &response)
	if err == nil {
		return response, nil
	}
	if !isAcceptedAsyncError(err) {
		return SendMessageResponse{}, err
	}
	return SendMessageResponse{AcceptedAsync: true}, nil
}

func (c *Client) SendSticker(ctx context.Context, instanceName string, request SendStickerRequest) error {
	return c.do(ctx, http.MethodPost, "/message/sendSticker/"+url.PathEscape(instanceName), request, nil)
}

func (c *Client) SendPresence(ctx context.Context, instanceName string, request SendPresenceRequest) error {
	return c.do(ctx, http.MethodPost, "/chat/sendPresence/"+url.PathEscape(instanceName), request, nil)
}

func (c *Client) SendReaction(ctx context.Context, instanceName string, request SendReactionRequest) error {
	return c.do(ctx, http.MethodPost, "/message/sendReaction/"+url.PathEscape(instanceName), request, nil)
}

func (c *Client) do(ctx context.Context, method, path string, requestBody any, responseBody any) error {
	var body io.Reader
	if requestBody != nil {
		data, err := json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("marshal evolution request: %w", err)
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("create evolution request: %w", err)
	}
	req.Header.Set("apikey", c.apiKey)
	if requestBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send evolution request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read evolution response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Body:       strings.TrimSpace(string(data)),
		}
	}

	if responseBody == nil || len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, responseBody); err != nil {
		return fmt.Errorf("decode evolution response: %w", err)
	}
	return nil
}

func isAcceptedAsyncError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr interface{ Timeout() bool }
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	text := err.Error()
	return strings.Contains(text, "Client.Timeout exceeded while awaiting headers") ||
		strings.Contains(text, "Remote end closed connection without response") ||
		strings.Contains(text, "EOF")
}

func (c *Client) statusAsyncWait() time.Duration {
	wait := c.httpClient.Timeout
	if wait <= 0 || wait > 10*time.Second {
		return 10 * time.Second
	}
	return wait
}
