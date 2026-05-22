package evolution

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"
)

func TestClientSendStatusPropagatesBadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/send/status/text" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		http.Error(w, `{"message":"bad request"}`, http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, APIKey: "secret", Timeout: time.Second})
	_, err := client.SendStatus(context.Background(), "chip_1", SendStatusRequest{
		Type:    "text",
		Content: "Bom dia",
	})
	if err == nil {
		t.Fatal("SendStatus() error = nil")
	}
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("error = %T, want *HTTPError", err)
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

func TestClientFetchInstancesEvolutionGo(t *testing.T) {
	var gotAPIKey string
	var gotAuthorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.Header.Get("apikey")
		gotAuthorization = r.Header.Get("Authorization")
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s", r.Method)
		}
		if r.URL.Path != "/instance/all" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"success","data":[{"id":"abc123","name":"chip_1","connected":true,"jid":"5511888888888@s.whatsapp.net"}]}`))
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
	if gotAuthorization != "Bearer secret" {
		t.Fatalf("authorization = %q", gotAuthorization)
	}
	if len(instances) != 1 {
		t.Fatalf("instances len = %d", len(instances))
	}
	if instances[0].Name != "chip_1" {
		t.Fatalf("instance name = %q", instances[0].Name)
	}
}

func TestClientCreateInstanceEvolutionGo(t *testing.T) {
	var payload map[string]any
	uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
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
		_, _ = w.Write([]byte(`{"message":"success","data":{"instanceId":"uuid-gerado","name":"chip_5511999999999","token":"instance-token","status":"created"}}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, APIKey: "secret", Timeout: time.Second})
	_, err := client.CreateInstance(context.Background(), CreateInstanceRequest{
		InstanceName:    "chip_5511999999999",
		Token:           "instance-token",
		RejectCall:      true,
		MsgCall:         "Nao posso atender agora.",
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

	if payload["name"] != "chip_5511999999999" {
		t.Fatalf("name = %#v", payload["name"])
	}
	instanceID, _ := payload["instanceId"].(string)
	if !uuidPattern.MatchString(instanceID) {
		t.Fatalf("instanceId = %q, want generated uuid", instanceID)
	}
	if instanceID == "chip_5511999999999" {
		t.Fatalf("instanceId should not reuse instance name: %q", instanceID)
	}
	if payload["token"] != "instance-token" {
		t.Fatalf("token = %#v", payload["token"])
	}

	proxy, ok := payload["proxy"].(map[string]any)
	if !ok {
		t.Fatalf("proxy = %#v", payload["proxy"])
	}
	if proxy["host"] != "proxy.example.com" {
		t.Fatalf("proxy.host = %#v", proxy["host"])
	}
	if proxy["port"] != "8000" {
		t.Fatalf("proxy.port = %#v", proxy["port"])
	}

	settings, ok := payload["advancedSettings"].(map[string]any)
	if !ok {
		t.Fatalf("advancedSettings = %#v", payload["advancedSettings"])
	}
	if settings["rejectCall"] != true {
		t.Fatalf("advancedSettings.rejectCall = %#v", settings["rejectCall"])
	}
	if settings["ignoreGroups"] != true {
		t.Fatalf("advancedSettings.ignoreGroups = %#v", settings["ignoreGroups"])
	}
}

func TestClientConnectInstanceEvolutionGo(t *testing.T) {
	var payload map[string]any
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch callCount {
		case 1:
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s", r.Method)
			}
			if r.URL.Path != "/instance/connect" {
				t.Fatalf("path = %s", r.URL.Path)
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"message":"success","data":{"jid":"5511888888888@s.whatsapp.net","webhookUrl":"https://backend.example.com/webhooks/evolution","eventString":"messages.upsert,connection.update"}}`))
		case 2:
			if r.Method != http.MethodGet {
				t.Fatalf("method = %s", r.Method)
			}
			if r.URL.Path != "/instance/qr" {
				t.Fatalf("path = %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"message":"success","data":{"qrcode":"2@abcd1234","code":"data:image/png;base64,AAA"}}`))
		default:
			t.Fatalf("unexpected call %d: %s %s", callCount, r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:    server.URL,
		APIKey:     "instance-token",
		Timeout:    time.Second,
		WebhookURL: "https://pairing.ngrok-free.app/api/v1/webhooks/evolution",
	})
	response, err := client.ConnectInstance(context.Background(), "chip_1", "")
	if err != nil {
		t.Fatalf("ConnectInstance() error = %v", err)
	}
	if payload["webhookUrl"] != "https://pairing.ngrok-free.app/api/v1/webhooks/evolution" {
		t.Fatalf("webhookUrl = %#v", payload["webhookUrl"])
	}
	subscribe, ok := payload["subscribe"].([]any)
	if !ok {
		t.Fatalf("subscribe = %#v", payload["subscribe"])
	}
	if len(subscribe) != 2 {
		t.Fatalf("subscribe len = %d", len(subscribe))
	}
	if subscribe[0] != "messages.upsert" || subscribe[1] != "connection.update" {
		t.Fatalf("subscribe = %#v", subscribe)
	}
	if response.PairingCode != "2@abcd1234" {
		t.Fatalf("pairing code = %q", response.PairingCode)
	}
	if response.Code != "data:image/png;base64,AAA" {
		t.Fatalf("code = %q", response.Code)
	}
}

