package evolution

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Config struct {
	BaseURL    string
	APIKey     string
	Timeout    time.Duration
	WebhookURL string
}

type Client struct {
	baseURL    string
	apiKey     string
	webhookURL string
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
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:     cfg.APIKey,
		webhookURL: strings.TrimSpace(cfg.WebhookURL),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

type InstanceSummary struct {
	ID               string `json:"id,omitempty"`
	Name             string `json:"name"`
	InstanceName     string `json:"instanceName"`
	ConnectionStatus string `json:"connectionStatus"`
	JID              string `json:"jid,omitempty"`
	Connected        bool   `json:"connected,omitempty"`
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
		ID           string `json:"id,omitempty"`
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
	JID         string `json:"jid,omitempty"`
	EventString string `json:"eventString,omitempty"`
	WebhookURL  string `json:"webhookUrl,omitempty"`
}

type ConnectionStateResponse struct {
	Instance struct {
		InstanceName string `json:"instanceName"`
		ID           string `json:"id"`
		State        string `json:"state"`
		JID          string `json:"jid,omitempty"`
		Connected    bool   `json:"connected,omitempty"`
		LoggedIn     bool   `json:"loggedIn,omitempty"`
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
	Type          string   `json:"type"`
	Content       string   `json:"content"`
	Caption       string   `json:"caption,omitempty"`
	Background    string   `json:"backgroundColor,omitempty"`
	Font          int      `json:"font,omitempty"`
	AllContacts   bool     `json:"allContacts"`
	StatusJIDList []string `json:"statusJidList,omitempty"`
	Media         string   `json:"media,omitempty"`
	Delay         int      `json:"delay,omitempty"`
	LinkPreview   bool     `json:"linkPreview,omitempty"`
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
	var itemsData []evolutionGoInstanceSummary
	if err := c.doEnvelope(ctx, http.MethodGet, "/instance/all", nil, &itemsData); err != nil {
		return nil, err
	}
	items := make([]InstanceSummary, 0, len(itemsData))
	for _, item := range itemsData {
		summary := InstanceSummary{
			ID:               item.ID,
			Name:             item.Name,
			InstanceName:     item.Name,
			JID:              item.JID,
			Connected:        item.Connected,
			ConnectionStatus: mapStatus(item.Status, item.Connected),
		}
		if instanceName != "" && summary.InstanceName != instanceName && summary.ID != instanceName {
			continue
		}
		items = append(items, summary)
	}
	return items, nil
}

func (c *Client) CreateInstance(ctx context.Context, request CreateInstanceRequest) (CreateInstanceResponse, error) {
	payload := evolutionGoCreateRequest{
		InstanceID: newEvolutionGoInstanceID(),
		Name:       request.InstanceName,
		Token:      request.Token,
		AdvancedSettings: &evolutionGoAdvancedSettings{
			AlwaysOnline:  request.AlwaysOnline,
			IgnoreGroups:  request.GroupsIgnore,
			IgnoreStatus:  !request.ReadStatus,
			MsgRejectCall: request.MsgCall,
			ReadMessages:  request.ReadMessages,
			RejectCall:    request.RejectCall,
		},
	}
	if request.ProxyHost != "" || request.ProxyPort != "" || request.ProxyUsername != "" || request.ProxyPassword != "" {
		payload.Proxy = &evolutionGoProxyConfig{
			Protocol: request.ProxyProtocol,
			Host:     request.ProxyHost,
			Port:     request.ProxyPort,
			Username: request.ProxyUsername,
			Password: request.ProxyPassword,
		}
	}

	var data evolutionGoCreateResponse
	if err := c.doEnvelope(ctx, http.MethodPost, "/instance/create", payload, &data); err != nil {
		return CreateInstanceResponse{}, err
	}
	var response CreateInstanceResponse
	response.Instance.InstanceName = data.Name
	response.Instance.ID = data.ID
	response.Hash.APIKey = data.Token
	return response, nil
}

func newEvolutionGoInstanceID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("00000000-0000-4000-8000-%012d", time.Now().UnixNano()%1_000_000_000_000)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4],
		b[4:6],
		b[6:8],
		b[8:10],
		b[10:16],
	)
}

