package tools

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"

	"github.com/sipeed/jameclaw/pkg/config"
)

type macCommandRunner func(ctx context.Context, command string, args ...string) (string, error)

type MacControlTool struct {
	goos string
	cfg  config.MacControlToolsConfig
	run  macCommandRunner
}

func NewMacControlTool(cfg config.MacControlToolsConfig) *MacControlTool {
	return &MacControlTool{
		goos: runtime.GOOS,
		cfg:  cfg,
		run:  runMacCommand,
	}
}

func (t *MacControlTool) Name() string {
	return "mac_control"
}

func (t *MacControlTool) Description() string {
	return "Control macOS desktop actions, especially Google Chrome browser access: open Chrome, open websites like Instagram, search the web, focus browser windows, type text, press keyboard shortcuts, open apps/URLs/files, control Finder, run Shortcuts, take screenshots, and optionally run approved AppleScript. Opening apps, URLs, searches, and paths defaults to background mode so the user's current app stays in front; pass background=false or use activate_app only when the user needs the app brought forward. Use this as the browser fallback when agent-browser or CDP automation is unavailable."
}

func (t *MacControlTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"description": "Desktop action to perform. For websites such as Instagram, use action=open_url with app=Google Chrome and an https URL. For browser searches, use action=search with app=Google Chrome. Prefer background opening unless the user asks to see or control the app.",
				"enum": []string{
					"open_app", "activate_app", "quit_app", "list_apps", "frontmost_app", "front_window_title", "front_window_bounds",
					"open_url", "open_path", "open_finder", "reveal_path", "search",
					"run_shortcut", "screenshot", "type_text", "keyboard_shortcut", "key_code", "mouse_click", "run_applescript",
				},
			},
			"app": map[string]any{
				"type":        "string",
				"description": "macOS application name. Use Google Chrome for browser requests unless the user asks for another browser.",
			},
			"url": map[string]any{
				"type":        "string",
				"description": "HTTP or HTTPS URL to open.",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "Local file/folder path for open_path, open_finder, reveal_path, or screenshot output.",
			},
			"query": map[string]any{
				"type":        "string",
				"description": "Search query to open in the browser.",
			},
			"engine": map[string]any{
				"type":        "string",
				"description": "Search engine for action=search. Defaults to google.",
				"enum":        []string{"google", "duckduckgo", "brave", "bing", "yahoo", "perplexity", "kagi"},
			},
			"shortcut": map[string]any{
				"type":        "string",
				"description": "macOS Shortcut name for action=run_shortcut.",
			},
			"text": map[string]any{
				"type":        "string",
				"description": "Text to type, or raw AppleScript source for action=run_applescript.",
			},
			"keys": map[string]any{
				"type":        "array",
				"description": "Keyboard shortcut keys. Use modifier names command, shift, option, control plus one regular key, e.g. [\"command\", \"l\"].",
				"items":       map[string]any{"type": "string"},
			},
			"key_code": map[string]any{
				"type":        "number",
				"description": "macOS virtual key code for action=key_code.",
			},
			"x": map[string]any{
				"type":        "number",
				"description": "Screen x coordinate for action=mouse_click.",
			},
			"y": map[string]any{
				"type":        "number",
				"description": "Screen y coordinate for action=mouse_click.",
			},
			"background": map[string]any{
				"type":        "boolean",
				"description": "For open_app, open_url, open_path, and search, keep the opened app/browser behind the user's current app. Defaults to true.",
			},
		},
		"required": []string{"action"},
	}
}

func (t *MacControlTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	if t.goos != "darwin" {
		return ErrorResult("mac_control is only available on macOS")
	}
	if t.run == nil {
		t.run = runMacCommand
	}

	action := strings.TrimSpace(getStringArg(args, "action"))
	output, message, err := t.execute(ctx, action, args)
	if err != nil {
		return ErrorResult(err.Error()).WithError(err)
	}
	if strings.TrimSpace(output) != "" {
		return NewToolResult(message + "\n\n" + strings.TrimSpace(output))
	}
	return NewToolResult(message)
}