func TestClientConnectInstanceEvolutionGoFallsBackToQRCodeWhenConnectFails(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch callCount {
		case 1:
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s", r.Method)
			}
			if r.URL.Path != "/instance/connect" {
				t.Fatalf("path = %s", r.URL.Path)
			}
			http.Error(w, `{"message":"pairing session already exists"}`, http.StatusInternalServerError)
		case 2:
			if r.Method != http.MethodGet {
				t.Fatalf("method = %s", r.Method)
			}
			if r.URL.Path != "/instance/qr" {
				t.Fatalf("path = %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"message":"success","data":{"qrcode":"data:image/png;base64,REFRESH","code":"2@REFRESH"}}`))
		default:
			t.Fatalf("unexpected call %d: %s %s", callCount, r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL: server.URL,
		APIKey:  "instance-token",
		Timeout: time.Second,
	})
	response, err := client.ConnectInstance(context.Background(), "chip_1", "")
	if err != nil {
		t.Fatalf("ConnectInstance() error = %v", err)
	}
	if response.PairingCode != "data:image/png;base64,REFRESH" {
		t.Fatalf("pairing code = %q", response.PairingCode)
	}
	if response.Code != "2@REFRESH" {
		t.Fatalf("code = %q", response.Code)
	}
}

func TestClientConnectInstanceReturnsConnectDataWhenQRFails(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch callCount {
		case 1:
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s", r.Method)
			}
			if r.URL.Path != "/instance/connect" {
				t.Fatalf("path = %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"message":"success","data":{"jid":"5511888888888@s.whatsapp.net","webhookUrl":"https://example.com/webhook","eventString":"messages.upsert"}}`))
		case 2:
			if r.Method != http.MethodGet {
				t.Fatalf("method = %s", r.Method)
			}
			if r.URL.Path != "/instance/qr" {
				t.Fatalf("path = %s", r.URL.Path)
			}
			http.Error(w, `{"error":"session already logged in"}`, http.StatusBadRequest)
		default:
			t.Fatalf("unexpected call %d: %s %s", callCount, r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL: server.URL,
		APIKey:  "instance-token",
		Timeout: time.Second,
	})
	response, err := client.ConnectInstance(context.Background(), "chip_1", "")
	if err != nil {
		t.Fatalf("ConnectInstance() error = %v", err)
	}
	if response.JID != "5511888888888@s.whatsapp.net" {
		t.Fatalf("jid = %q", response.JID)
	}
	if response.EventString != "messages.upsert" {
		t.Fatalf("eventString = %q", response.EventString)
	}
	if response.WebhookURL != "https://example.com/webhook" {
		t.Fatalf("webhookUrl = %q", response.WebhookURL)
	}
}

func TestClientConnectionStateEvolutionGo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s", r.Method)
		}
		if r.URL.Path != "/instance/all" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"uuid-gerado","name":"chip_1","status":"open","profileName":"Chip 1"}]}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, APIKey: "instance-token", Timeout: time.Second})
	response, err := client.ConnectionState(context.Background(), "chip_1")
	if err != nil {
		t.Fatalf("ConnectionState() error = %v", err)
	}
	if response.Instance.State != "open" {
		t.Fatalf("state = %q", response.Instance.State)
	}
}

