package agent

import (
	"strings"
	"testing"
)

func TestWorkspaceClaimsPreventConcurrentWritesToTheSameFile(t *testing.T) {
	workspace := t.TempDir()
	loop := &AgentLoop{}
	first := &turnState{turnID: "turn-one", agentID: "builder", agent: &AgentInstance{Workspace: workspace}}
	second := &turnState{turnID: "turn-two", agentID: "reviewer", agent: &AgentInstance{Workspace: workspace}}
	args := map[string]any{"path": "web/app.tsx"}

	if conflict := loop.claimWorkspaceWrite(first, "write_file", args); conflict != "" {
		t.Fatalf("first writer unexpectedly blocked: %s", conflict)
	}
	if conflict := loop.claimWorkspaceWrite(second, "edit_file", args); !strings.Contains(conflict, "builder") {
		t.Fatalf("expected conflicting writer to be identified, got %q", conflict)
	}

	loop.releaseWorkspaceClaims(first)
	if conflict := loop.claimWorkspaceWrite(second, "edit_file", args); conflict != "" {
		t.Fatalf("writer should proceed after the claim is released: %s", conflict)
	}
}

func TestActiveWorkspaceContextHidesOtherConversationTaskDetails(t *testing.T) {
	workspace := t.TempDir()
	loop := &AgentLoop{}
	other := &turnState{
		turnID: "turn-other", agentID: "researcher", sessionKey: "other-session",
		channel: "telegram", chatID: "private-chat", userMessage: "Sensitive client research",
		agent: &AgentInstance{Workspace: workspace}, phase: TurnPhaseTools,
	}
	loop.registerActiveTurn(other)
	defer loop.clearActiveTurn(other)

	context := loop.activeWorkspaceContext("builder", workspace, "web", "current-chat")
	if !strings.Contains(context, "private task in the shared workspace") {
		t.Fatalf("expected safe coordination context, got %q", context)
	}
	if strings.Contains(context, "Sensitive client research") {
		t.Fatalf("coordination context leaked another conversation's task: %q", context)
	}
}
