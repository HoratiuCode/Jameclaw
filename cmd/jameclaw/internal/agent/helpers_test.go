package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sipeed/jameclaw/cmd/jameclaw/internal"
	"github.com/sipeed/jameclaw/pkg/config"
)

func TestResolveAgentEmojiUsesWorkspaceSignature(t *testing.T) {
	workspace := t.TempDir()
	agentPath := filepath.Join(workspace, "AGENT.md")
	if err := os.WriteFile(agentPath, []byte("You are Jame, the default assistant for this workspace.\nYour name is JameClaw 🤖.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", agentPath, err)
	}

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: workspace,
			},
		},
	}

	if got := resolveAgentEmoji(cfg); got != "🤖" {
		t.Fatalf("resolveAgentEmoji() = %q, want %q", got, "🤖")
	}
}

func TestResolveAgentEmojiFallsBackToLogoWithoutWorkspace(t *testing.T) {
	if got := resolveAgentEmoji(&config.Config{}); got != internal.Logo {
		t.Fatalf("resolveAgentEmoji() = %q, want %q", got, internal.Logo)
	}
}

func TestResolveTerminalSessionKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv(config.EnvHome, home)

	if got := resolveTerminalSessionKey(" explicit "); got != "explicit" {
		t.Fatalf("explicit session = %q, want explicit", got)
	}
	if got := resolveTerminalSessionKey(""); got != "cli:default" {
		t.Fatalf("empty session = %q, want cli:default", got)
	}

	writeLastTerminalSession(" remembered ")
	if got := resolveTerminalSessionKey(""); got != "remembered" {
		t.Fatalf("remembered session = %q, want remembered", got)
	}
}

func TestResolveCtrlCAction(t *testing.T) {
	now := time.Now()
	if got := resolveCtrlCAction("draft", time.Time{}, now); got != "clear" {
		t.Fatalf("draft action = %q, want clear", got)
	}
	if got := resolveCtrlCAction("", time.Time{}, now); got != "warn" {
		t.Fatalf("first empty action = %q, want warn", got)
	}
	if got := resolveCtrlCAction("", now.Add(-500*time.Millisecond), now); got != "exit" {
		t.Fatalf("second empty action = %q, want exit", got)
	}
	if got := resolveCtrlCAction("", now.Add(-2*time.Second), now); got != "warn" {
		t.Fatalf("late empty action = %q, want warn", got)
	}
}
