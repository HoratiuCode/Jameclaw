package gateway

import "strings"

// SessionSource describes where an inbound turn came from.
//
// It is intentionally platform-neutral so channel adapters can attach richer
// routing and authorization context without leaking platform-specific structs
// into the agent loop.
type SessionSource struct {
	Platform         string `json:"platform"`
	ChatID           string `json:"chat_id"`
	ChatName         string `json:"chat_name,omitempty"`
	ChatType         string `json:"chat_type,omitempty"`
	ThreadID         string `json:"thread_id,omitempty"`
	UserID           string `json:"user_id,omitempty"`
	UserName         string `json:"user_name,omitempty"`
	Profile          string `json:"profile,omitempty"`
	RoleAuthorized   bool   `json:"role_authorized,omitempty"`
	TrustedRelay     bool   `json:"trusted_relay,omitempty"`
	TriggerMessageID string `json:"trigger_message_id,omitempty"`
}

func (s SessionSource) SessionKey(agentID string) string {
	agentID = cleanSessionSegment(agentID, "main")
	platform := cleanSessionSegment(s.Platform, "unknown")
	chatType := cleanSessionSegment(s.ChatType, "direct")
	chatID := cleanSessionSegment(s.ChatID, "unknown")
	if s.ThreadID != "" {
		return "agent:" + agentID + ":" + platform + ":" + chatType + ":" + chatID + ":thread:" + cleanSessionSegment(s.ThreadID, "unknown")
	}
	return "agent:" + agentID + ":" + platform + ":" + chatType + ":" + chatID
}

func cleanSessionSegment(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	replacer := strings.NewReplacer(":", "_", "/", "_", "\\", "_", "\n", " ", "\r", " ")
	return replacer.Replace(value)
}
