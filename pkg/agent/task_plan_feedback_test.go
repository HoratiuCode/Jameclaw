package agent

import (
	"strings"
	"testing"
)

func TestHasTaskPlanningSignal(t *testing.T) {
	if !hasTaskPlanningSignal("Please research the competitors and compare their pricing.") {
		t.Fatal("expected research request to need a plan")
	}
	if hasTaskPlanningSignal("hello") {
		t.Fatal("simple greeting should not need a plan")
	}
}

func TestNormalizeTaskPlanLimitsAndFormatsBullets(t *testing.T) {
	plan := normalizeTaskPlan("1. Inspect the project\n2. Implement the change\n3. Run tests", 2)
	if plan != "- Inspect the project\n- Implement the change" {
		t.Fatalf("plan = %q", plan)
	}
	if strings.Contains(plan, "Run tests") {
		t.Fatalf("plan exceeds max steps: %q", plan)
	}
}

func TestTaskPlanPromptRequiresClarificationBeforePlanning(t *testing.T) {
	if !strings.Contains(taskPlanSystemPrompt, "CLARIFY:") || !strings.Contains(taskPlanSystemPrompt, "Do not provide a plan") {
		t.Fatalf("task plan prompt does not require clarification: %s", taskPlanSystemPrompt)
	}
}
