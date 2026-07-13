package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestTodoToolWriteMergeAndRead(t *testing.T) {
	tool := NewTodoTool(t.TempDir())
	ctx := WithToolContext(context.Background(), "telegram", "chat-1")

	result := tool.Execute(ctx, map[string]any{
		"todos": []map[string]any{
			{"id": "step-1", "content": "Inspect current implementation", "status": "completed"},
			{"id": "step-2", "content": "Add durable planner", "status": "in_progress"},
			{"id": "step-3", "content": "Run tests", "status": "pending"},
		},
	})
	if result.IsError {
		t.Fatalf("write failed: %s", result.ForLLM)
	}

	result = tool.Execute(ctx, map[string]any{
		"merge": true,
		"todos": []map[string]any{
			{"id": "step-2", "content": "Add durable planner", "status": "completed"},
			{"id": "step-3", "content": "Run tests", "status": "in_progress"},
		},
	})
	if result.IsError {
		t.Fatalf("merge failed: %s", result.ForLLM)
	}

	var payload struct {
		Todos   []TodoItem     `json:"todos"`
		Summary map[string]int `json:"summary"`
	}
	if err := json.Unmarshal([]byte(result.ForLLM), &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(payload.Todos) != 3 {
		t.Fatalf("todos len = %d, want 3", len(payload.Todos))
	}
	if payload.Todos[1].Status != todoStatusCompleted {
		t.Fatalf("step-2 status = %q, want completed", payload.Todos[1].Status)
	}
	if payload.Todos[2].Status != todoStatusInProgress {
		t.Fatalf("step-3 status = %q, want in_progress", payload.Todos[2].Status)
	}
	if payload.Summary["completed"] != 2 || payload.Summary["in_progress"] != 1 {
		t.Fatalf("summary = %#v", payload.Summary)
	}
}

func TestTodoToolScopesPlansByChannelAndChat(t *testing.T) {
	tool := NewTodoTool(t.TempDir())
	ctxA := WithToolContext(context.Background(), "telegram", "chat-a")
	ctxB := WithToolContext(context.Background(), "telegram", "chat-b")

	tool.Execute(ctxA, map[string]any{
		"todos": []map[string]any{
			{"id": "a", "content": "A task", "status": "in_progress"},
		},
	})
	tool.Execute(ctxB, map[string]any{
		"todos": []map[string]any{
			{"id": "b", "content": "B task", "status": "in_progress"},
		},
	})

	contextA := tool.FormatForContext("telegram", "chat-a")
	if !strings.Contains(contextA, "A task") || strings.Contains(contextA, "B task") {
		t.Fatalf("unexpected context for chat-a:\n%s", contextA)
	}
	contextB := tool.FormatForContext("telegram", "chat-b")
	if !strings.Contains(contextB, "B task") || strings.Contains(contextB, "A task") {
		t.Fatalf("unexpected context for chat-b:\n%s", contextB)
	}
}

func TestTodoToolContextOnlyIncludesActiveItems(t *testing.T) {
	tool := NewTodoTool(t.TempDir())
	ctx := WithToolContext(context.Background(), "terminal", "local")

	tool.Execute(ctx, map[string]any{
		"todos": []map[string]any{
			{"id": "done", "content": "Already done", "status": "completed"},
			{"id": "active", "content": "Keep working", "status": "in_progress"},
			{"id": "cancelled", "content": "Do not do", "status": "cancelled"},
		},
	})

	plan := tool.FormatForContext("terminal", "local")
	if !strings.Contains(plan, "Keep working") {
		t.Fatalf("active plan missing active item:\n%s", plan)
	}
	if strings.Contains(plan, "Already done") || strings.Contains(plan, "Do not do") {
		t.Fatalf("active plan should omit inactive items:\n%s", plan)
	}
}
