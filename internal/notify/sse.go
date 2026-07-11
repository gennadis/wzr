package notify

import (
	"context"
	"sync"
)

const hubChannelBuffer = 64

// Hub manages per-run SSE event channels.
type Hub struct {
	mu   sync.Mutex
	subs map[string]chan StepEvent
}

// NewHub creates an empty Hub.
func NewHub() *Hub {
	return &Hub{subs: make(map[string]chan StepEvent)}
}

// Subscribe returns a channel that receives events for the given run ID.
func (h *Hub) Subscribe(runID string) <-chan StepEvent {
	ch := make(chan StepEvent, hubChannelBuffer)
	h.mu.Lock()
	h.subs[runID] = ch
	h.mu.Unlock()
	return ch
}

// Unsubscribe closes and removes the channel for the given run ID.
func (h *Hub) Unsubscribe(runID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if ch, ok := h.subs[runID]; ok {
		close(ch)
		delete(h.subs, runID)
	}
}

func (h *Hub) send(runID string, event StepEvent) {
	h.mu.Lock()
	ch, ok := h.subs[runID]
	h.mu.Unlock()
	if !ok {
		return
	}
	select {
	case ch <- event:
	default: // drop if subscriber is not keeping up
	}
}

// SSENotifier implements Notifier by forwarding events to the Hub.
type SSENotifier struct {
	hub *Hub
}

// NewSSENotifier creates an SSENotifier backed by the given Hub.
func NewSSENotifier(hub *Hub) *SSENotifier {
	return &SSENotifier{hub: hub}
}

// Notify sends the event to the hub channel for its run ID.
func (s *SSENotifier) Notify(_ context.Context, event StepEvent) error {
	s.hub.send(event.RunID, event)
	return nil
}
