package api

import (
	"testing"

	"github.com/sipeed/jameclaw/pkg/providers"
)

func TestRecentConversationRecoveryIssuesDetectsCorrectiveFeedback(t *testing.T) {
	issues := recentConversationRecoveryIssues([]sessionFile{{
		Key: "agent:main:jame:direct:jame:session-1",
		Messages: []providers.Message{
			{Role: "user", Content: "Build a dashboard"},
			{Role: "assistant", Content: "Here is a chart."},
			{Role: "user", Content: "That is wrong, please fix this."},
		},
	}}, "2026-07-20T12:00:00Z")
	if len(issues) != 1 {
		t.Fatalf("issues = %d, want 1", len(issues))
	}
	if issues[0].RecoveryPrompt == "" || issues[0].Status != "needs_follow_up" {
		t.Fatalf("unexpected recovery issue: %#v", issues[0])
	}
}

func TestRecentConversationRecoveryIssuesIgnoresOrdinaryQuestions(t *testing.T) {
	issues := recentConversationRecoveryIssues([]sessionFile{{
		Key: "agent:main:jame:direct:jame:session-1",
		Messages: []providers.Message{
			{Role: "assistant", Content: "The deployment is complete."},
			{Role: "user", Content: "What should I do next?"},
		},
	}}, "2026-07-20T12:00:00Z")
	if len(issues) != 0 {
		t.Fatalf("issues = %#v, want none", issues)
	}
}
