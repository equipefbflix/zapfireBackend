package evolution

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientFetchInstances(t *testing.T) {
	var gotAPIKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.Header.Get("apikey")
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s", r.Method)
		}
		if r.URL.Path != "/instance/fetchInstances" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"name":"chip_1","connectionStatus":"open"}]`))
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL: server.URL,
		APIKey:  "secret",
		Timeout: time.Second,
	})

	instances, err := client.FetchInstances(context.Background(), "")
	if err != nil {
		t.Fatalf("FetchInstances() error = %v", err)
	}
	if gotAPIKey != "secret" {
		t.Fatalf("apikey = %q", gotAPIKey)
	}
	if len(instances) != 1 {
		t.Fatalf("instances len = %d", len(instances))
	}
	if instances[0].Name != "chip_1" {
		t.Fatalf("instance name = %q", instances[0].Name)
	}
}

func TestClientCreateInstanceWithProxy(t *testing.T) {
	var payload CreateInstanceRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		if r.URL.Path != "/instance/create" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"instance":{"instanceName":"chip_5511999999999"},"hash":{"apikey":"instance-key"}}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, APIKey: "secret", Timeout: time.Second})
	response, err := client.CreateInstance(context.Background(), CreateInstanceRequest{
		InstanceName:    "chip_5511999999999",
		Integration:     "WHATSAPP-BAILEYS",
		QRCode:          true,
		Number:          "5511999999999",
		RejectCall:      true,
		GroupsIgnore:    true,
		AlwaysOnline:    true,
		ReadMessages:    true,
		ReadStatus:      true,
		SyncFullHistory: false,
		ProxyHost:       "proxy.example.com",
		ProxyPort:       "8000",
		ProxyProtocol:   "http",
		ProxyUsername:   "user",
		ProxyPassword:   "pass",
	})
	if err != nil {
		t.Fatalf("CreateInstance() error = %v", err)
	}

	if payload.ProxyHost != "proxy.example.com" {
		t.Fatalf("proxy host = %q", payload.ProxyHost)
	}
	if payload.ProxyPort != "8000" {
		t.Fatalf("proxy port = %q", payload.ProxyPort)
	}
	if payload.ProxyProtocol != "http" {
		t.Fatalf("proxy protocol = %q", payload.ProxyProtocol)
	}
	if response.Hash.APIKey != "instance-key" {
		t.Fatalf("instance api key = %q", response.Hash.APIKey)
	}
}

func TestClientCreateInstanceAcceptsStringHash(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"instance":{"instanceName":"chip_5511999999999"},"hash":"instance-key"}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, APIKey: "secret", Timeout: time.Second})
	response, err := client.CreateInstance(context.Background(), CreateInstanceRequest{
		InstanceName: "chip_5511999999999",
		Integration:  "WHATSAPP-BAILEYS",
		QRCode:       true,
	})
	if err != nil {
		t.Fatalf("CreateInstance() error = %v", err)
	}
	if response.Hash.APIKey != "instance-key" {
		t.Fatalf("instance api key = %q", response.Hash.APIKey)
	}
}

func TestClientSendText(t *testing.T) {
	var payload SendTextRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/message/sendText/chip_1" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		_, _ = w.Write([]byte(`{"key":{"id":"message-id","remoteJid":"5511888888888@s.whatsapp.net","fromMe":true}}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, APIKey: "secret", Timeout: time.Second})
	response, err := client.SendText(context.Background(), "chip_1", SendTextRequest{
		Number:      "5511888888888",
		Text:        "Oi",
		Delay:       1200,
		LinkPreview: false,
	})
	if err != nil {
		t.Fatalf("SendText() error = %v", err)
	}
	if payload.Number != "5511888888888" {
		t.Fatalf("number = %q", payload.Number)
	}
	if response.Key.ID != "message-id" {
		t.Fatalf("message id = %q", response.Key.ID)
	}
}

func TestClientSendMedia(t *testing.T) {
	var payload SendMediaRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/message/sendMedia/chip_1" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		_, _ = w.Write([]byte(`{"key":{"id":"media-id","remoteJid":"5511888888888@s.whatsapp.net","fromMe":true}}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, APIKey: "secret", Timeout: time.Second})
	response, err := client.SendMedia(context.Background(), "chip_1", SendMediaRequest{
		Number:    "5511888888888",
		MediaType: "image",
		MimeType:  "image/png",
		Caption:   "Oi",
		Media:     "https://example.com/file.png",
		FileName:  "file.png",
		Delay:     1200,
	})
	if err != nil {
		t.Fatalf("SendMedia() error = %v", err)
	}
	if payload.Media != "https://example.com/file.png" {
		t.Fatalf("media = %q", payload.Media)
	}
	if response.Key.ID != "media-id" {
		t.Fatalf("message id = %q", response.Key.ID)
	}
}

