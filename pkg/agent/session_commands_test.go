package agent

import (
	"strings"
	"testing"

	"github.com/sipeed/jameclaw/pkg/providers"
)

func TestSessionCommandOperations(t *testing.T) {
	al, _, msgBus, _, cleanup := newTestAgentLoop(t)
	defer cleanup()
	defer msgBus.Close()

	agent := al.GetRegistry().GetDefaultAgent()
	if agent == nil {
		t.Fatal("expected default agent")
	}

	sessionKey := "cli:test"
	history := []providers.Message{
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: "first response"},
		{Role: "user", Content: "second"},
		{Role: "assistant", Content: "second response"},
	}
	agent.Sessions.SetHistory(sessionKey, history)
	agent.Sessions.SetSummary(sessionKey, "existing summary")

	stats, err := al.SessionStats(sessionKey)
	if err != nil {
		t.Fatalf("SessionStats() error = %v", err)
	}
	if stats.MessageCount != len(history) {
		t.Fatalf("MessageCount = %d, want %d", stats.MessageCount, len(history))
	}
	if stats.TokenEstimate <= 0 {
		t.Fatalf("TokenEstimate = %d, want positive", stats.TokenEstimate)
	}
	if stats.Summary != "existing summary" {
		t.Fatalf("Summary = %q", stats.Summary)
	}

	prompt, ok, err := al.LastUserPrompt(sessionKey)
	if err != nil {
		t.Fatalf("LastUserPrompt() error = %v", err)
	}
	if !ok || prompt != "second" {
		t.Fatalf("LastUserPrompt() = %q, %v; want second, true", prompt, ok)
	}

	removed, err := al.UndoLastTurn(sessionKey)
	if err != nil {
		t.Fatalf("UndoLastTurn() error = %v", err)
	}
	if removed != 2 {
		t.Fatalf("removed = %d, want 2", removed)
	}
	if got := agent.Sessions.GetHistory(sessionKey); len(got) != 2 {
		t.Fatalf("history len after undo = %d, want 2", len(got))
	}

	if err := al.ResetSession(sessionKey); err != nil {
		t.Fatalf("ResetSession() error = %v", err)
	}
	if got := agent.Sessions.GetHistory(sessionKey); len(got) != 0 {
		t.Fatalf("history len after reset = %d, want 0", len(got))
	}
	if got := agent.Sessions.GetSummary(sessionKey); got != "" {
		t.Fatalf("summary after reset = %q, want empty", got)
	}
}

func TestCompressSession(t *testing.T) {
	al, _, msgBus, _, cleanup := newTestAgentLoop(t)
	defer cleanup()
	defer msgBus.Close()

	agent := al.GetRegistry().GetDefaultAgent()
	if agent == nil {
		t.Fatal("expected default agent")
	}

	sessionKey := "cli:compress"
	agent.Sessions.SetHistory(sessionKey, []providers.Message{
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: "first response"},
		{Role: "user", Content: "second"},
		{Role: "assistant", Content: "second response"},
		{Role: "user", Content: "third"},
		{Role: "assistant", Content: "third response"},
	})

	dropped, remaining, ok, err := al.CompressSession(sessionKey)
	if err != nil {
		t.Fatalf("CompressSession() error = %v", err)
	}
	if !ok {
		t.Fatal("CompressSession() ok = false, want true")
	}
	if dropped == 0 || remaining == 0 {
		t.Fatalf("dropped=%d remaining=%d, want both positive", dropped, remaining)
	}
	if summary := agent.Sessions.GetSummary(sessionKey); !strings.Contains(summary, "Emergency compression") {
		t.Fatalf("summary = %q, want compression note", summary)
	}
}
