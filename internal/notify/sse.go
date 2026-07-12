package notify

import (
	"context"
	"sync"
)

const hubChannelBuffer = 64

// Hub manages per-run SSE event channels.
type Hub struct {
	mu     sync.Mutex
	subs   map[string]chan StepEvent
	replay map[string][]StepEvent
}

// NewHub creates an empty Hub.
func NewHub() *Hub {
	return &Hub{
		subs:   make(map[string]chan StepEvent),
		replay: make(map[string][]StepEvent),
	}
}

// Subscribe returns a channel that receives events for the given run ID.
// Any events already buffered in the replay log are sent immediately so that
// late subscribers (connecting after the run has already started) don't miss them.
func (h *Hub) Subscribe(runID string) <-chan StepEvent {
	ch := make(chan StepEvent, hubChannelBuffer)
	h.mu.Lock()
	for _, ev := range h.replay[runID] {
		select {
		case ch <- ev:
		default:
		}
	}
	h.subs[runID] = ch
	h.mu.Unlock()
	return ch
}

// Unsubscribe closes and removes the channel for the given run ID and clears
// the replay buffer so memory is freed once the run is done.
func (h *Hub) Unsubscribe(runID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if ch, ok := h.subs[runID]; ok {
		close(ch)
		delete(h.subs, runID)
	}
	delete(h.replay, runID)
}

func (h *Hub) send(runID string, event StepEvent) {
	h.mu.Lock()
	h.replay[runID] = append(h.replay[runID], event)
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
