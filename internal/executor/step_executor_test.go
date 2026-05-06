package executor

import (
	"context"
	"testing"

	"aquecedor-evolution/backend/internal/evolution"
	"aquecedor-evolution/backend/internal/repository"
)

type fakeEvolutionStepClient struct {
	textInstance     string
	textRequest      evolution.SendTextRequest
	mediaInstance    string
	mediaRequest     evolution.SendMediaRequest
	audioInstance    string
	audioRequest     evolution.SendWhatsAppAudioRequest
	statusInstance   string
	statusRequest    evolution.SendStatusRequest
	stickerInstance  string
	stickerRequest   evolution.SendStickerRequest
	presenceInstance string
	presenceRequest  evolution.SendPresenceRequest
	reactionInstance string
	reactionRequest  evolution.SendReactionRequest
}

func (c *fakeEvolutionStepClient) SendText(ctx context.Context, instanceName string, request evolution.SendTextRequest) (evolution.SendMessageResponse, error) {
	c.textInstance = instanceName
	c.textRequest = request
	return evolution.SendMessageResponse{Key: evolution.MessageKey{ID: "message-id"}}, nil
}

func (c *fakeEvolutionStepClient) SendMedia(ctx context.Context, instanceName string, request evolution.SendMediaRequest) (evolution.SendMessageResponse, error) {
	c.mediaInstance = instanceName
	c.mediaRequest = request
	return evolution.SendMessageResponse{Key: evolution.MessageKey{ID: "media-id"}}, nil
}

func (c *fakeEvolutionStepClient) SendWhatsAppAudio(ctx context.Context, instanceName string, request evolution.SendWhatsAppAudioRequest) (evolution.SendMessageResponse, error) {
	c.audioInstance = instanceName
	c.audioRequest = request
	return evolution.SendMessageResponse{Key: evolution.MessageKey{ID: "audio-id"}}, nil
}

func (c *fakeEvolutionStepClient) SendStatus(ctx context.Context, instanceName string, request evolution.SendStatusRequest) (evolution.SendMessageResponse, error) {
	c.statusInstance = instanceName
	c.statusRequest = request
	return evolution.SendMessageResponse{Key: evolution.MessageKey{ID: "status-id"}}, nil
}

func (c *fakeEvolutionStepClient) SendSticker(ctx context.Context, instanceName string, request evolution.SendStickerRequest) error {
	c.stickerInstance = instanceName
	c.stickerRequest = request
	return nil
}

func (c *fakeEvolutionStepClient) SendPresence(ctx context.Context, instanceName string, request evolution.SendPresenceRequest) error {
	c.presenceInstance = instanceName
	c.presenceRequest = request
	return nil
}

func (c *fakeEvolutionStepClient) SendReaction(ctx context.Context, instanceName string, request evolution.SendReactionRequest) error {
	c.reactionInstance = instanceName
	c.reactionRequest = request
	return nil
}

func TestStepExecutorSendText(t *testing.T) {
	client := &fakeEvolutionStepClient{}
	executor := NewStepExecutor(client)
	step := repository.ConversationStep{
		ActionType: "send_text",
		Payload: map[string]any{
			"number":      "5511999999999",
			"text":        "Bom dia",
			"delay":       float64(1000),
			"linkPreview": true,
		},
	}

	result, err := executor.Execute(context.Background(), "chip-sp-01", step)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if client.textInstance != "chip-sp-01" {
		t.Fatalf("instance = %q", client.textInstance)
	}
	if client.textRequest.Text != "Bom dia" {
		t.Fatalf("Text = %q", client.textRequest.Text)
	}
	if result.MessageKey.ID != "message-id" {
		t.Fatalf("MessageKey.ID = %q", result.MessageKey.ID)
	}
}

func TestStepExecutorSendPresence(t *testing.T) {
	client := &fakeEvolutionStepClient{}
	executor := NewStepExecutor(client)
	step := repository.ConversationStep{
		ActionType: "send_presence",
		Payload: map[string]any{
			"number":   "5511999999999",
			"presence": "composing",
			"delay":    float64(1000),
		},
	}

	if _, err := executor.Execute(context.Background(), "chip-sp-01", step); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if client.presenceRequest.Presence != "composing" {
		t.Fatalf("Presence = %q", client.presenceRequest.Presence)
	}
	if client.presenceRequest.Delay != 1000 {
		t.Fatalf("Delay = %d", client.presenceRequest.Delay)
	}
}