func (c *Client) DeleteInstance(ctx context.Context, instanceName string) error {
	return c.do(ctx, http.MethodDelete, "/instance/delete/"+url.PathEscape(instanceName), nil, nil)
}

func (c *Client) DeleteInstanceByUUID(ctx context.Context, uuid string) error {
	return c.do(ctx, http.MethodDelete, "/instance/delete/"+url.PathEscape(uuid), nil, nil)
}

func (c *Client) RestartInstance(ctx context.Context, instanceName string) error {
	return c.do(ctx, http.MethodPost, "/instance/reconnect", struct{}{}, nil)
}

func (c *Client) ConnectInstance(ctx context.Context, instanceName, number string) (ConnectInstanceResponse, error) {
	connectPayload := evolutionGoConnectRequest{}
	if number != "" {
		connectPayload.Phone = number
	}
	if c.webhookURL != "" {
		connectPayload.WebhookURL = c.webhookURL
		connectPayload.Subscribe = []string{
			"messages.upsert",
			"connection.update",
		}
	}

	var connectData evolutionGoConnectResponse
	connectErr := c.doEnvelope(ctx, http.MethodPost, "/instance/connect", connectPayload, &connectData)
	if connectErr != nil {
		var qrData evolutionGoQRResponse
		if qrErr := c.doEnvelope(ctx, http.MethodGet, "/instance/qr", nil, &qrData); qrErr == nil {
			return ConnectInstanceResponse{
				PairingCode: qrData.Qrcode,
				Code:        qrData.Code,
			}, nil
		}
		return ConnectInstanceResponse{}, connectErr
	}

	var qrData evolutionGoQRResponse
	if err := c.doEnvelope(ctx, http.MethodGet, "/instance/qr", nil, &qrData); err != nil {
		return ConnectInstanceResponse{
			JID:         connectData.JID,
			EventString: connectData.EventString,
			WebhookURL:  connectData.WebhookURL,
		}, nil
	}

	return ConnectInstanceResponse{
		PairingCode: qrData.Qrcode,
		Code:        qrData.Code,
		JID:         connectData.JID,
		EventString: connectData.EventString,
		WebhookURL:  connectData.WebhookURL,
	}, nil
}

