package agent

import (
	"testing"

	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/pkg/providers"
)

func TestProviderForCandidateBuildsTheSecondaryProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	primary := &config.ModelConfig{ModelName: "codex", Model: "codex-cli/gpt-5.4"}
	secondary := &config.ModelConfig{ModelName: "claude", Model: "anthropic/claude-sonnet-4-6"}
	secondary.SetAPIKey("test-key")
	cfg.ModelList = []*config.ModelConfig{primary, secondary}

	primaryProvider := providers.NewCodexCliProvider(t.TempDir())
	agent := &AgentInstance{
		Workspace: t.TempDir(),
		Provider:  primaryProvider,
		Candidates: []providers.FallbackCandidate{
			{Provider: "codex-cli", Model: "gpt-5.4"},
			{Provider: "anthropic", Model: "claude-sonnet-4-6"},
		},
	}

	got, err := providerForCandidate(cfg, agent, agent.Candidates[1])
	if err != nil {
		t.Fatalf("providerForCandidate() error = %v", err)
	}
	if got == primaryProvider {
		t.Fatal("secondary candidate reused the primary provider")
	}
	if _, ok := got.(*providers.HTTPProvider); !ok {
		t.Fatalf("secondary provider = %T, want *providers.HTTPProvider", got)
	}
}
