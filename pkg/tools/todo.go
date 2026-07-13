package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const (
	todoStatusPending    = "pending"
	todoStatusInProgress = "in_progress"
	todoStatusCompleted  = "completed"
	todoStatusCancelled  = "cancelled"

	maxTodoItems        = 256
	maxTodoContentChars = 4000
)

var validTodoStatuses = map[string]struct{}{
	todoStatusPending:    {},
	todoStatusInProgress: {},
	todoStatusCompleted:  {},
	todoStatusCancelled:  {},
}

type TodoItem struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Status  string `json:"status"`
}

type todoState struct {
	Scopes map[string][]TodoItem `json:"scopes"`
}

type TodoTool struct {
	path string
	mu   sync.Mutex
}

func NewTodoTool(workspace string) *TodoTool {
	return &TodoTool{
		path: filepath.Join(workspace, "plans", "todos.json"),
	}
}

func (t *TodoTool) Name() string {
	return "todo"
}

func (t *TodoTool) Description() string {
	return "Persistent planning tool for large or long-running tasks. Use it to create a short ordered plan before substantial work, keep exactly one item in_progress while working, update progress after meaningful steps, and read the current plan before resuming older work. Omit todos to read the current plan. Provide todos to replace the plan, or set merge=true to update items by id."
}

func (t *TodoTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"todos": map[string]any{
				"type":        "array",
				"description": "Optional full or partial todo list. Each item has id, content, and status.",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id": map[string]any{
							"type":        "string",
							"description": "Stable short id chosen by the agent, such as step-1.",
						},
						"content": map[string]any{
							"type":        "string",
							"description": "Short actionable step.",
						},
						"status": map[string]any{
							"type":        "string",
							"enum":        []string{todoStatusPending, todoStatusInProgress, todoStatusCompleted, todoStatusCancelled},
							"description": "Current item status.",
						},
					},
					"required": []string{"id", "content", "status"},
				},
			},
			"merge": map[string]any{
				"type":        "boolean",
				"description": "When true, update existing items by id and append new items. When false, replace the entire plan.",
			},
		},
	}
}

func (t *TodoTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	scope := todoScope(ToolChannel(ctx), ToolChatID(ctx))
	state, err := t.load()
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to load todo plan: %v", err)).WithError(err)
	}

	if raw, ok := args["todos"]; ok && raw != nil {
		items, err := parseTodoItems(raw)
		if err != nil {
			return ErrorResult(err.Error()).WithError(err)
		}
		merge, _ := args["merge"].(bool)
		if state.Scopes == nil {
			state.Scopes = make(map[string][]TodoItem)
		}
		if merge {
			state.Scopes[scope] = mergeTodoItems(state.Scopes[scope], items)
		} else {
			state.Scopes[scope] = normalizeTodoItems(items)
		}
		if err := t.save(state); err != nil {
			return ErrorResult(fmt.Sprintf("failed to save todo plan: %v", err)).WithError(err)
		}
	}

	return SilentResult(formatTodoResult(state.Scopes[scope]))
}

func (t *TodoTool) FormatForContext(channel, chatID string) string {
	state, err := t.load()
	if err != nil {
		return ""
	}
	items := activeTodoItems(state.Scopes[todoScope(channel, chatID)])
	if len(items) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("# Active Long-Running Task Plan\n\n")
	sb.WriteString("The following plan was preserved for this channel/chat. Continue from the in_progress item, update the todo tool after meaningful progress, and do not repeat completed work.\n")
	for _, item := range items {
		marker := "[ ]"
		if item.Status == todoStatusInProgress {
			marker = "[>]"
		}
		fmt.Fprintf(&sb, "- %s %s. %s (%s)\n", marker, item.ID, item.Content, item.Status)
	}
	return strings.TrimSpace(sb.String())
}

func (t *TodoTool) load() (todoState, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.loadLocked()
}