func (c *Client) ConnectionState(ctx context.Context, instanceName string) (ConnectionStateResponse, error) {
	var allResp struct {
		Data []evolutionGoStatusResponse `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, "/instance/all", nil, &allResp); err != nil {
		return ConnectionStateResponse{}, err
	}

	for _, inst := range allResp.Data {
		if inst.Name == instanceName {
			var response ConnectionStateResponse
			response.Instance.InstanceName = inst.Name
			response.Instance.ID = inst.InstanceID
			response.Instance.State = mapStatus(inst.Status, inst.Connected)
			response.Instance.JID = firstNonEmpty(inst.MyJID, inst.JID)
			response.Instance.Connected = inst.Connected
			response.Instance.LoggedIn = inst.LoggedIn
			return response, nil
		}
	}

	return ConnectionStateResponse{}, fmt.Errorf("instance %q not found in /instance/all", instanceName)
}

func (c *Client) SendText(ctx context.Context, instanceName string, request SendTextRequest) (SendMessageResponse, error) {
	payload := evolutionGoTextRequest{
		Number: request.Number,
		Text:   request.Text,
		Delay:  request.Delay,
	}
	if request.Quoted != nil {
		payload.Quoted = &evolutionGoQuotedRequest{
			MessageID:   request.Quoted.Key.ID,
			Participant: request.Quoted.Key.RemoteJID,
		}
	}
	return c.sendMessage(ctx, "/send/text", payload)
}

func (c *Client) SendMedia(ctx context.Context, instanceName string, request SendMediaRequest) (SendMessageResponse, error) {
	payload := evolutionGoMediaRequest{
		Number:   request.Number,
		URL:      request.Media,
		Type:     request.MediaType,
		Caption:  request.Caption,
		Filename: request.FileName,
		Delay:    request.Delay,
	}
	return c.sendMessage(ctx, "/send/media", payload)
}

func (c *Client) SendWhatsAppAudio(ctx context.Context, instanceName string, request SendWhatsAppAudioRequest) (SendMessageResponse, error) {
	payload := evolutionGoMediaRequest{
		Number: request.Number,
		URL:    request.Audio,
		Type:   "audio",
		Delay:  request.Delay,
	}
	return c.sendMessage(ctx, "/send/media", payload)
}

func (c *Client) SendStatus(ctx context.Context, instanceName string, request SendStatusRequest) (SendMessageResponse, error) {
	if request.Type == "text" || request.Media == "" {
		return c.sendMessage(ctx, "/send/status/text", evolutionGoStatusTextRequest{
			Text: request.Content,
		})
	}
	return c.sendMessage(ctx, "/send/status/media", evolutionGoStatusMediaRequest{
		Type:    request.Type,
		URL:     request.Media,
		Caption: request.Caption,
	})
}

func (c *Client) SendSticker(ctx context.Context, instanceName string, request SendStickerRequest) error {
	payload := evolutionGoStickerRequest{
		Number:  request.Number,
		Sticker: request.Sticker,
		Delay:   request.Delay,
	}
	_, err := c.sendMessage(ctx, "/send/sticker", payload)
	return err
}

func (c *Client) SendPresence(ctx context.Context, instanceName string, request SendPresenceRequest) error {
	payload := evolutionGoPresenceRequest{
		Number: request.Number,
		State:  request.Presence,
	}
	if request.Presence == "recording" {
		payload.State = "composing"
		payload.IsAudio = true
	}
	return c.do(ctx, http.MethodPost, "/message/presence", payload, nil)
}

func (c *Client) SendReaction(ctx context.Context, instanceName string, request SendReactionRequest) error {
	payload := evolutionGoReactionRequest{
		Number:   request.Key.RemoteJID,
		ID:       request.Key.ID,
		FromMe:   request.Key.FromMe,
		Reaction: request.Reaction,
	}
	if strings.Contains(request.Key.RemoteJID, "@g.us") {
		payload.Participant = request.Key.RemoteJID
	}
	_, err := c.sendMessage(ctx, "/message/react", payload)
	return err
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
	if strings.TrimSpace(c.apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
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

func (c *Client) doEnvelope(ctx context.Context, method, path string, requestBody any, responseBody any) error {
	var raw json.RawMessage
	envelope := envelope[json.RawMessage]{Data: raw}
	if err := c.do(ctx, method, path, requestBody, &envelope); err != nil {
		return err
	}
	if responseBody == nil || len(envelope.Data) == 0 {
		return nil
	}
	if err := json.Unmarshal(envelope.Data, responseBody); err != nil {
		return fmt.Errorf("decode evolution envelope data: %w", err)
	}
	return nil
}

func (c *Client) sendMessage(ctx context.Context, path string, payload any) (SendMessageResponse, error) {
	var data evolutionGoSendMessageResponse
	if err := c.doEnvelope(ctx, http.MethodPost, path, payload, &data); err != nil {
		return SendMessageResponse{}, err
	}
	return SendMessageResponse{
		Key: MessageKey{
			ID: data.MessageID,
		},
	}, nil
}

type envelope[T any] struct {
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type evolutionGoInstanceSummary struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Connected bool   `json:"connected"`
	JID       string `json:"jid"`
	Status    string `json:"status"`
}

type evolutionGoCreateRequest struct {
	InstanceID       string                       `json:"instanceId,omitempty"`
	Name             string                       `json:"name"`
	Token            string                       `json:"token"`
	Proxy            *evolutionGoProxyConfig      `json:"proxy,omitempty"`
	AdvancedSettings *evolutionGoAdvancedSettings `json:"advancedSettings,omitempty"`
}

type evolutionGoProxyConfig struct {
	Protocol string `json:"protocol,omitempty"`
	Port     string `json:"port"`
	Password string `json:"password"`
	Username string `json:"username"`
	Host     string `json:"host"`
}

type evolutionGoAdvancedSettings struct {
	AlwaysOnline  bool   `json:"alwaysOnline"`
	IgnoreGroups  bool   `json:"ignoreGroups"`
	IgnoreStatus  bool   `json:"ignoreStatus"`
	MsgRejectCall string `json:"msgRejectCall,omitempty"`
	ReadMessages  bool   `json:"readMessages"`
	RejectCall    bool   `json:"rejectCall"`
}

type evolutionGoCreateResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Token  string `json:"token"`
	Status string `json:"status"`
}

type evolutionGoConnectRequest struct {
	WebhookURL string   `json:"webhookUrl,omitempty"`
	Subscribe  []string `json:"subscribe,omitempty"`
	Phone      string   `json:"phone,omitempty"`
}

type evolutionGoConnectResponse struct {
	JID         string `json:"jid"`
	WebhookURL  string `json:"webhookUrl"`
	EventString string `json:"eventString"`
}

type evolutionGoQRResponse struct {
	Qrcode string `json:"qrcode"`
	Code   string `json:"code"`
}

type evolutionGoStatusResponse struct {
	InstanceID        string `json:"id"`
	Name              string `json:"name"`
	Status            string `json:"status"`
	ProfileName       string `json:"profileName"`
	ProfilePictureURL string `json:"profilePictureUrl"`
	MyJID             string `json:"myJid"`
	JID               string `json:"jid"`
	Connected         bool   `json:"connected"`
	LoggedIn          bool   `json:"loggedIn"`
}

type evolutionGoQuotedRequest struct {
	MessageID   string `json:"messageId"`
	Participant string `json:"participant,omitempty"`
}

type evolutionGoTextRequest struct {
	Number string                    `json:"number"`
	Text   string                    `json:"text"`
	Delay  int                       `json:"delay,omitempty"`
	Quoted *evolutionGoQuotedRequest `json:"quoted,omitempty"`
}

type evolutionGoMediaRequest struct {
	Number   string `json:"number"`
	URL      string `json:"url"`
	Type     string `json:"type"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
	Delay    int    `json:"delay,omitempty"`
}

type evolutionGoStickerRequest struct {
	Number  string `json:"number"`
	Sticker string `json:"sticker"`
	Delay   int    `json:"delay,omitempty"`
}

type evolutionGoPresenceRequest struct {
	Number  string `json:"number"`
	State   string `json:"state"`
	IsAudio bool   `json:"isAudio"`
}

type evolutionGoReactionRequest struct {
	Number      string `json:"number"`
	Reaction    string `json:"reaction"`
	ID          string `json:"id"`
	FromMe      bool   `json:"fromMe"`
	Participant string `json:"participant,omitempty"`
}

type evolutionGoStatusTextRequest struct {
	Text string `json:"text"`
}

type evolutionGoStatusMediaRequest struct {
	Type    string `json:"type"`
	URL     string `json:"url"`
	Caption string `json:"caption,omitempty"`
}

type evolutionGoSendMessageResponse struct {
	MessageID string `json:"messageId"`
	Status    string `json:"status"`
}

func mapStatus(status string, connected bool) string {
	status = strings.TrimSpace(strings.ToLower(status))
	if status != "" {
		return status
	}
	if connected {
		return "open"
	}
	return "close"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
