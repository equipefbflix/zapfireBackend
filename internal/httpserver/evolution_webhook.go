package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"aquecedor-evolution/backend/internal/repository"
)

type evolutionWebhookResponse struct {
	ID           string `json:"id"`
	EventType    string `json:"eventType"`
	InstanceName string `json:"instanceName"`
}

func (s *Server) handleEvolutionWebhook(w http.ResponseWriter, r *http.Request) {
	if s.cfg.EvolutionEvents == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "evolution event store is not configured"})
		return
	}
	if s.cfg.App.WebhookEvolutionSecret != "" && r.Header.Get("X-Webhook-Secret") != s.cfg.App.WebhookEvolutionSecret {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "invalid webhook secret"})
		return
	}

	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json body"})
		return
	}

	eventType := stringField(payload, "event")
	if eventType == "" {
		eventType = stringField(payload, "eventType")
	}
	if eventType == "" {
		eventType = webhookEventTypeFromPath(r.URL.Path)
	}
	if eventType == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "event or eventType is required"})
		return
	}
	eventType = normalizeEvolutionEventType(eventType)

	instanceName := stringField(payload, "instance")
	if instanceName == "" {
		instanceName = stringField(payload, "instanceName")
	}

	created, err := s.cfg.EvolutionEvents.Create(r.Context(), repository.CreateEvolutionEventParams{
		InstanceName: instanceName,
		EventType:    eventType,
		Payload:      payload,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to persist evolution event"})
		return
	}

	if s.cfg.EvolutionSync != nil {
		if err := s.cfg.EvolutionSync.Sync(r.Context(), created); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to sync evolution event"})
			return
		}
	}

	writeJSON(w, http.StatusAccepted, evolutionWebhookResponse{
		ID:           created.ID,
		EventType:    created.EventType,
		InstanceName: created.InstanceName,
	})
}

func stringField(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return text
}

func webhookEventTypeFromPath(path string) string {
	const prefix = "/api/v1/webhooks/evolution/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	suffix := strings.TrimPrefix(path, prefix)
	suffix = strings.Trim(suffix, "/")
	return suffix
}

func normalizeEvolutionEventType(value string) string {
	normalized := strings.TrimSpace(value)
	normalized = strings.ReplaceAll(normalized, ".", "_")
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ToUpper(normalized)
	return normalized
}
