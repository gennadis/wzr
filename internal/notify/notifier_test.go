package notify

import (
	"context"
	"errors"
	"testing"
)

// stubNotifier records calls and optionally returns an error.
type stubNotifier struct {
	called bool
	err    error
}

func (s *stubNotifier) Notify(_ context.Context, _ StepEvent) error {
	s.called = true
	return s.err
}

func TestMultiNotifier_CallsAll(t *testing.T) {
	a := &stubNotifier{}
	b := &stubNotifier{}
	m := NewMultiNotifier(a, b)

	if err := m.Notify(context.Background(), StepEvent{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !a.called || !b.called {
		t.Error("not all notifiers were called")
	}
}

func TestMultiNotifier_ContinuesOnFailure(t *testing.T) {
	boom := errors.New("boom")
	a := &stubNotifier{err: boom}
	b := &stubNotifier{}
	m := NewMultiNotifier(a, b)

	err := m.Notify(context.Background(), StepEvent{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !b.called {
		t.Error("second notifier was not called after first failed")
	}
	if !errors.Is(err, boom) {
		t.Errorf("expected boom in error, got %v", err)
	}
}

func TestHub_SubscribeAndBroadcast(t *testing.T) {
	hub := NewHub()
	ch := hub.Subscribe("run1")

	event := StepEvent{RunID: "run1", Status: StatusRunning, StepID: "s1"}
	sse := NewSSENotifier(hub)
	if err := sse.Notify(context.Background(), event); err != nil {
		t.Fatalf("Notify: %v", err)
	}

	select {
	case got := <-ch:
		if got.StepID != "s1" {
			t.Errorf("StepID: got %q", got.StepID)
		}
	default:
		t.Error("no event received")
	}
}

func TestHub_Unsubscribe(t *testing.T) {
	hub := NewHub()
	hub.Subscribe("run1")
	hub.Unsubscribe("run1")

	// send after unsubscribe should not panic
	sse := NewSSENotifier(hub)
	if err := sse.Notify(context.Background(), StepEvent{RunID: "run1"}); err != nil {
		t.Fatalf("Notify after unsubscribe: %v", err)
	}
}

func TestHub_UnsubscribeClosesChannel(t *testing.T) {
	hub := NewHub()
	ch := hub.Subscribe("run1")
	hub.Unsubscribe("run1")

	_, open := <-ch
	if open {
		t.Error("channel should be closed after Unsubscribe")
	}
}

func TestSberChatNotifier_ReturnsNil(t *testing.T) {
	n := &SberChatNotifier{BaseURL: "http://example.com", ChatID: "c1", Token: "tok"}
	if err := n.Notify(context.Background(), StepEvent{}); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}
