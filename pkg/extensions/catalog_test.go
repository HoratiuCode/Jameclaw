package extensions

import (
	"slices"
	"testing"
)

func TestProviderCatalogCoversFactoryProtocols(t *testing.T) {
	want := []string{
		"openai", "azure", "azure-openai", "bedrock", "litellm", "openrouter",
		"groq", "zhipu", "gemini", "nvidia", "ollama", "moonshot",
		"shengsuanyun", "deepseek", "cerebras", "vivgrid", "volcengine",
		"vllm", "qwen", "qwen-intl", "qwen-international", "dashscope-intl",
		"qwen-us", "dashscope-us", "mistral", "avian", "longcat",
		"modelscope", "novita", "nous", "coding-plan", "alibaba-coding",
		"qwen-coding", "minimax", "anthropic", "anthropic-messages",
		"coding-plan-anthropic", "alibaba-coding-anthropic", "antigravity",
		"claude-cli", "claudecli", "codex-cli", "codexcli",
		"github-copilot", "copilot",
	}

	got := map[string]bool{}
	for _, provider := range ProviderCatalog(nil) {
		for _, protocol := range provider.Protocols {
			got[protocol] = true
		}
	}

	for _, protocol := range want {
		if !got[protocol] {
			t.Fatalf("catalog missing protocol %q", protocol)
		}
	}
}

func TestFindPresetReturnsValidModelConfig(t *testing.T) {
	provider, preset, ok := FindPreset("openrouter", "openrouter-auto")
	if !ok {
		t.Fatal("openrouter preset not found")
	}
	if !slices.Contains(provider.Protocols, "openrouter") {
		t.Fatalf("provider protocols = %v, want openrouter", provider.Protocols)
	}
	modelCfg := preset.ToModelConfig("")
	if err := modelCfg.Validate(); err != nil {
		t.Fatalf("preset config did not validate: %v", err)
	}
}

func TestNousPresetUsesAvailablePortalModel(t *testing.T) {
	_, preset, ok := FindPreset("nous", "nous-model")
	if !ok {
		t.Fatal("nous preset not found")
	}
	if preset.Model != "nous/anthropic/claude-sonnet-4.6" {
		t.Fatalf("nous model = %q, want nous/anthropic/claude-sonnet-4.6", preset.Model)
	}
	if preset.Name != "Claude Sonnet 4.6" {
		t.Fatalf("nous model label = %q, want Claude Sonnet 4.6", preset.Name)
	}
}