func (t *MacControlTool) execute(ctx context.Context, action string, args map[string]any) (string, string, error) {
	app := cleanMacControlValue(getStringArg(args, "app"))
	path := cleanMacControlValue(getStringArg(args, "path"))
	background := getOptionalBoolArg(args, "background", true)

	switch action {
	case "open_app":
		if app == "" {
			return "", "", fmt.Errorf("app is required for action=open_app")
		}
		return t.runWithMessage(ctx, "open", "Opened app "+app, openArgs(background, "-a", app)...)
	case "activate_app":
		if app == "" {
			return "", "", fmt.Errorf("app is required for action=activate_app")
		}
		return t.runAppleScript(ctx, fmt.Sprintf(`tell application %s to activate`, appleScriptStringLiteral(app)), "Activated app "+app)
	case "quit_app":
		if app == "" {
			return "", "", fmt.Errorf("app is required for action=quit_app")
		}
		return t.runAppleScript(ctx, fmt.Sprintf(`tell application %s to quit`, appleScriptStringLiteral(app)), "Quit app "+app)
	case "list_apps":
		return t.runAppleScript(ctx, `tell application "System Events" to get name of application processes whose background only is false`, "Listed running visible apps")
	case "frontmost_app":
		return t.runAppleScript(ctx, `tell application "System Events" to get name of first application process whose frontmost is true`, "Read frontmost app")
	case "front_window_title":
		return t.runAppleScript(ctx, `tell application "System Events"
	set frontApp to first application process whose frontmost is true
	if exists window 1 of frontApp then
		return name of window 1 of frontApp
	else
		return name of frontApp
	end if
end tell`, "Read front window title")
	case "front_window_bounds":
		return t.runAppleScript(ctx, `tell application "System Events"
	set frontApp to first application process whose frontmost is true
	if exists window 1 of frontApp then
		set windowPosition to position of window 1 of frontApp
		set windowSize to size of window 1 of frontApp
		return (item 1 of windowPosition as text) & "," & (item 2 of windowPosition as text) & "," & (item 1 of windowSize as text) & "," & (item 2 of windowSize as text)
	else
		return ""
	end if
end tell`, "Read front window bounds")
	case "open_url":
		rawURL := cleanMacControlValue(getStringArg(args, "url"))
		if err := validateBrowserURL(rawURL); err != nil {
			return "", "", err
		}
		return t.runWithMessage(ctx, "open", "Opened "+rawURL, openTargetArgs(background, app, rawURL)...)
	case "open_path":
		if path == "" {
			return "", "", fmt.Errorf("path is required for action=open_path")
		}
		return t.runWithMessage(ctx, "open", "Opened "+path, openTargetArgs(background, app, path)...)
	case "open_finder":
		if path == "" {
			path = "."
		}
		return t.runWithMessage(ctx, "open", "Opened Finder at "+path, path)
	case "reveal_path":
		if path == "" {
			return "", "", fmt.Errorf("path is required for action=reveal_path")
		}
		return t.runWithMessage(ctx, "open", "Revealed "+path, "-R", path)
	case "search":
		query := strings.TrimSpace(getStringArg(args, "query"))
		if query == "" {
			return "", "", fmt.Errorf("query is required for action=search")
		}
		searchURL, err := searchEngineURL(cleanMacControlValue(getStringArg(args, "engine")), query)
		if err != nil {
			return "", "", err
		}
		return t.runWithMessage(ctx, "open", "Opened search results for "+query, openTargetArgs(background, app, searchURL)...)
	case "run_shortcut":
		if !t.cfg.AllowShortcuts {
			return "", "", fmt.Errorf("run_shortcut is disabled by tools.mac_control.allow_shortcuts")
		}
		shortcut := cleanMacControlValue(getStringArg(args, "shortcut"))
		if shortcut == "" {
			return "", "", fmt.Errorf("shortcut is required for action=run_shortcut")
		}
		return t.runWithMessage(ctx, "shortcuts", "Ran Shortcut "+shortcut, "run", shortcut)
	case "screenshot":
		if !t.cfg.AllowScreenshots {
			return "", "", fmt.Errorf("screenshot is disabled by tools.mac_control.allow_screenshots")
		}
		if path == "" {
			return "", "", fmt.Errorf("path is required for action=screenshot")
		}
		return t.runWithMessage(ctx, "screencapture", "Saved screenshot to "+path, "-x", path)
	case "type_text":
		if !t.cfg.AllowUIAutomation || !t.cfg.AllowTyping {
			return "", "", fmt.Errorf("type_text is disabled by tools.mac_control UI automation or typing settings")
		}
		text := getStringArg(args, "text")
		if text == "" {
			return "", "", fmt.Errorf("text is required for action=type_text")
		}
		return t.runAppleScript(ctx, fmt.Sprintf(`tell application "System Events" to keystroke %s`, appleScriptStringLiteral(text)), "Typed text")
	case "keyboard_shortcut":
		if !t.cfg.AllowUIAutomation {
			return "", "", fmt.Errorf("keyboard_shortcut is disabled by tools.mac_control.allow_ui_automation")
		}
		script, label, err := keyboardShortcutScript(args)
		if err != nil {
			return "", "", err
		}
		return t.runAppleScript(ctx, script, "Pressed "+label)
	case "key_code":
		if !t.cfg.AllowUIAutomation {
			return "", "", fmt.Errorf("key_code is disabled by tools.mac_control.allow_ui_automation")
		}
		code, ok := getNumberArg(args, "key_code")
		if !ok {
			return "", "", fmt.Errorf("key_code is required for action=key_code")
		}
		return t.runAppleScript(ctx, fmt.Sprintf(`tell application "System Events" to key code %d`, int(code)), fmt.Sprintf("Pressed key code %d", int(code)))
	case "mouse_click":
		if !t.cfg.AllowUIAutomation {
			return "", "", fmt.Errorf("mouse_click is disabled by tools.mac_control.allow_ui_automation")
		}
		x, ok := getNumberArg(args, "x")
		if !ok {
			return "", "", fmt.Errorf("x is required for action=mouse_click")
		}
		y, ok := getNumberArg(args, "y")
		if !ok {
			return "", "", fmt.Errorf("y is required for action=mouse_click")
		}
		return t.runAppleScript(ctx, fmt.Sprintf(`tell application "System Events" to click at {%d, %d}`, int(x), int(y)), fmt.Sprintf("Clicked at %d,%d", int(x), int(y)))
	case "run_applescript":
		if !t.cfg.AllowAppleScript {
			return "", "", fmt.Errorf("run_applescript is disabled by tools.mac_control.allow_applescript")
		}
		script := strings.TrimSpace(getStringArg(args, "text"))
		if script == "" {
			return "", "", fmt.Errorf("text AppleScript source is required for action=run_applescript")
		}
		return t.runAppleScript(ctx, script, "Ran AppleScript")
	default:
		return "", "", fmt.Errorf("unsupported mac_control action %q", action)
	}
}

