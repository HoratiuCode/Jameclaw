package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sipeed/jameclaw/pkg/config"
)

func TestNewAgentInstanceMergesFrontmatterSkills(t *testing.T) {
	workspace := t.TempDir()
	agentPath := filepath.Join(workspace, "AGENT.md")
	if err := os.WriteFile(agentPath, []byte(`---
name: jame
skills:
  - search
---
# Agent
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: workspace,
			},
		},
	}
	instance := NewAgentInstance(
		&config.AgentConfig{
			ID:     "main",
			Skills: []string{"review"},
		},
		&cfg.Agents.Defaults,
		cfg,
		nil,
	)

	if len(instance.SkillsFilter) != 2 {
		t.Fatalf("expected merged skills, got %v", instance.SkillsFilter)
	}
	if instance.SkillsFilter[0] != "review" || instance.SkillsFilter[1] != "search" {
		t.Fatalf("unexpected skills order: %v", instance.SkillsFilter)
	}
}