func (t *TodoTool) save(state todoState) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if state.Scopes == nil {
		state.Scopes = make(map[string][]TodoItem)
	}
	if err := os.MkdirAll(filepath.Dir(t.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(t.path, data, 0o600)
}

func (t *TodoTool) loadLocked() (todoState, error) {
	state := todoState{Scopes: make(map[string][]TodoItem)}
	data, err := os.ReadFile(t.path)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return state, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return state, nil
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return todoState{}, err
	}
	if state.Scopes == nil {
		state.Scopes = make(map[string][]TodoItem)
	}
	return state, nil
}

func parseTodoItems(raw any) ([]TodoItem, error) {
	if text, ok := raw.(string); ok {
		var decoded []TodoItem
		if err := json.Unmarshal([]byte(text), &decoded); err != nil {
			return nil, fmt.Errorf("todos must be a list of objects, got unparseable string")
		}
		return decoded, nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("todos must be a list of objects")
	}
	var items []TodoItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("todos must be a list of objects")
	}
	return items, nil
}

func mergeTodoItems(existing, updates []TodoItem) []TodoItem {
	out := normalizeTodoItems(existing)
	index := make(map[string]int, len(out))
	for i, item := range out {
		index[item.ID] = i
	}
	for _, update := range normalizeTodoItems(updates) {
		if i, ok := index[update.ID]; ok {
			out[i] = update
			continue
		}
		index[update.ID] = len(out)
		out = append(out, update)
	}
	if len(out) > maxTodoItems {
		out = out[:maxTodoItems]
	}
	return out
}

func normalizeTodoItems(items []TodoItem) []TodoItem {
	out := make([]TodoItem, 0, min(len(items), maxTodoItems))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		item.ID = strings.TrimSpace(item.ID)
		if item.ID == "" {
			item.ID = "?"
		}
		if _, ok := seen[item.ID]; ok {
			for i := range out {
				if out[i].ID == item.ID {
					out[i] = normalizeTodoItem(item)
					break
				}
			}
			continue
		}
		seen[item.ID] = struct{}{}
		out = append(out, normalizeTodoItem(item))
		if len(out) >= maxTodoItems {
			break
		}
	}
	return out
}

func normalizeTodoItem(item TodoItem) TodoItem {
	item.ID = strings.TrimSpace(item.ID)
	if item.ID == "" {
		item.ID = "?"
	}
	item.Content = strings.TrimSpace(item.Content)
	if item.Content == "" {
		item.Content = "(no description)"
	}
	if len(item.Content) > maxTodoContentChars {
		item.Content = item.Content[:maxTodoContentChars-len("... [truncated]")] + "... [truncated]"
	}
	item.Status = strings.TrimSpace(strings.ToLower(item.Status))
	if _, ok := validTodoStatuses[item.Status]; !ok {
		item.Status = todoStatusPending
	}
	return item
}

func activeTodoItems(items []TodoItem) []TodoItem {
	active := make([]TodoItem, 0, len(items))
	for _, item := range normalizeTodoItems(items) {
		if item.Status == todoStatusPending || item.Status == todoStatusInProgress {
			active = append(active, item)
		}
	}
	return active
}

func formatTodoResult(items []TodoItem) string {
	items = normalizeTodoItems(items)
	counts := map[string]int{
		todoStatusPending:    0,
		todoStatusInProgress: 0,
		todoStatusCompleted:  0,
		todoStatusCancelled:  0,
	}
	for _, item := range items {
		counts[item.Status]++
	}
	statuses := make([]string, 0, len(counts))
	for status := range counts {
		statuses = append(statuses, status)
	}
	sort.Strings(statuses)
	summary := make(map[string]int, len(statuses)+1)
	summary["total"] = len(items)
	for _, status := range statuses {
		summary[status] = counts[status]
	}

	data, _ := json.Marshal(map[string]any{
		"todos":   items,
		"summary": summary,
	})
	return string(data)
}

func todoScope(channel, chatID string) string {
	channel = strings.TrimSpace(channel)
	chatID = strings.TrimSpace(chatID)
	if channel == "" && chatID == "" {
		return "local"
	}
	if channel == "" {
		channel = "unknown"
	}
	if chatID == "" {
		chatID = "unknown"
	}
	return channel + ":" + chatID
}