func TestClientSendWhatsAppAudio(t *testing.T) {
	var payload SendWhatsAppAudioRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/message/sendWhatsAppAudio/chip_1" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		_, _ = w.Write([]byte(`{"key":{"id":"audio-id","remoteJid":"5511888888888@s.whatsapp.net","fromMe":true}}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, APIKey: "secret", Timeout: time.Second})
	response, err := client.SendWhatsAppAudio(context.Background(), "chip_1", SendWhatsAppAudioRequest{
		Number: "5511888888888",
		Audio:  "https://example.com/audio.ogg",
		Delay:  1200,
	})
	if err != nil {
		t.Fatalf("SendWhatsAppAudio() error = %v", err)
	}
	if payload.Audio != "https://example.com/audio.ogg" {
		t.Fatalf("audio = %q", payload.Audio)
	}
	if response.Key.ID != "audio-id" {
		t.Fatalf("message id = %q", response.Key.ID)
	}
}

func TestClientSendStatus(t *testing.T) {
	var payload SendStatusRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/message/sendStatus/chip_1" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		_, _ = w.Write([]byte(`{"key":{"id":"status-id","remoteJid":"status@broadcast","fromMe":true}}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, APIKey: "secret", Timeout: time.Second})
	response, err := client.SendStatus(context.Background(), "chip_1", SendStatusRequest{
		Type:        "text",
		Content:     "Bom dia",
		Background:  "#112233",
		Font:        2,
		AllContacts: true,
		StatusJIDList: []string{"5511888888888@s.whatsapp.net"},
		Delay:       800,
		LinkPreview: false,
	})
	if err != nil {
		t.Fatalf("SendStatus() error = %v", err)
	}
	if payload.Content != "Bom dia" {
		t.Fatalf("content = %q", payload.Content)
	}
	if payload.Type != "text" {
		t.Fatalf("type = %q", payload.Type)
	}
	if !payload.AllContacts {
		t.Fatal("allContacts = false")
	}
	if len(payload.StatusJIDList) != 1 || payload.StatusJIDList[0] != "5511888888888@s.whatsapp.net" {
		t.Fatalf("statusJidList = %+v", payload.StatusJIDList)
	}
	if response.Key.ID != "status-id" {
		t.Fatalf("message id = %q", response.Key.ID)
	}
}

func TestClientSendStatusAcceptsAsyncTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, APIKey: "secret", Timeout: 10 * time.Millisecond})
	response, err := client.SendStatus(context.Background(), "chip_1", SendStatusRequest{
		Type:        "text",
		Content:     "Bom dia",
		AllContacts: true,
	})
	if err != nil {
		t.Fatalf("SendStatus() error = %v", err)
	}
	if !response.AcceptedAsync {
		t.Fatal("AcceptedAsync = false")
	}
	if response.Key.ID != "" {
		t.Fatalf("message id = %q", response.Key.ID)
	}
}

func TestClientSendStatusPropagatesBadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"bad request"}`, http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, APIKey: "secret", Timeout: time.Second})
	_, err := client.SendStatus(context.Background(), "chip_1", SendStatusRequest{
		Type:        "text",
		Content:     "Bom dia",
		AllContacts: true,
	})
	if err == nil {
		t.Fatal("SendStatus() error = nil")
	}
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("error = %T, want *HTTPError", err)
	}
}

func TestClientSendSticker(t *testing.T) {
	var payload SendStickerRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/message/sendSticker/chip_1" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, APIKey: "secret", Timeout: time.Second})
	if err := client.SendSticker(context.Background(), "chip_1", SendStickerRequest{
		Number:  "5511888888888",
		Sticker: "https://example.com/sticker.webp",
		Delay:   1200,
	}); err != nil {
		t.Fatalf("SendSticker() error = %v", err)
	}
	if payload.Sticker != "https://example.com/sticker.webp" {
		t.Fatalf("sticker = %q", payload.Sticker)
	}
}

func TestClientSendPresence(t *testing.T) {
	var payload SendPresenceRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/sendPresence/chip_1" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, APIKey: "secret", Timeout: time.Second})
	if err := client.SendPresence(context.Background(), "chip_1", SendPresenceRequest{
		Number:   "5511888888888",
		Delay:    1200,
		Presence: "composing",
	}); err != nil {
		t.Fatalf("SendPresence() error = %v", err)
	}
	if payload.Number != "5511888888888" {
		t.Fatalf("number = %q", payload.Number)
	}
	if payload.Delay != 1200 {
		t.Fatalf("delay = %d", payload.Delay)
	}
	if payload.Presence != "composing" {
		t.Fatalf("presence = %q", payload.Presence)
	}
}

func TestClientDeleteInstance(t *testing.T) {
	var gotMethod string
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"status":"SUCCESS","error":false,"response":{"message":"Instance deleted"}}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, APIKey: "secret", Timeout: time.Second})
	if err := client.DeleteInstance(context.Background(), "chip_1"); err != nil {
		t.Fatalf("DeleteInstance() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Fatalf("method = %s", gotMethod)
	}
	if gotPath != "/instance/delete/chip_1" {
		t.Fatalf("path = %s", gotPath)
	}
}

func TestClientReturnsHTTPErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad api key", http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, APIKey: "secret", Timeout: time.Second})
	_, err := client.FetchInstances(context.Background(), "")
	if err == nil {
		t.Fatal("FetchInstances() error = nil, want error")
	}

	httpErr, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if httpErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status code = %d", httpErr.StatusCode)
	}
	if httpErr.Body == "" {
		t.Fatal("body is empty")
	}
}
