package agent

import (
	"fmt"
	"strings"
)

// SessionStats is a lightweight snapshot used by terminal slash commands.
type SessionStats struct {
	SessionKey    string
	MessageCount  int
	TokenEstimate int
	ContextWindow int
	Summary       string
}

// SessionStats returns history, summary, and rough token usage for a session.
func (al *AgentLoop) SessionStats(sessionKey string) (SessionStats, error) {
	agent := al.agentForSession(sessionKey)
	if agent == nil {
		return SessionStats{}, fmt.Errorf("no agent available for session %q", sessionKey)
	}

	history := agent.Sessions.GetHistory(sessionKey)
	return SessionStats{
		SessionKey:    sessionKey,
		MessageCount:  len(history),
		TokenEstimate: al.estimateTokens(history),
		ContextWindow: agent.ContextWindow,
		Summary:       agent.Sessions.GetSummary(sessionKey),
	}, nil
}

// ResetSession clears history and summary for a session.
func (al *AgentLoop) ResetSession(sessionKey string) error {
	agent := al.agentForSession(sessionKey)
	if agent == nil {
		return fmt.Errorf("no agent available for session %q", sessionKey)
	}

	agent.Sessions.SetHistory(sessionKey, nil)
	agent.Sessions.SetSummary(sessionKey, "")
	return agent.Sessions.Save(sessionKey)
}

// UndoLastTurn removes the most recent user turn and everything after it.
func (al *AgentLoop) UndoLastTurn(sessionKey string) (int, error) {
	agent := al.agentForSession(sessionKey)
	if agent == nil {
		return 0, fmt.Errorf("no agent available for session %q", sessionKey)
	}

	history := agent.Sessions.GetHistory(sessionKey)
	if len(history) == 0 {
		return 0, nil
	}

	cut := -1
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "user" {
			cut = i
			break
		}
	}
	if cut < 0 {
		return 0, nil
	}

	removed := len(history) - cut
	agent.Sessions.SetHistory(sessionKey, history[:cut])
	if err := agent.Sessions.Save(sessionKey); err != nil {
		return removed, err
	}
	return removed, nil
}

// LastUserPrompt returns the last user message in the session.
func (al *AgentLoop) LastUserPrompt(sessionKey string) (string, bool, error) {
	agent := al.agentForSession(sessionKey)
	if agent == nil {
		return "", false, fmt.Errorf("no agent available for session %q", sessionKey)
	}

	history := agent.Sessions.GetHistory(sessionKey)
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "user" && strings.TrimSpace(history[i].Content) != "" {
			return history[i].Content, true, nil
		}
	}
	return "", false, nil
}

// CompressSession drops older session history at safe turn boundaries and records a summary note.
func (al *AgentLoop) CompressSession(sessionKey string) (dropped, remaining int, ok bool, err error) {
	agent := al.agentForSession(sessionKey)
	if agent == nil {
		return 0, 0, false, fmt.Errorf("no agent available for session %q", sessionKey)
	}

	result, compressed := al.forceCompression(agent, sessionKey)
	if !compressed {
		return 0, len(agent.Sessions.GetHistory(sessionKey)), false, nil
	}
	return result.DroppedMessages, result.RemainingMessages, true, nil
}
