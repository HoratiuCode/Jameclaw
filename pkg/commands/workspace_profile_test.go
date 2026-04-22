package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/jameclaw/pkg/config"
)

func TestCustomizationCommandsPersistWorkspaceChanges(t *testing.T) {
	workspace := t.TempDir()
	agentPath := filepath.Join(workspace, "AGENT.md")
	soulPath := filepath.Join(workspace, "SOUL.md")
	stylePath := filepath.Join(workspace, "STYLE.md")

	if err := os.WriteFile(agentPath, []byte(`---
name: jame
description: Test agent
skills:
  - review
---
You are Jame, the default assistant for this workspace.
Your name is JameClaw 🦐.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(soulPath, []byte(`# Soul

I am JameClaw: calm, helpful, practical, and memory disciplined.

## Personality

- Helpful and friendly
- Concise and to the point
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stylePath, []byte(`# Style

- Tone: concise and warm
- Formatting: short markdown blocks
`), 0o644); err != nil {
		t.Fatal(err)
	}

	activeSkills := []string{"review"}
	rt := &Runtime{
		Config: &config.Config{
			Agents: config.AgentsConfig{
				Defaults: config.AgentDefaults{Workspace: workspace},
			},
		},
		ListSkillNames: func() []string {
			return []string{"review", "search"}
		},
		GetActiveSkills: func() []string {
			return append([]string(nil), activeSkills...)
		},
		SetActiveSkills: func(values []string) error {
			activeSkills = append([]string(nil), values...)
			return nil
		},
	}
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/start",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/start outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !strings.Contains(reply, "🦐") {
		t.Fatalf("/start reply should use workspace emoji, got %q", reply)
	}

	reply = ""
	res = ex.Execute(context.Background(), Request{
		Text: "/emoji 🤖",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/emoji outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !strings.Contains(reply, "Updated signature emoji to 🤖") {
		t.Fatalf("/emoji reply=%q", reply)
	}
	rawAgent, err := os.ReadFile(agentPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rawAgent), "Your name is JameClaw 🤖.") {
		t.Fatalf("AGENT.md did not persist emoji:\n%s", string(rawAgent))
	}

	reply = ""
	res = ex.Execute(context.Background(), Request{
		Text: "/persona Calm, direct, and warm.",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/persona outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !strings.Contains(reply, "Updated personality") {
		t.Fatalf("/persona reply=%q", reply)
	}
	rawSoul, err := os.ReadFile(soulPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rawSoul), "Calm, direct, and warm.") {
		t.Fatalf("SOUL.md did not persist persona:\n%s", string(rawSoul))
	}

	reply = ""
	res = ex.Execute(context.Background(), Request{
		Text: "/style Use concise, friendly, and direct replies.",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/style outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !strings.Contains(reply, "Updated speaking style") {
		t.Fatalf("/style reply=%q", reply)
	}
	rawStyle, err := os.ReadFile(stylePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rawStyle), "Use concise, friendly, and direct replies.") {
		t.Fatalf("STYLE.md did not persist speaking style:\n%s", string(rawStyle))
	}

	reply = ""
	res = ex.Execute(context.Background(), Request{
		Text: "/skills add search",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/skills add outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !strings.Contains(reply, "Added skill \"search\"") {
		t.Fatalf("/skills add reply=%q", reply)
	}
	if len(activeSkills) != 2 || activeSkills[0] != "review" || activeSkills[1] != "search" {
		t.Fatalf("runtime active skills not updated: %v", activeSkills)
	}
	persistedSkills := ReadAgentSkills(workspace)
	if len(persistedSkills) != 2 || persistedSkills[0] != "review" || persistedSkills[1] != "search" {
		t.Fatalf("AGENT.md skills not persisted: %v", persistedSkills)
	}
}
