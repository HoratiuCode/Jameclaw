package tools

import (
	"context"
	"reflect"
	"testing"

	"github.com/sipeed/jameclaw/pkg/config"
)

type macCommandCall struct {
	command string
	args    []string
}

func testMacControlTool(calls *[]macCommandCall) *MacControlTool {
	return &MacControlTool{
		goos: "darwin",
		cfg: config.MacControlToolsConfig{
			ToolConfig:        config.ToolConfig{Enabled: true},
			AllowUIAutomation: true,
			AllowTyping:       true,
			AllowShortcuts:    true,
			AllowScreenshots:  true,
			AllowAppleScript:  false,
		},
		run: func(_ context.Context, command string, args ...string) (string, error) {
			*calls = append(*calls, macCommandCall{command: command, args: append([]string{}, args...)})
			return "", nil
		},
	}
}

func TestMacControlToolOpenApp(t *testing.T) {
	var calls []macCommandCall
	tool := testMacControlTool(&calls)

	result := tool.Execute(context.Background(), map[string]any{
		"action": "open_app",
		"app":    "Notes",
	})
	if result.IsError {
		t.Fatalf("Execute() error = %v", result.Err)
	}
	assertMacCommand(t, calls, "open", []string{"-a", "Notes"})
}

func TestMacControlToolSearchWithBrowserApp(t *testing.T) {
	var calls []macCommandCall
	tool := testMacControlTool(&calls)

	result := tool.Execute(context.Background(), map[string]any{
		"action": "search",
		"query":  "jameclaw local ai",
		"engine": "duckduckgo",
		"app":    "Safari",
	})
	if result.IsError {
		t.Fatalf("Execute() error = %v", result.Err)
	}
	assertMacCommand(t, calls, "open", []string{"-a", "Safari", "https://duckduckgo.com/?q=jameclaw+local+ai"})
}

func TestMacControlToolActivateAppUsesAppleScript(t *testing.T) {
	var calls []macCommandCall
	tool := testMacControlTool(&calls)

	result := tool.Execute(context.Background(), map[string]any{
		"action": "activate_app",
		"app":    "Safari",
	})
	if result.IsError {
		t.Fatalf("Execute() error = %v", result.Err)
	}
	assertMacCommand(t, calls, "osascript", []string{"-e", `tell application "Safari" to activate`})
}

func TestMacControlToolRunShortcut(t *testing.T) {
	var calls []macCommandCall
	tool := testMacControlTool(&calls)

	result := tool.Execute(context.Background(), map[string]any{
		"action":   "run_shortcut",
		"shortcut": "Morning Setup",
	})
	if result.IsError {
		t.Fatalf("Execute() error = %v", result.Err)
	}
	assertMacCommand(t, calls, "shortcuts", []string{"run", "Morning Setup"})
}

func TestMacControlToolKeyboardShortcut(t *testing.T) {
	var calls []macCommandCall
	tool := testMacControlTool(&calls)

	result := tool.Execute(context.Background(), map[string]any{
		"action": "keyboard_shortcut",
		"keys":   []any{"command", "shift", "4"},
	})
	if result.IsError {
		t.Fatalf("Execute() error = %v", result.Err)
	}
	assertMacCommand(t, calls, "osascript", []string{"-e", `tell application "System Events" to keystroke "4" using {command down, shift down}`})
}

func TestMacControlToolRejectsUnsupportedURLScheme(t *testing.T) {
	var calls []macCommandCall
	tool := testMacControlTool(&calls)

	result := tool.Execute(context.Background(), map[string]any{
		"action": "open_url",
		"url":    "file:///etc/passwd",
	})
	if !result.IsError {
		t.Fatal("Execute() succeeded, want error")
	}
	if len(calls) != 0 {
		t.Fatalf("run should not be called, got %#v", calls)
	}
}

func TestMacControlToolRawAppleScriptDisabledByDefault(t *testing.T) {
	var calls []macCommandCall
	tool := testMacControlTool(&calls)

	result := tool.Execute(context.Background(), map[string]any{
		"action": "run_applescript",
		"text":   `display dialog "hi"`,
	})
	if !result.IsError {
		t.Fatal("Execute() succeeded, want error")
	}
	if len(calls) != 0 {
		t.Fatalf("run should not be called, got %#v", calls)
	}
}

func TestMacControlToolRequiresMacOS(t *testing.T) {
	var calls []macCommandCall
	tool := testMacControlTool(&calls)
	tool.goos = "linux"

	result := tool.Execute(context.Background(), map[string]any{
		"action": "open_app",
		"app":    "Notes",
	})
	if !result.IsError {
		t.Fatal("Execute() succeeded, want error")
	}
	if len(calls) != 0 {
		t.Fatalf("run should not be called, got %#v", calls)
	}
}

func assertMacCommand(t *testing.T, calls []macCommandCall, command string, args []string) {
	t.Helper()
	if len(calls) != 1 {
		t.Fatalf("calls = %#v, want exactly one call", calls)
	}
	if calls[0].command != command {
		t.Fatalf("command = %q, want %q", calls[0].command, command)
	}
	if !reflect.DeepEqual(calls[0].args, args) {
		t.Fatalf("args = %#v, want %#v", calls[0].args, args)
	}
}
