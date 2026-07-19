package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
