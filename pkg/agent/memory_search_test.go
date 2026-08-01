package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/jameclaw/pkg/providers"
)

type memoryFlushTestProvider struct {
	content string
}

func (p *memoryFlushTestProvider) Chat(
	_ context.Context,
	_ []providers.Message,
	_ []providers.ToolDefinition,
	_ string,
	_ map[string]any,
) (*providers.LLMResponse, error) {
	return &providers.LLMResponse{Content: p.content}, nil
}

func (p *memoryFlushTestProvider) GetDefaultModel() string { return "test" }

func TestMemorySearchHybridRanking(t *testing.T) {
	workspace := t.TempDir()
	memoryDir := filepath.Join(workspace, "memory")
	if err := os.MkdirAll(memoryDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := `# Preferences
The user loves writing backend services in Rust.

# Travel
The user booked a train to Vienna for Tuesday.`
	if err := os.WriteFile(filepath.Join(memoryDir, "MEMORY.md"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	store := NewMemoryStore(workspace)
	results := store.Search("Which programming language does the user prefer for backend work?", 3, 2000)
	if len(results) == 0 {
		t.Fatal("expected a relevant memory result")
	}
	if !strings.Contains(results[0].Snippet, "Rust") {
		t.Fatalf("top result does not contain the relevant preference: %q", results[0].Snippet)
	}
	if strings.Contains(results[0].Snippet, "Vienna") {
		t.Fatalf("top result included an unrelated memory section: %q", results[0].Snippet)
	}
}

func TestBuildMessagesInjectsOnlyRelevantMemory(t *testing.T) {
	workspace := setupWorkspace(t, map[string]string{
		"memory/MEMORY.md": "# Preferences\nUser prefers Rust for backend programming.",
	})
	cb := NewContextBuilder(workspace)

	staticPrompt := cb.BuildSystemPromptWithCache()
	if strings.Contains(staticPrompt, "prefers Rust") {
		t.Fatal("static prompt should not contain the full memory file")
	}

	messages := cb.BuildMessages(nil, "", "What language do I like for backend development?", nil, "cli", "direct", "", "")
	if len(messages) == 0 || !strings.Contains(messages[0].Content, "prefers Rust") {
		t.Fatalf("relevant memory was not injected: %q", messages[0].Content)
	}
	if !strings.Contains(messages[0].Content, "# Relevant Memory") {
		t.Fatal("retrieved memory section is missing")
	}
}

func TestBuildMessagesUsesConversationForFollowUpMemoryRetrieval(t *testing.T) {
	workspace := setupWorkspace(t, map[string]string{
		"memory/MEMORY.md": "# Preferences\nUser prefers Rust for backend programming.",
	})
	cb := NewContextBuilder(workspace)
	messages := cb.BuildMessages(
		[]providers.Message{{Role: "user", Content: "What programming language do I prefer for backend work?"}},
		"",
		"Can you remind me?",
		nil, "cli", "direct", "", "",
	)
	if len(messages) == 0 || !strings.Contains(messages[0].Content, "prefers Rust") {
		t.Fatalf("follow-up did not retrieve relevant memory: %q", messages[0].Content)
	}
}

func TestMemoryRecencyBoostFavorsRecentDailyNotes(t *testing.T) {
	now := time.Now()
	recent := memoryRecencyBoost("memory/202607/20260729.md", now.Add(-24*time.Hour), now)
	old := memoryRecencyBoost("memory/202601/20260101.md", now.AddDate(0, -6, 0), now)
	longTerm := memoryRecencyBoost("memory/MEMORY.md", now, now)
	if recent <= old {
		t.Fatalf("recent boost = %f, old boost = %f", recent, old)
	}
	if longTerm != 0 {
		t.Fatalf("long-term memory must not decay, boost = %f", longTerm)
	}
}

func TestMemorySearchDoesNotTreatRecencyAsRelevance(t *testing.T) {
	workspace := setupWorkspace(t, map[string]string{
		"memory/MEMORY.md":          "# Preferences\nUser prefers Rust for backend programming.",
		"memory/202607/20260730.md": "# Daily log\nI am checking a Desktop folder, waiting for files, and preparing a competitor research report.",
	})
	recentPath := filepath.Join(workspace, "memory", "202607", "20260730.md")
	if err := os.Chtimes(recentPath, time.Now(), time.Now()); err != nil {
		t.Fatal(err)
	}

	results := NewMemoryStore(workspace).Search("What backend programming language do I prefer?", 5, 2000)
	if len(results) == 0 || !strings.Contains(results[0].Snippet, "Rust") {
		t.Fatalf("durable relevant memory was not ranked first: %#v", results)
	}
	for _, result := range results {
		if strings.Contains(result.Snippet, "Desktop folder") {
			t.Fatalf("unrelated recent log leaked into recall: %#v", results)
		}
	}
}

func TestAppendTodaySkipsDuplicateMemoryEntry(t *testing.T) {
	store := NewMemoryStore(t.TempDir())
	entry := "## Working knowledge\n\n- The user prefers concise answers."
	if err := store.AppendToday(entry); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendToday(entry); err != nil {
		t.Fatal(err)
	}
	note := store.ReadToday()
	if count := strings.Count(note, "The user prefers concise answers."); count != 1 {
		t.Fatalf("duplicate memory entry count = %d, want 1: %q", count, note)
	}
}

func TestBuildMemoryFlushPrompt(t *testing.T) {
	prompt := buildMemoryFlushPrompt([]providers.Message{
		{Role: "user", Content: "Please remember that I prefer concise answers."},
		{Role: "assistant", Content: "Understood."},
	})
	if !strings.Contains(prompt, "prefer concise answers") {
		t.Fatalf("flush prompt omitted conversation content: %q", prompt)
	}
	if !strings.Contains(prompt, noDurableMemoryToken) {
		t.Fatal("flush prompt must define the no-memory sentinel")
	}
}

func TestMemoryFlushPersistsDailyNote(t *testing.T) {
	workspace := t.TempDir()
	cb := NewContextBuilder(workspace)
	agent := &AgentInstance{
		ID:             "main",
		Model:          "test",
		MaxTokens:      4096,
		Provider:       &memoryFlushTestProvider{content: "- User prefers concise answers."},
		ContextBuilder: cb,
	}

	loop := &AgentLoop{}
	loop.flushMemoryBeforeCompaction(context.Background(), agent, []providers.Message{
		{Role: "user", Content: "Please remember that I prefer concise answers."},
	})

	note := cb.memory.ReadToday()
	if !strings.Contains(note, "User prefers concise answers") {
		t.Fatalf("daily memory was not persisted: %q", note)
	}
}

func TestRememberResearchTurnPersistsWorkingKnowledge(t *testing.T) {
	workspace := t.TempDir()
	cb := NewContextBuilder(workspace)
	loop := &AgentLoop{}

	loop.rememberResearchTurn(&turnState{
		agent:       &AgentInstance{ContextBuilder: cb},
		userMessage: "Research Steve Jobs' product philosophy",
	}, "Steve Jobs emphasized focus: saying no to many good ideas to protect a small number of great ones.")

	note := cb.memory.ReadToday()
	for _, want := range []string{"Working knowledge", "Steve Jobs", "saying no to many good ideas"} {
		if !strings.Contains(note, want) {
			t.Fatalf("research memory missing %q: %q", want, note)
		}
	}
}

func TestRememberResearchTurnSkipsOrdinaryChat(t *testing.T) {
	cb := NewContextBuilder(t.TempDir())
	loop := &AgentLoop{}

	loop.rememberResearchTurn(&turnState{
		agent:       &AgentInstance{ContextBuilder: cb},
		userMessage: "Thanks, that helps.",
	}, "You're welcome.")

	if note := cb.memory.ReadToday(); note != "" {
		t.Fatalf("ordinary chat should not become working knowledge: %q", note)
	}
}

func TestCompactResearchLearningKeepsFinalOutcome(t *testing.T) {
	progress := strings.Repeat("I am checking another source and waiting for the result.\n", 80)
	outcome := "Final finding: the orange navigation system should remain consistent across every desktop screen."
	result := compactResearchLearning(progress+outcome, 220)
	if !strings.Contains(result, outcome) {
		t.Fatalf("final research outcome was lost: %q", result)
	}
	if strings.Count(result, "I am checking") > 3 {
		t.Fatalf("too much process chatter remained in memory: %q", result)
	}
}
