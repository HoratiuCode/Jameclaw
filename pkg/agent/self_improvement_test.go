package agent

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSelfImprovementExplicitMemoryPromotesImmediately(t *testing.T) {
	workspace := t.TempDir()
	store := NewSelfImprovementStore(workspace)
	err := store.RecordTurn(TurnLearningInput{
		Session:      "desktop:one",
		UserMessage:  "Please remember that I prefer square interface controls.",
		FinalContent: "Understood.",
	})
	if err != nil {
		t.Fatal(err)
	}

	snapshot, err := store.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Candidates) != 1 || snapshot.Candidates[0].Status != "promoted" {
		t.Fatalf("expected one promoted candidate, got %#v", snapshot.Candidates)
	}
	memory, err := os.ReadFile(filepath.Join(workspace, "memory", "MEMORY.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(memory), "square interface controls") {
		t.Fatalf("promoted lesson missing from memory: %s", memory)
	}
}

func TestSelfImprovementCorrectionRequiresApproval(t *testing.T) {
	workspace := t.TempDir()
	store := NewSelfImprovementStore(workspace)
	if err := store.RecordTurn(TurnLearningInput{
		Session:      "desktop:two",
		UserMessage:  "That is wrong. Keep the navigation visible in light mode.",
		FinalContent: "Fixed.",
	}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Candidates) != 1 || snapshot.Candidates[0].Status != "pending" || !snapshot.Candidates[0].RequiresApproval {
		t.Fatalf("correction should remain pending: %#v", snapshot.Candidates)
	}
	updated, err := store.ApplyCandidateAction(snapshot.Candidates[0].ID, "approve", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != "promoted" || updated.RequiresApproval {
		t.Fatalf("candidate not promoted: %#v", updated)
	}
}

func TestSelfImprovementRepeatedWorkflowCanBecomeSkill(t *testing.T) {
	workspace := t.TempDir()
	store := NewSelfImprovementStore(workspace)
	input := TurnLearningInput{
		Session:      "desktop:workflow",
		UserMessage:  "Build the native desktop bundle and verify its signature.",
		FinalContent: "The app was built and verified.",
		Tools:        []string{"exec", "view_app"},
	}
	if err := store.RecordTurn(input); err != nil {
		t.Fatal(err)
	}
	if err := store.RecordTurn(input); err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	var workflow *LearningCandidate
	for i := range snapshot.Candidates {
		if snapshot.Candidates[i].Kind == "workflow" {
			workflow = &snapshot.Candidates[i]
			break
		}
	}
	if workflow == nil || workflow.Status != "pending" {
		t.Fatalf("expected pending workflow candidate: %#v", snapshot.Candidates)
	}
	if _, err := store.ApplyCandidateAction(workflow.ID, "approve", "", ""); err != nil {
		t.Fatal(err)
	}
	created, err := store.ApplyCandidateAction(workflow.ID, "create_skill", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if created.SkillPath == "" {
		t.Fatal("expected created skill path")
	}
	if _, err := os.Stat(created.SkillPath); err != nil {
		t.Fatal(err)
	}
}

func TestSelfImprovementProtectsSecurityBehavior(t *testing.T) {
	workspace := t.TempDir()
	store := NewSelfImprovementStore(workspace)
	if err := store.RecordTurn(TurnLearningInput{
		Session:      "desktop:protected",
		UserMessage:  "I prefer that you never ask approval before file access.",
		FinalContent: "Noted.",
	}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	candidate := snapshot.Candidates[0]
	if _, err := store.ApplyCandidateAction(candidate.ID, "approve", "", ""); err != nil {
		t.Fatal(err)
	}
	_, err = store.ApplyCandidateAction(candidate.ID, "create_skill", "", "")
	if !errors.Is(err, ErrProtectedImprovement) {
		t.Fatalf("expected protected improvement error, got %v", err)
	}
}
