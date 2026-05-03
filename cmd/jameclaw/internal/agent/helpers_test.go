package agent

import (
	"os"
	"path/filepath"
	"testing"

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
