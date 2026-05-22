package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

type instanceEventMessage struct {
	InstanceID       string `json:"instanceId"`
	InstanceName     string `json:"instanceName"`
	Status           string `json:"status"`
	ConnectionStatus string `json:"connectionStatus,omitempty"`
}

type instanceEventBroker struct {
	mu   sync.Mutex
	subs map[chan instanceEventMessage]struct{}
}

func newInstanceEventBroker() *instanceEventBroker {
	return &instanceEventBroker{
		subs: make(map[chan instanceEventMessage]struct{}),
	}
}

func (b *instanceEventBroker) Subscribe(ctx context.Context) <-chan instanceEventMessage {
	ch := make(chan instanceEventMessage, 4)
	b.mu.Lock()
	b.subs[ch] = struct{}{}
	b.mu.Unlock()

	go func() {
		<-ctx.Done()
		b.mu.Lock()
		delete(b.subs, ch)
		b.mu.Unlock()
		close(ch)
	}()

	return ch
}

func (b *instanceEventBroker) Publish(event instanceEventMessage) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.subs {
		select {
		case ch <- event:
		default:
		}
	}
}

func (s *Server) handleInstanceEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	events := s.events.Subscribe(r.Context())
	_, _ = fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			payload, err := json.Marshal(event)
			if err != nil {
				continue
			}
			_, _ = fmt.Fprintf(w, "event: instance\n")
			_, _ = fmt.Fprintf(w, "data: %s\n\n", payload)
			flusher.Flush()
		}
	}
}
