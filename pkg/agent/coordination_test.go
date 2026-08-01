package agent

import (
	"os"
	"path/filepath"
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

func TestActiveTeamOperationsContextShowsAssignmentsAndDependencies(t *testing.T) {
	workspace := t.TempDir()
	stateDir := filepath.Join(workspace, "state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	data := `{
  "goal": {"title":"Ship release","outcome":"Signed app passes QA","lead_agent_id":"main","status":"active"},
  "tasks": [
    {"id":"task-1","title":"Build app","owner_agent_id":"builder","status":"done","depends_on":[],"file_scopes":["build"]},
    {"id":"task-2","title":"Verify app","owner_agent_id":"reviewer","status":"planned","depends_on":["task-1"],"file_scopes":["macos"],"time_budget_minutes":30,"token_budget":4000}
  ]
}`
	if err := os.WriteFile(filepath.Join(stateDir, "team-operations.json"), []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	reviewerContext := activeTeamOperationsContext("reviewer", workspace)
	for _, expected := range []string{"Ship release", "Verify app", "task-1=done", "macos", "30 minutes / 4000 tokens"} {
		if !strings.Contains(reviewerContext, expected) {
			t.Fatalf("reviewer context missing %q:\n%s", expected, reviewerContext)
		}
	}
	if strings.Contains(reviewerContext, "Build app") {
		t.Fatalf("reviewer should not receive another agent's task contract:\n%s", reviewerContext)
	}
	leadContext := activeTeamOperationsContext("main", workspace)
	if !strings.Contains(leadContext, "You are the Team Lead") || !strings.Contains(leadContext, "Build app") {
		t.Fatalf("lead context missing team overview:\n%s", leadContext)
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
