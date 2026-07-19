package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/jameclaw/pkg/browserbridge"
	"github.com/sipeed/jameclaw/pkg/config"
)

type ChromeExtensionTool struct {
	bridge *browserbridge.Bridge
	cfg    config.MacControlToolsConfig
}

func NewChromeExtensionTool(bridge *browserbridge.Bridge, configs ...config.MacControlToolsConfig) *ChromeExtensionTool {
	if bridge == nil {
		bridge = browserbridge.Default
	}
	var cfg config.MacControlToolsConfig
	if len(configs) > 0 {
		cfg = configs[0]
	}
	return &ChromeExtensionTool{bridge: bridge, cfg: cfg}
}

func (t *ChromeExtensionTool) Name() string { return "chrome_extension" }
func (t *ChromeExtensionTool) Description() string {
	return "Control the active Chrome tab through the installed JameClaw Companion extension. Primary workflow: inspect the page, then navigate, click CSS selectors, type or paste into CSS selectors (including contenteditable message composers), scroll, go back, or reload. This supports composing Instagram and X direct messages in a logged-in browser. Visual fallback when selectors are missing or the page changed: (1) call screenshot with app=Google Chrome, (2) use the returned image to identify the next visible target, (3) use mac_control mouse_click, type_text, or keyboard_shortcut only to navigate or fill a form, then (4) take another Chrome screenshot and verify the result. Never use mac_control mouse or keyboard to submit an external message. Send only through this tool's click action: the extension asks for an in-browser confirmation before any Send, Post, Tweet, or Reply click."
}
func (t *ChromeExtensionTool) Parameters() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{
		"action":   map[string]any{"type": "string", "enum": []string{"inspect", "navigate", "click", "type", "scroll", "go_back", "reload"}},
		"url":      map[string]any{"type": "string", "description": "HTTP(S) URL for navigate."},
		"selector": map[string]any{"type": "string", "description": "CSS selector for click or type."},
		"text":     map[string]any{"type": "string", "description": "Text for type."},
		"x":        map[string]any{"type": "number", "description": "Horizontal scroll delta."},
		"y":        map[string]any{"type": "number", "description": "Vertical scroll delta."},
	}, "required": []string{"action"}}
}
func (t *ChromeExtensionTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	if !t.cfg.AllowOpenApps {
		return ErrorResult("Chrome extension control is disabled; enable Allow opening Mac apps in the Web Console System Configuration")
	}
	if err := (&MacControlTool{cfg: t.cfg}).checkChannelAccess(ctx); err != nil {
		return ErrorResult(err.Error()).WithError(err)
	}
	action := strings.TrimSpace(getStringArg(args, "action"))
	switch action {
	case "inspect", "navigate", "click", "type", "scroll", "go_back", "reload":
	default:
		return ErrorResult("unsupported chrome_extension action")
	}
	if action == "navigate" && !strings.HasPrefix(strings.ToLower(strings.TrimSpace(getStringArg(args, "url"))), "http") {
		return ErrorResult("navigate requires an http or https url")
	}
	if (action == "click" || action == "type") && strings.TrimSpace(getStringArg(args, "selector")) == "" {
		return ErrorResult(action + " requires a CSS selector")
	}
	if action == "type" && getStringArg(args, "text") == "" {
		return ErrorResult("type requires text")
	}
	result, err := t.bridge.Dispatch(ctx, action, args)
	if err != nil {
		return ErrorResult(err.Error()).WithError(err)
	}
	if result.Error != "" {
		err := fmt.Errorf("Chrome extension: %s", result.Error)
		return ErrorResult(err.Error()).WithError(err)
	}
	return NewToolResult(result.Content)
}
