package executor

import (
	"context"
	"fmt"

	"aquecedor-evolution/backend/internal/evolution"
	"aquecedor-evolution/backend/internal/repository"
)

type EvolutionStepClient interface {
	SendText(ctx context.Context, instanceName string, request evolution.SendTextRequest) (evolution.SendMessageResponse, error)
	SendMedia(ctx context.Context, instanceName string, request evolution.SendMediaRequest) (evolution.SendMessageResponse, error)
	SendWhatsAppAudio(ctx context.Context, instanceName string, request evolution.SendWhatsAppAudioRequest) (evolution.SendMessageResponse, error)
	SendStatus(ctx context.Context, instanceName string, request evolution.SendStatusRequest) (evolution.SendMessageResponse, error)
	SendSticker(ctx context.Context, instanceName string, request evolution.SendStickerRequest) error
	SendPresence(ctx context.Context, instanceName string, request evolution.SendPresenceRequest) error
	SendReaction(ctx context.Context, instanceName string, request evolution.SendReactionRequest) error
}

type StepResult struct {
	MessageKey     *evolution.MessageKey
	AcceptedAsync  bool
	ResponsePayload map[string]any
}

type StepExecutor struct {
	client EvolutionStepClient
}

func NewStepExecutor(client EvolutionStepClient) StepExecutor {
	return StepExecutor{client: client}
}

func (e StepExecutor) Execute(ctx context.Context, instanceName string, step repository.ConversationStep) (StepResult, error) {
	switch step.ActionType {
	case "send_text":
		response, err := e.client.SendText(ctx, instanceName, evolution.SendTextRequest{
			Number:      requiredString(step.Payload, "number"),
			Text:        requiredString(step.Payload, "text"),
			Delay:       intField(step.Payload, "delay"),
			LinkPreview: boolField(step.Payload, "linkPreview"),
		})
		if err != nil {
			return StepResult{}, err
		}
		return stepResultFromMessageResponse(response), nil
	case "send_reply":
		response, err := e.client.SendText(ctx, instanceName, evolution.SendTextRequest{
			Number:      requiredString(step.Payload, "number"),
			Text:        requiredString(step.Payload, "text"),
			Delay:       intField(step.Payload, "delay"),
			LinkPreview: boolField(step.Payload, "linkPreview"),
			Quoted: &evolution.QuotedInfo{
				Key: evolution.MessageKey{
					RemoteJID: requiredString(step.Payload, "remoteJid"),
					FromMe:    boolField(step.Payload, "fromMe"),
					ID:        requiredString(step.Payload, "messageId"),
				},
			},
		})
		if err != nil {
			return StepResult{}, err
		}
		return stepResultFromMessageResponse(response), nil
	case "send_presence":
		return e.sendPresence(ctx, instanceName, step, requiredString(step.Payload, "presence"))
	case "send_typing":
		return e.sendPresence(ctx, instanceName, step, "composing")
	case "send_recording":
		return e.sendPresence(ctx, instanceName, step, "recording")
	case "send_audio":
		response, err := e.client.SendWhatsAppAudio(ctx, instanceName, evolution.SendWhatsAppAudioRequest{
			Number: requiredString(step.Payload, "number"),
			Audio:  requiredString(step.Payload, "audio"),
			Delay:  intField(step.Payload, "delay"),
		})
		if err != nil {
			return StepResult{}, err
		}
		return stepResultFromMessageResponse(response), nil
	case "send_status":
		response, err := e.client.SendStatus(ctx, instanceName, evolution.SendStatusRequest{
			Type:        requiredString(step.Payload, "type"),
			Content:     requiredString(step.Payload, "content"),
			Caption:     requiredString(step.Payload, "caption"),
			Background:  requiredString(step.Payload, "background"),
			Font:        intField(step.Payload, "font"),
			AllContacts: boolField(step.Payload, "allContacts"),
			StatusJIDList: stringSliceField(step.Payload, "statusJidList"),
			Media:       requiredString(step.Payload, "media"),
			Delay:       intField(step.Payload, "delay"),
			LinkPreview: boolField(step.Payload, "linkPreview"),
		})
		if err != nil {
			return StepResult{}, err
		}
		return stepResultFromMessageResponse(response), nil
	case "send_media":
		response, err := e.client.SendMedia(ctx, instanceName, evolution.SendMediaRequest{
			Number:    requiredString(step.Payload, "number"),
			MediaType: requiredString(step.Payload, "mediatype"),
			MimeType:  requiredString(step.Payload, "mimetype"),
			Caption:   requiredString(step.Payload, "caption"),
			Media:     requiredString(step.Payload, "media"),
			FileName:  requiredString(step.Payload, "fileName"),
			Delay:     intField(step.Payload, "delay"),
		})
		if err != nil {
			return StepResult{}, err
		}
		return stepResultFromMessageResponse(response), nil
	case "send_sticker":
		if err := e.client.SendSticker(ctx, instanceName, evolution.SendStickerRequest{
			Number:  requiredString(step.Payload, "number"),
			Sticker: requiredString(step.Payload, "sticker"),
			Delay:   intField(step.Payload, "delay"),
		}); err != nil {
			return StepResult{}, err
		}
		return StepResult{}, nil
	case "send_reaction":
		request := evolution.SendReactionRequest{
			Key: evolution.MessageKey{
				RemoteJID: requiredString(step.Payload, "remoteJid"),
				FromMe:    boolField(step.Payload, "fromMe"),
				ID:        requiredString(step.Payload, "messageId"),
			},
			Reaction: requiredString(step.Payload, "reaction"),
		}
		if err := e.client.SendReaction(ctx, instanceName, request); err != nil {
			return StepResult{}, err
		}
		return StepResult{}, nil
	default:
		return StepResult{}, fmt.Errorf("unsupported action type %q", step.ActionType)
	}
}

func (e StepExecutor) sendPresence(ctx context.Context, instanceName string, step repository.ConversationStep, presence string) (StepResult, error) {
	request := evolution.SendPresenceRequest{
		Number:   requiredString(step.Payload, "number"),
		Delay:    intField(step.Payload, "delay"),
		Presence: presence,
	}
	if err := e.client.SendPresence(ctx, instanceName, request); err != nil {
		return StepResult{}, err
	}
	return StepResult{}, nil
}

func stepResultFromMessageResponse(response evolution.SendMessageResponse) StepResult {
	result := StepResult{}
	if response.Key.ID != "" {
		result.MessageKey = &response.Key
	}
	if response.AcceptedAsync {
		result.AcceptedAsync = true
		result.ResponsePayload = map[string]any{"acceptedAsync": true}
	}
	return result
}

func requiredString(payload map[string]any, key string) string {
	value, _ := payload[key].(string)
	return value
}

func intField(payload map[string]any, key string) int {
	value, ok := payload[key]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func boolField(payload map[string]any, key string) bool {
	value, _ := payload[key].(bool)
	return value
}

func stringSliceField(payload map[string]any, key string) []string {
	value, ok := payload[key]
	if !ok {
		return nil
	}
	items, ok := value.([]any)
	if !ok {
		if typed, ok := value.([]string); ok {
			return typed
		}
		return nil
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		if text, ok := item.(string); ok {
			result = append(result, text)
		}
	}
	return result
}