func TestStepExecutorSendReaction(t *testing.T) {
	client := &fakeEvolutionStepClient{}
	executor := NewStepExecutor(client)
	step := repository.ConversationStep{
		ActionType: "send_reaction",
		Payload: map[string]any{
			"remoteJid": "5511999999999@s.whatsapp.net",
			"messageId": "message-id",
			"fromMe":    true,
			"reaction":  "👍",
		},
	}

	if _, err := executor.Execute(context.Background(), "chip-sp-01", step); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if client.reactionRequest.Key.ID != "message-id" {
		t.Fatalf("message id = %q", client.reactionRequest.Key.ID)
	}
	if client.reactionRequest.Reaction != "👍" {
		t.Fatalf("Reaction = %q", client.reactionRequest.Reaction)
	}
}

func TestStepExecutorSendReply(t *testing.T) {
	client := &fakeEvolutionStepClient{}
	executor := NewStepExecutor(client)
	step := repository.ConversationStep{
		ActionType: "send_reply",
		Payload: map[string]any{
			"number":      "5511999999999",
			"text":        "Respondi aqui",
			"remoteJid":   "5511999999999@s.whatsapp.net",
			"messageId":   "message-id",
			"fromMe":      false,
			"delay":       float64(1000),
			"linkPreview": false,
		},
	}

	if _, err := executor.Execute(context.Background(), "chip-sp-01", step); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if client.textRequest.Quoted == nil {
		t.Fatal("Quoted = nil")
	}
	if client.textRequest.Quoted.Key.ID != "message-id" {
		t.Fatalf("quoted message id = %q", client.textRequest.Quoted.Key.ID)
	}
	if client.textRequest.Text != "Respondi aqui" {
		t.Fatalf("Text = %q", client.textRequest.Text)
	}
}

func TestStepExecutorSendMedia(t *testing.T) {
	client := &fakeEvolutionStepClient{}
	executor := NewStepExecutor(client)
	step := repository.ConversationStep{
		ActionType: "send_media",
		Payload: map[string]any{
			"number":    "5511999999999",
			"mediatype": "image",
			"mimetype":  "image/png",
			"caption":   "Bom dia",
			"media":     "https://example.com/file.png",
			"fileName":  "file.png",
			"delay":     float64(1000),
		},
	}

	result, err := executor.Execute(context.Background(), "chip-sp-01", step)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if client.mediaRequest.Media != "https://example.com/file.png" {
		t.Fatalf("media = %q", client.mediaRequest.Media)
	}
	if result.MessageKey == nil || result.MessageKey.ID != "media-id" {
		t.Fatalf("message key = %+v", result.MessageKey)
	}
}

func TestStepExecutorSendSticker(t *testing.T) {
	client := &fakeEvolutionStepClient{}
	executor := NewStepExecutor(client)
	step := repository.ConversationStep{
		ActionType: "send_sticker",
		Payload: map[string]any{
			"number":  "5511999999999",
			"sticker": "https://example.com/sticker.webp",
			"delay":   float64(1000),
		},
	}

	if _, err := executor.Execute(context.Background(), "chip-sp-01", step); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if client.stickerRequest.Sticker != "https://example.com/sticker.webp" {
		t.Fatalf("sticker = %q", client.stickerRequest.Sticker)
	}
}

func TestStepExecutorSendAudio(t *testing.T) {
	client := &fakeEvolutionStepClient{}
	executor := NewStepExecutor(client)
	step := repository.ConversationStep{
		ActionType: "send_audio",
		Payload: map[string]any{
			"number": "5511999999999",
			"audio":  "https://example.com/audio.ogg",
			"delay":  float64(1500),
		},
	}

	result, err := executor.Execute(context.Background(), "chip-sp-01", step)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if client.audioRequest.Audio != "https://example.com/audio.ogg" {
		t.Fatalf("audio = %q", client.audioRequest.Audio)
	}
	if result.MessageKey == nil || result.MessageKey.ID != "audio-id" {
		t.Fatalf("message key = %+v", result.MessageKey)
	}
}

