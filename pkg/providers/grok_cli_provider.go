package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const grokCLIPathEnv = "JAMECLAW_GROK_CLI_PATH"

// GrokCliProvider uses the signed-in Grok Build CLI installed on this computer.
// It deliberately does not grant Grok Build its native computer tools: JameClaw
// remains responsible for executing tools after it receives a tool request.
type GrokCliProvider struct {
	command   string
	workspace string
}

// NewGrokCliProvider creates a provider backed by Grok Build.
func NewGrokCliProvider(workspace string) *GrokCliProvider {
	return &GrokCliProvider{command: resolveGrokCLICommand(), workspace: workspace}
}

// GrokCLIAvailable reports whether a Grok Build executable can be found.
func GrokCLIAvailable() bool {
	return isExecutableFile(resolveGrokCLICommand())
}

func resolveGrokCLICommand() string {
	if custom := strings.TrimSpace(os.Getenv(grokCLIPathEnv)); custom != "" && isExecutableFile(custom) {
		return custom
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		candidate := filepath.Join(home, ".local", "bin", "grok")
		if isExecutableFile(candidate) {
			return candidate
		}
	}
	if path, err := exec.LookPath("grok"); err == nil && path != "" {
		return path
	}
	return "grok"
}

// Chat runs Grok Build in its documented one-turn JSON mode.
func (p *GrokCliProvider) Chat(
	ctx context.Context, messages []Message, tools []ToolDefinition, model string, options map[string]any,
) (*LLMResponse, error) {
	if !isExecutableFile(p.command) {
		return nil, fmt.Errorf("grok cli error: Grok Build was not found; install it or set %s", grokCLIPathEnv)
	}

	args := []string{"--single", cliMessagesToPrompt(messages), "--output-format", "json", "--max-turns", "1"}
	if systemPrompt := cliSystemPrompt(messages, tools); systemPrompt != "" {
		args = append(args, "--system-prompt-override", systemPrompt+"\n\nDo not use Grok Build's native tools. Return any requested JameClaw tool call in the specified response format.")
	}
	if model = strings.TrimSpace(model); model != "" && model != "grok-build" && model != "default" {
		args = append(args, "--model", model)
	}

	cmd := exec.CommandContext(ctx, p.command, args...)
	if p.workspace != "" {
		cmd.Dir = p.workspace
	}
	cmd.Env = codexCLIEnv(p.command)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if detail := strings.TrimSpace(stderr.String()); detail != "" {
			return nil, fmt.Errorf("grok cli error: %s", detail)
		}
		return nil, fmt.Errorf("grok cli error: %w", err)
	}
	return parseGrokCLIResponse(stdout.String())
}

func (p *GrokCliProvider) GetDefaultModel() string { return "grok-build" }

func cliMessagesToPrompt(messages []Message) string {
	var parts []string
	for _, message := range messages {
		switch message.Role {
		case "user":
			parts = append(parts, "User: "+message.Content)
		case "assistant":
			parts = append(parts, "Assistant: "+message.Content)
		case "tool":
			parts = append(parts, fmt.Sprintf("[Tool Result for %s]: %s", message.ToolCallID, message.Content))
		}
	}
	if len(parts) == 1 && strings.HasPrefix(parts[0], "User: ") {
		return strings.TrimPrefix(parts[0], "User: ")
	}
	return strings.Join(parts, "\n")
}

func cliSystemPrompt(messages []Message, tools []ToolDefinition) string {
	var parts []string
	for _, message := range messages {
		if message.Role == "system" {
			parts = append(parts, message.Content)
		}
	}
	if len(tools) > 0 {
		parts = append(parts, buildCLIToolsPrompt(tools))
	}
	return strings.Join(parts, "\n\n")
}

func parseGrokCLIResponse(output string) (*LLMResponse, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return nil, fmt.Errorf("grok cli returned an empty response")
	}
	var value any
	if err := json.Unmarshal([]byte(output), &value); err != nil {
		return &LLMResponse{Content: output, FinishReason: "stop"}, nil
	}
	content := grokResponseText(value)
	if content == "" {
		return nil, fmt.Errorf("grok cli response did not contain text")
	}
	toolCalls := extractToolCallsFromText(content)
	finishReason := "stop"
	if len(toolCalls) > 0 {
		finishReason, content = "tool_calls", stripToolCallsFromText(content)
	}
	return &LLMResponse{Content: strings.TrimSpace(content), ToolCalls: toolCalls, FinishReason: finishReason}, nil
}

func grokResponseText(value any) string {
	switch item := value.(type) {
	case string:
		return item
	case map[string]any:
		for _, key := range []string{"result", "content", "text", "response", "output", "message"} {
			if text := grokResponseText(item[key]); text != "" {
				return text
			}
		}
	case []any:
		var parts []string
		for _, child := range item {
			if text := grokResponseText(child); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "\n")
	}
	return ""
}
