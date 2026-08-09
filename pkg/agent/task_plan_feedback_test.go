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

func TestRequestsVisiblePlan(t *testing.T) {
	for _, request := range []string{
		"Plan the migration before changing anything.",
		"Give me the steps first.",
		"How will you approach this refactor?",
	} {
		if !requestsVisiblePlan(request) {
			t.Fatalf("expected explicit planning request to be recognized: %q", request)
		}
	}
	if requestsVisiblePlan("Fix the login redirect and run the test suite.") {
		t.Fatal("ordinary execution request must stay on the fast lane")
	}
}

func TestNormalizeTaskPlanLimitsAndFormatsBullets(t *testing.T) {
	plan := normalizeTaskPlan("1. Inspect the project\n2. Implement the change\n3. Run tests", 2)
	if plan != "- [ ] Inspect the project\n- [ ] Implement the change" {
		t.Fatalf("plan = %q", plan)
	}
	if strings.Contains(plan, "Run tests") {
		t.Fatalf("plan exceeds max steps: %q", plan)
	}
}

func TestTaskPlanPromptEncouragesIndependentExecution(t *testing.T) {
	if !strings.Contains(taskPlanSystemPrompt, "Proceed independently") || !strings.Contains(taskPlanSystemPrompt, "low-risk, reversible assumption") {
		t.Fatalf("task plan prompt does not encourage independent execution: %s", taskPlanSystemPrompt)
	}
	if !strings.Contains(taskPlanSystemPrompt, "irreversible, external, security-sensitive") {
		t.Fatalf("task plan prompt must retain high-impact clarification boundary: %s", taskPlanSystemPrompt)
	}
	if !strings.Contains(taskPlanSystemPrompt, "Tools: ") || !strings.Contains(taskPlanSystemPrompt, "exact tool names") {
		t.Fatalf("task plan prompt must disclose expected tools: %s", taskPlanSystemPrompt)
	}
}
