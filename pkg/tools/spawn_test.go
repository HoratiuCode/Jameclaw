package tools

import (
	"context"
	"strings"
	"sync"
	"testing"
)

// mockSpawner implements SubTurnSpawner for testing
type mockSpawner struct{}

func (m *mockSpawner) SpawnSubTurn(ctx context.Context, cfg SubTurnConfig) (*ToolResult, error) {
	// Extract task from system prompt for response
	task := cfg.SystemPrompt
	if strings.Contains(task, "Task: ") {
		parts := strings.Split(task, "Task: ")
		if len(parts) > 1 {
			task = parts[1]
		}
	}
	return &ToolResult{
		ForLLM:  "Task completed: " + task,
		ForUser: "Task completed",
	}, nil
}

func TestSpawnTool_Execute_EmptyTask(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test")
	tool := NewSpawnTool(manager)

	ctx := context.Background()

	tests := []struct {
		name string
		args map[string]any
	}{
		{"empty string", map[string]any{"task": ""}},
		{"whitespace only", map[string]any{"task": "   "}},
		{"tabs and newlines", map[string]any{"task": "\t\n  "}},
		{"missing task key", map[string]any{"label": "test"}},
		{"wrong type", map[string]any{"task": 123}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.Execute(ctx, tt.args)
			if result == nil {
				t.Fatal("Result should not be nil")
			}
			if !result.IsError {
				t.Error("Expected error for invalid task parameter")
			}
			if !strings.Contains(result.ForLLM, "task is required") {
				t.Errorf("Error message should mention 'task is required', got: %s", result.ForLLM)
			}
		})
	}
}

func TestSpawnTool_Execute_ValidTask(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", "/tmp/test")
	tool := NewSpawnTool(manager)
	tool.SetSpawner(&mockSpawner{})

	ctx := context.Background()
	args := map[string]any{
		"task":  "Write a haiku about coding",
		"label": "haiku-task",
	}

	result := tool.Execute(ctx, args)
	if result == nil {
		t.Fatal("Result should not be nil")
	}
	if result.IsError {
		t.Errorf("Expected success for valid task, got error: %s", result.ForLLM)
	}
	if !result.Async {
		t.Error("SpawnTool should return async result")
	}
}

func TestSubagentManager_SpawnTracksOpenClawStyleLifecycle(t *testing.T) {
	provider := &MockLLMProvider{}
	manager := NewSubagentManager(provider, "test-model", t.TempDir())
	var wg sync.WaitGroup
	wg.Add(1)
	manager.SetSpawner(func(
		ctx context.Context,
		task, label, agentID string,
		tools *ToolRegistry,
		maxTokens int,
		temperature float64,
		hasMaxTokens, hasTemperature bool,
	) (*ToolResult, error) {
		defer wg.Done()
		return NewToolResult("done: " + task), nil
	})

	msg, err := manager.Spawn(context.Background(), "check status", "status-check", "", "telegram", "chat-1", nil)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}
	if !strings.Contains(msg, "subagent-1") {
		t.Fatalf("spawn acknowledgement should include task id, got %q", msg)
	}

	wg.Wait()
	task, ok := manager.GetTaskCopy("subagent-1")
	if !ok {
		t.Fatal("expected task subagent-1")
	}
	if task.Status != SubagentStatusSucceeded {
		t.Fatalf("status = %q, want %q", task.Status, SubagentStatusSucceeded)
	}
	if task.Created == 0 || task.Started == 0 || task.Ended == 0 {
		t.Fatalf("expected created/started/ended timestamps, got %+v", task)
	}
	if task.DeliveryStatus != "completed" {
		t.Fatalf("delivery status = %q, want completed", task.DeliveryStatus)
	}
	if !strings.Contains(task.TerminalSummary, "done: check status") {
		t.Fatalf("terminal summary = %q", task.TerminalSummary)
	}
}

func TestSpawnTool_Execute_NilManager(t *testing.T) {
	tool := NewSpawnTool(nil)

	ctx := context.Background()
	args := map[string]any{"task": "test task"}

	result := tool.Execute(ctx, args)
	if !result.IsError {
		t.Error("Expected error for nil manager")
	}
	if !strings.Contains(result.ForLLM, "Subagent manager not configured") {
		t.Errorf("Error message should mention manager not configured, got: %s", result.ForLLM)
	}
}