func (t *MacControlTool) runAppleScript(ctx context.Context, script, message string) (string, string, error) {
	output, err := t.run(ctx, "osascript", "-e", script)
	return output, message, err
}

func (t *MacControlTool) runWithMessage(ctx context.Context, command, message string, args ...string) (string, string, error) {
	output, err := t.run(ctx, command, args...)
	return output, message, err
}

func runMacCommand(ctx context.Context, command string, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, command, args...).CombinedOutput()
	return string(out), err
}

func openTargetArgs(background bool, app string, target string) []string {
	args := openArgs(background)
	if app != "" {
		args = append(args, "-a", app)
	}
	return append(args, target)
}

func openArgs(background bool, args ...string) []string {
	if !background {
		return append([]string{}, args...)
	}
	return append([]string{"-g"}, args...)
}

func appendOpenApp(args []string, app string) []string {
	if app == "" {
		return args
	}
	return append([]string{"-a", app}, args...)
}

func cleanMacControlValue(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\x00", "")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	return value
}

func validateBrowserURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("url is required for action=open_url")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("url must be a valid HTTP or HTTPS URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("url scheme must be http or https")
	}
	return nil
}

func searchEngineURL(engine, query string) (string, error) {
	escaped := url.QueryEscape(query)
	switch strings.ToLower(strings.TrimSpace(engine)) {
	case "", "google":
		return "https://www.google.com/search?q=" + escaped, nil
	case "duckduckgo":
		return "https://duckduckgo.com/?q=" + escaped, nil
	case "brave":
		return "https://search.brave.com/search?q=" + escaped, nil
	case "bing":
		return "https://www.bing.com/search?q=" + escaped, nil
	case "yahoo":
		return "https://search.yahoo.com/search?p=" + escaped, nil
	case "perplexity":
		return "https://www.perplexity.ai/search?q=" + escaped, nil
	case "kagi":
		return "https://kagi.com/search?q=" + escaped, nil
	default:
		return "", fmt.Errorf("unsupported search engine %q", engine)
	}
}

func keyboardShortcutScript(args map[string]any) (string, string, error) {
	keys, ok := args["keys"].([]any)
	if !ok || len(keys) == 0 {
		return "", "", fmt.Errorf("keys array is required for action=keyboard_shortcut")
	}

	modifiers := make([]string, 0, 3)
	key := ""
	labels := make([]string, 0, len(keys))
	for _, raw := range keys {
		part, ok := raw.(string)
		if !ok {
			return "", "", fmt.Errorf("keys must contain strings")
		}
		part = strings.ToLower(cleanMacControlValue(part))
		labels = append(labels, part)
		switch part {
		case "command", "cmd":
			modifiers = append(modifiers, "command down")
		case "shift":
			modifiers = append(modifiers, "shift down")
		case "option", "alt":
			modifiers = append(modifiers, "option down")
		case "control", "ctrl":
			modifiers = append(modifiers, "control down")
		default:
			if key != "" {
				return "", "", fmt.Errorf("keyboard_shortcut supports one non-modifier key")
			}
			key = part
		}
	}
	if key == "" {
		return "", "", fmt.Errorf("keyboard_shortcut requires one non-modifier key")
	}

	script := fmt.Sprintf(`tell application "System Events" to keystroke %s`, appleScriptStringLiteral(key))
	if len(modifiers) > 0 {
		script += " using {" + strings.Join(modifiers, ", ") + "}"
	}
	return script, strings.Join(labels, "+"), nil
}

func appleScriptStringLiteral(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}

func getStringArg(args map[string]any, key string) string {
	value, _ := args[key].(string)
	return value
}

func getNumberArg(args map[string]any, key string) (float64, bool) {
	switch value := args[key].(type) {
	case float64:
		return value, true
	case int:
		return float64(value), true
	default:
		return 0, false
	}
}

func getOptionalBoolArg(args map[string]any, key string, fallback bool) bool {
	value, ok := args[key]
	if !ok {
		return fallback
	}
	boolValue, ok := value.(bool)
	if !ok {
		return fallback
	}
	return boolValue
}
