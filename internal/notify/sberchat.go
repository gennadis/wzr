package notify

import "context"

// SberChatNotifier posts events to the SberChat REST API.
// TODO(team): fill in BaseURL, ChatID, Token and implement Notify.
// Expected POST payload: {"chat_id": ChatID, "text": formatEvent(event)}
// Auth: Bearer token in Authorization header.
type SberChatNotifier struct {
	BaseURL string
	ChatID  string
	Token   string
}

// Notify sends event as a human-readable message to SberChat.
func (s *SberChatNotifier) Notify(_ context.Context, _ StepEvent) error {
	// TODO(team): POST to s.BaseURL/messages with event formatted as human-readable message
	return nil
}