func TestClientSendTextEvolutionGo(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/send/text" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		_, _ = w.Write([]byte(`{"message":"success","data":{"messageId":"message-id","status":"sent"}}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, APIKey: "instance-token", Timeout: time.Second})
	response, err := client.SendText(context.Background(), "chip_1", SendTextRequest{
		Number:      "5511888888888",
		Text:        "Oi",
		Delay:       1200,
		LinkPreview: false,
	})
	if err != nil {
		t.Fatalf("SendText() error = %v", err)
	}
	if payload["number"] != "5511888888888" {
		t.Fatalf("number = %#v", payload["number"])
	}
	if payload["text"] != "Oi" {
		t.Fatalf("text = %#v", payload["text"])
	}
	if response.Key.ID != "message-id" {
		t.Fatalf("message id = %q", response.Key.ID)
	}
}

func TestClientSendMediaEvolutionGo(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/send/media" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		_, _ = w.Write([]byte(`{"message":"success","data":{"messageId":"media-id","status":"sent"}}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, APIKey: "instance-token", Timeout: time.Second})
	response, err := client.SendMedia(context.Background(), "chip_1", SendMediaRequest{
		Number:    "5511888888888",
		MediaType: "image",
		Caption:   "Oi",
		Media:     "https://example.com/file.png",
		FileName:  "file.png",
		Delay:     1200,
	})
	if err != nil {
		t.Fatalf("SendMedia() error = %v", err)
	}
	if payload["url"] != "https://example.com/file.png" {
		t.Fatalf("url = %#v", payload["url"])
	}
	if payload["type"] != "image" {
		t.Fatalf("type = %#v", payload["type"])
	}
	if payload["filename"] != "file.png" {
		t.Fatalf("filename = %#v", payload["filename"])
	}
	if response.Key.ID != "media-id" {
		t.Fatalf("message id = %q", response.Key.ID)
	}
}

func TestClientSendStickerEvolutionGo(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/send/sticker" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"success","data":{"messageId":"sticker-id","status":"sent"}}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, APIKey: "instance-token", Timeout: time.Second})
	if err := client.SendSticker(context.Background(), "chip_1", SendStickerRequest{
		Number:  "5511888888888",
		Sticker: "https://example.com/sticker.webp",
		Delay:   1200,
	}); err != nil {
		t.Fatalf("SendSticker() error = %v", err)
	}
	if payload["sticker"] != "https://example.com/sticker.webp" {
		t.Fatalf("sticker = %#v", payload["sticker"])
	}
}

func TestClientSendPresenceEvolutionGo(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/message/presence" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, APIKey: "instance-token", Timeout: time.Second})
	if err := client.SendPresence(context.Background(), "chip_1", SendPresenceRequest{
		Number:   "5511888888888",
		Presence: "composing",
	}); err != nil {
		t.Fatalf("SendPresence() error = %v", err)
	}
	if payload["state"] != "composing" {
		t.Fatalf("state = %#v", payload["state"])
	}
	if payload["isAudio"] != false {
		t.Fatalf("isAudio = %#v", payload["isAudio"])
	}
}

func TestClientSendReactionEvolutionGo(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/message/react" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"success","data":{"messageId":"reaction-id","status":"sent"}}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, APIKey: "instance-token", Timeout: time.Second})
	if err := client.SendReaction(context.Background(), "chip_1", SendReactionRequest{
		Key: MessageKey{
			RemoteJID: "5511888888888@s.whatsapp.net",
			FromMe:    true,
			ID:        "message-id",
		},
		Reaction: "👍",
	}); err != nil {
		t.Fatalf("SendReaction() error = %v", err)
	}
	if payload["number"] != "5511888888888@s.whatsapp.net" {
		t.Fatalf("number = %#v", payload["number"])
	}
	if payload["id"] != "message-id" {
		t.Fatalf("id = %#v", payload["id"])
	}
	if payload["reaction"] != "👍" {
		t.Fatalf("reaction = %#v", payload["reaction"])
	}
}

func TestClientSendStatusTextEvolutionGo(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/send/status/text" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		_, _ = w.Write([]byte(`{"message":"success","data":{"messageId":"status-id","status":"sent"}}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, APIKey: "instance-token", Timeout: time.Second})
	response, err := client.SendStatus(context.Background(), "chip_1", SendStatusRequest{
		Type:       "text",
		Content:    "Bom dia",
		Background: "#112233",
		Font:       2,
	})
	if err != nil {
		t.Fatalf("SendStatus() error = %v", err)
	}
	if payload["text"] != "Bom dia" {
		t.Fatalf("text = %#v", payload["text"])
	}
	if response.Key.ID != "status-id" {
		t.Fatalf("message id = %q", response.Key.ID)
	}
}