func TestStepExecutorSendStatus(t *testing.T) {
	client := &fakeEvolutionStepClient{}
	executor := NewStepExecutor(client)
	step := repository.ConversationStep{
		ActionType: "send_status",
		Payload: map[string]any{
			"type":        "text",
			"content":     "Bom dia",
			"background":  "#112233",
			"font":        float64(2),
			"allContacts": true,
			"statusJidList": []any{"5511888888888@s.whatsapp.net"},
			"delay":       float64(700),
			"linkPreview": true,
		},
	}

	result, err := executor.Execute(context.Background(), "chip-sp-01", step)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if client.statusRequest.Content != "Bom dia" {
		t.Fatalf("content = %q", client.statusRequest.Content)
	}
	if client.statusRequest.Type != "text" {
		t.Fatalf("type = %q", client.statusRequest.Type)
	}
	if !client.statusRequest.AllContacts {
		t.Fatal("allContacts = false")
	}
	if len(client.statusRequest.StatusJIDList) != 1 || client.statusRequest.StatusJIDList[0] != "5511888888888@s.whatsapp.net" {
		t.Fatalf("statusJidList = %+v", client.statusRequest.StatusJIDList)
	}
	if result.MessageKey == nil || result.MessageKey.ID != "status-id" {
		t.Fatalf("message key = %+v", result.MessageKey)
	}
}

func TestStepExecutorSendStatusAcceptsAsyncResponse(t *testing.T) {
	executor := NewStepExecutor(&statusAsyncFakeClient{})
	step := repository.ConversationStep{
		ActionType: "send_status",
		Payload: map[string]any{
			"type":        "text",
			"content":     "Bom dia",
			"allContacts": true,
		},
	}

	result, err := executor.Execute(context.Background(), "chip-sp-01", step)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.MessageKey != nil {
		t.Fatalf("message key = %+v", result.MessageKey)
	}
	if !result.AcceptedAsync {
		t.Fatal("AcceptedAsync = false")
	}
	if result.ResponsePayload["acceptedAsync"] != true {
		t.Fatalf("response payload = %+v", result.ResponsePayload)
	}
}

type statusAsyncFakeClient struct{ fakeEvolutionStepClient }

func (c statusAsyncFakeClient) SendStatus(ctx context.Context, instanceName string, request evolution.SendStatusRequest) (evolution.SendMessageResponse, error) {
	return evolution.SendMessageResponse{AcceptedAsync: true}, nil
}

func TestStepExecutorSendTyping(t *testing.T) {
	client := &fakeEvolutionStepClient{}
	executor := NewStepExecutor(client)
	step := repository.ConversationStep{
		ActionType: "send_typing",
		Payload: map[string]any{
			"number": "5511999999999",
			"delay":  float64(1200),
		},
	}

	if _, err := executor.Execute(context.Background(), "chip-sp-01", step); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if client.presenceRequest.Presence != "composing" {
		t.Fatalf("presence = %q", client.presenceRequest.Presence)
	}
	if client.presenceRequest.Delay != 1200 {
		t.Fatalf("delay = %d", client.presenceRequest.Delay)
	}
}

func TestStepExecutorSendRecording(t *testing.T) {
	client := &fakeEvolutionStepClient{}
	executor := NewStepExecutor(client)
	step := repository.ConversationStep{
		ActionType: "send_recording",
		Payload: map[string]any{
			"number": "5511999999999",
			"delay":  float64(900),
		},
	}

	if _, err := executor.Execute(context.Background(), "chip-sp-01", step); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if client.presenceRequest.Presence != "recording" {
		t.Fatalf("presence = %q", client.presenceRequest.Presence)
	}
	if client.presenceRequest.Delay != 900 {
		t.Fatalf("delay = %d", client.presenceRequest.Delay)
	}
}

func TestStepExecutorRejectsUnsupportedAction(t *testing.T) {
	executor := NewStepExecutor(&fakeEvolutionStepClient{})

	if _, err := executor.Execute(context.Background(), "chip-sp-01", repository.ConversationStep{ActionType: "unsupported_action"}); err == nil {
		t.Fatal("Execute() error = nil, want error")
	}
}
