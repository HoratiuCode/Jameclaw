package tools

import (
	"context"
	"reflect"
	"strings"
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
			ToolConfig:          config.ToolConfig{Enabled: true},
			AllowOpenApps:       true,
			AllowUIAutomation:   true,
			AllowTyping:         true,
			AllowShortcuts:      true,
			AllowScreenshots:    true,
			AllowAppleScript:    false,
			AllowMusicPlaylists: true,
		},
		run: func(_ context.Context, command string, args ...string) (string, error) {
			*calls = append(*calls, macCommandCall{command: command, args: append([]string{}, args...)})
			return "", nil
		},
	}
}

func TestMacControlToolBlocksOpeningAppsUntilEnabled(t *testing.T) {
	var calls []macCommandCall
	tool := testMacControlTool(&calls)
	tool.cfg.AllowOpenApps = false

	result := tool.Execute(context.Background(), map[string]any{
		"action": "open_app",
		"app":    "Notes",
	})
	if !result.IsError || !strings.Contains(result.ForLLM, "allow_open_apps") {
		t.Fatalf("result = %#v", result)
	}
	if len(calls) != 0 {
		t.Fatalf("open app ran unexpectedly: %#v", calls)
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
	assertMacCommand(t, calls, "open", []string{"-g", "-a", "Notes"})
}

func TestMacControlToolBlocksRemoteChannelByDefault(t *testing.T) {
	var calls []macCommandCall
	tool := testMacControlTool(&calls)
	result := tool.Execute(WithToolContext(context.Background(), "telegram", "chat-1"), map[string]any{
		"action": "open_app",
		"app":    "Safari",
	})
	if !result.IsError || !strings.Contains(result.ForLLM, "restricted to internal channels") {
		t.Fatalf("result = %#v", result)
	}
	if len(calls) != 0 {
		t.Fatalf("remote action ran unexpectedly: %#v", calls)
	}
}

func TestMacControlToolAllowsExplicitTelegramRemoteAccess(t *testing.T) {
	var calls []macCommandCall
	tool := testMacControlTool(&calls)
	tool.cfg.AllowRemote = true
	tool.cfg.RemoteChannels = []string{"telegram"}
	result := tool.Execute(WithToolContext(context.Background(), "telegram", "chat-1"), map[string]any{
		"action": "open_url",
		"url":    "https://example.com",
	})
	if result.IsError {
		t.Fatalf("Execute() error = %v", result.Err)
	}
	assertMacCommand(t, calls, "open", []string{"-g", "https://example.com"})
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
	assertMacCommand(t, calls, "open", []string{"-g", "-a", "Safari", "https://duckduckgo.com/?q=jameclaw+local+ai"})
}

func TestMacControlToolCanOpenForeground(t *testing.T) {
	var calls []macCommandCall
	tool := testMacControlTool(&calls)

	result := tool.Execute(context.Background(), map[string]any{
		"action":     "open_url",
		"url":        "https://example.com",
		"app":        "Google Chrome",
		"background": false,
	})
	if result.IsError {
		t.Fatalf("Execute() error = %v", result.Err)
	}
	assertMacCommand(t, calls, "open", []string{"-a", "Google Chrome", "https://example.com"})
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

func TestMacControlToolCreateMusicPlaylist(t *testing.T) {
	var calls []macCommandCall
	tool := testMacControlTool(&calls)

	result := tool.Execute(context.Background(), map[string]any{
		"action":        "create_music_playlist",
		"playlist_name": "Road Trip",
	})
	if result.IsError {
		t.Fatalf("Execute() error = %v", result.Err)
	}
	assertMacCommand(t, calls, "osascript", []string{"-e", `tell application "Music"
	if exists user playlist "Road Trip" then
		return "Apple Music playlist already exists: " & "Road Trip"
	end if
	make new user playlist with properties {name:"Road Trip"}
	return "Created Apple Music playlist: " & "Road Trip"
end tell`})
}

func TestMacControlToolBlocksMusicPlaylistsUntilEnabled(t *testing.T) {
	var calls []macCommandCall
	tool := testMacControlTool(&calls)
	tool.cfg.AllowMusicPlaylists = false

	result := tool.Execute(context.Background(), map[string]any{
		"action":        "create_music_playlist",
		"playlist_name": "Road Trip",
	})
	if !result.IsError || !strings.Contains(result.ForLLM, "allow_music_playlists") {
		t.Fatalf("result = %#v", result)
	}
	if len(calls) != 0 {
		t.Fatalf("playlist creation ran unexpectedly: %#v", calls)
	}
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
