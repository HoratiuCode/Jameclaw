package onboard

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/jameclaw/cmd/jameclaw/internal"
	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/web/backend/launcherconfig"
)

func TestCopyEmbeddedToTargetUsesStructuredAgentFiles(t *testing.T) {
	targetDir := t.TempDir()

	if err := copyEmbeddedToTarget(targetDir); err != nil {
		t.Fatalf("copyEmbeddedToTarget() error = %v", err)
	}

	agentPath := filepath.Join(targetDir, "AGENT.md")
	if _, err := os.Stat(agentPath); err != nil {
		t.Fatalf("expected %s to exist: %v", agentPath, err)
	}

	soulPath := filepath.Join(targetDir, "SOUL.md")
	if _, err := os.Stat(soulPath); err != nil {
		t.Fatalf("expected %s to exist: %v", soulPath, err)
	}

	userPath := filepath.Join(targetDir, "USER.md")
	if _, err := os.Stat(userPath); err != nil {
		t.Fatalf("expected %s to exist: %v", userPath, err)
	}

	for _, legacyName := range []string{"AGENTS.md", "IDENTITY.md"} {
		legacyPath := filepath.Join(targetDir, legacyName)
		if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
			t.Fatalf("expected legacy file %s to be absent, got err=%v", legacyPath, err)
		}
	}
}

func TestIsCompleteRequiresChatReadyModel(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv(config.EnvHome, homeDir)

	cfg := config.DefaultConfig()
	configPath := internal.GetConfigPath()

	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	if IsComplete() {
		t.Fatal("IsComplete() = true, want false without a default model")
	}

	cfg.Agents.Defaults.ModelName = "llama3"
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() with model error = %v", err)
	}
	if !IsComplete() {
		t.Fatal("IsComplete() = false, want true with a local default model")
	}
}

func TestApplyModelChoiceSetsDefaultModelAndAPIKey(t *testing.T) {
	cfg := config.DefaultConfig()

	err := applyModelChoice(newLineReader("sk-test\n"), cfg, onboardModelOption{
		modelName:      "gpt-5.4",
		requiresAPIKey: true,
		keyLabel:       "OpenAI API key",
	})
	if err != nil {
		t.Fatalf("applyModelChoice() error = %v", err)
	}

	if got := cfg.Agents.Defaults.ModelName; got != "gpt-5.4" {
		t.Fatalf("default model = %q, want %q", got, "gpt-5.4")
	}

	modelCfg := lookupModelConfig(cfg, "gpt-5.4")
	if modelCfg == nil {
		t.Fatal("lookupModelConfig() returned nil")
	}
	if got := modelCfg.APIKey(); got != "sk-test" {
		t.Fatalf("APIKey() = %q, want %q", got, "sk-test")
	}
}

func TestApplyModelChoiceCreatesCatalogModel(t *testing.T) {
	cfg := config.DefaultConfig()

	err := applyModelChoice(newLineReader("nvapi-test\n"), cfg, onboardModelOption{
		providerID:     "nvidia",
		presetID:       "nvidia-llama",
		modelName:      "nvidia-llama",
		requiresAPIKey: true,
		keyLabel:       "NVIDIA API key",
	})
	if err != nil {
		t.Fatalf("applyModelChoice() error = %v", err)
	}

	if got := cfg.Agents.Defaults.ModelName; got != "nvidia-llama" {
		t.Fatalf("default model = %q, want NVIDIA model", got)
	}

	modelCfg := lookupModelConfig(cfg, "nvidia-llama")
	if modelCfg == nil {
		t.Fatal("lookupModelConfig() returned nil for NVIDIA catalog model")
	}
	if got := modelCfg.Model; got != "nvidia/meta/llama-3.1-405b-instruct" {
		t.Fatalf("Model = %q, want NVIDIA provider model", got)
	}
	if got := modelCfg.APIKey(); got != "nvapi-test" {
		t.Fatalf("APIKey() = %q, want %q", got, "nvapi-test")
	}
}

func TestApplyModelChoiceAcceptsCodexCLIWithoutAPIKey(t *testing.T) {
	cfg := config.DefaultConfig()

	err := applyModelChoice(newLineReader(""), cfg, onboardModelOption{
		providerID:     "codex-cli",
		presetID:       "codex-cli",
		modelName:      "codex-cli",
		requiresAPIKey: false,
		keyLabel:       "API key",
	})
	if err != nil {
		t.Fatalf("applyModelChoice() error = %v", err)
	}

	if got := cfg.Agents.Defaults.ModelName; got != "codex-cli" {
		t.Fatalf("default model = %q, want codex-cli", got)
	}
	modelCfg := lookupModelConfig(cfg, "codex-cli")
	if modelCfg == nil {
		t.Fatal("lookupModelConfig() returned nil for Codex CLI model")
	}
	if got := modelCfg.Model; got != "codex-cli/gpt-5.4" {
		t.Fatalf("Model = %q, want Codex CLI provider model", got)
	}
	if !modelReadyForChat(modelCfg) {
		t.Fatal("modelReadyForChat() = false, want true for Codex CLI without API key")
	}
}

func TestModelReadyForChatAcceptsLocalCLIProvidersWithoutAPIKey(t *testing.T) {
	for _, model := range []string{
		"ollama/llama3",
		"vllm/local-model",
		"claude-cli/sonnet",
		"codex-cli/gpt-5.4",
		"github-copilot/default",
		"antigravity/default",
	} {
		if !modelReadyForChat(&config.ModelConfig{ModelName: model, Model: model}) {
			t.Fatalf("modelReadyForChat(%q) = false, want true", model)
		}
	}
}

func TestBuildOnboardModelOptionsIncludesExpandedProviders(t *testing.T) {
	options := buildOnboardModelOptions()
	if len(options) < 30 {
		t.Fatalf("buildOnboardModelOptions() len = %d, want at least 30", len(options))
	}

	providers := make(map[string]onboardModelOption, len(options))
	for _, option := range options {
		providers[option.providerID] = option
	}

	for _, providerID := range []string{
		"moonshot",
		"nvidia",
		"novita",
		"gemini",
		"deepseek",
		"qwen",
		"qwen-intl",
		"qwen-us",
		"vllm",
		"codex-cli",
		"github-copilot",
	} {
		if _, ok := providers[providerID]; !ok {
			t.Fatalf("expected onboard model provider %q to be included", providerID)
		}
	}

	if moonshot := providers["moonshot"]; !strings.Contains(strings.ToLower(moonshot.label), "kimi") {
		t.Fatalf("moonshot label = %q, want it to mention Kimi", moonshot.label)
	}
}

func TestPromptTelegramSetupEnablesTelegramAndAllowFrom(t *testing.T) {
	cfg := config.DefaultConfig()

	err := promptTelegramSetup(newLineReader("y\nbot-token\n123456\n"), cfg)
	if err != nil {
		t.Fatalf("promptTelegramSetup() error = %v", err)
	}

	if !cfg.Channels.Telegram.Enabled {
		t.Fatal("Telegram.Enabled = false, want true")
	}
	if got := cfg.Channels.Telegram.Token(); got != "bot-token" {
		t.Fatalf("Token() = %q, want %q", got, "bot-token")
	}
	if len(cfg.Channels.Telegram.AllowFrom) != 1 || cfg.Channels.Telegram.AllowFrom[0] != "123456" {
		t.Fatalf("AllowFrom = %#v, want [123456]", cfg.Channels.Telegram.AllowFrom)
	}
}

func TestPromptTelegramSetupAcceptsUsernameAllowFrom(t *testing.T) {
	cfg := config.DefaultConfig()

	err := promptTelegramSetup(newLineReader("y\nbot-token\n@alice\n"), cfg)
	if err != nil {
		t.Fatalf("promptTelegramSetup() error = %v", err)
	}

	if len(cfg.Channels.Telegram.AllowFrom) != 1 || cfg.Channels.Telegram.AllowFrom[0] != "@alice" {
		t.Fatalf("AllowFrom = %#v, want [@alice]", cfg.Channels.Telegram.AllowFrom)
	}
}

func TestPromptLauncherAccessSetupSSHMode(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")

	access, err := promptLauncherAccessSetup(newLineReader("3\n19999\n"), configPath)
	if err != nil {
		t.Fatalf("promptLauncherAccessSetup() error = %v", err)
	}

	if access.mode != "Remote/no-GUI via SSH tunnel" {
		t.Fatalf("mode = %q", access.mode)
	}
	if !strings.Contains(access.sshHint, "ssh -N -L 19999:127.0.0.1:19999") {
		t.Fatalf("ssh hint = %q", access.sshHint)
	}

	got, err := launcherconfig.Load(access.configPath, launcherconfig.Default())
	if err != nil {
		t.Fatalf("launcherconfig.Load() error = %v", err)
	}
	if got.Port != 19999 || got.Public {
		t.Fatalf("launcher config = %+v, want port 19999 public false", got)
	}
}

func TestPromptLauncherAccessSetupCustomCIDRs(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")

	access, err := promptLauncherAccessSetup(newLineReader("4\n18888\n100.64.0.0/10, 192.168.50.0/24\n"), configPath)
	if err != nil {
		t.Fatalf("promptLauncherAccessSetup() error = %v", err)
	}
	if access.mode != "Tailnet/custom CIDRs" {
		t.Fatalf("mode = %q", access.mode)
	}

	got, err := launcherconfig.Load(access.configPath, launcherconfig.Default())
	if err != nil {
		t.Fatalf("launcherconfig.Load() error = %v", err)
	}
	wantCIDRs := []string{"100.64.0.0/10", "192.168.50.0/24"}
	if got.Port != 18888 || !got.Public || strings.Join(got.AllowedCIDRs, ",") != strings.Join(wantCIDRs, ",") {
		t.Fatalf("launcher config = %+v, want port 18888 public true cidrs %#v", got, wantCIDRs)
	}
}

func TestReadAgentSignatureEmojiDefaultsWhenMissing(t *testing.T) {
	if got := readAgentSignatureEmoji(t.TempDir()); got != defaultAgentSignatureEmoji {
		t.Fatalf("readAgentSignatureEmoji() = %q, want %q", got, defaultAgentSignatureEmoji)
	}
}

func TestApplyAgentSignatureEmojiUpdatesAgentTemplate(t *testing.T) {
	targetDir := t.TempDir()

	if err := copyEmbeddedToTarget(targetDir); err != nil {
		t.Fatalf("copyEmbeddedToTarget() error = %v", err)
	}
	if err := applyAgentSignatureEmoji(targetDir, "🤖"); err != nil {
		t.Fatalf("applyAgentSignatureEmoji() error = %v", err)
	}

	agentPath := filepath.Join(targetDir, "AGENT.md")
	data, err := os.ReadFile(agentPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", agentPath, err)
	}
	if !strings.Contains(string(data), "Your name is JameClaw 🤖.") {
		t.Fatalf("AGENT.md did not contain updated signature:\n%s", string(data))
	}
}

func TestPromptAgentSignatureEmojiKeepsCurrentWhenBlank(t *testing.T) {
	targetDir := t.TempDir()

	if err := copyEmbeddedToTarget(targetDir); err != nil {
		t.Fatalf("copyEmbeddedToTarget() error = %v", err)
	}
	if err := applyAgentSignatureEmoji(targetDir, "🦀"); err != nil {
		t.Fatalf("applyAgentSignatureEmoji() setup error = %v", err)
	}

	got, err := promptAgentSignatureEmoji(newLineReader("\n"), targetDir, "🦀")
	if err != nil {
		t.Fatalf("promptAgentSignatureEmoji() error = %v", err)
	}
	if got != "🦀" {
		t.Fatalf("promptAgentSignatureEmoji() = %q, want %q", got, "🦀")
	}

	if current := readAgentSignatureEmoji(targetDir); current != "🦀" {
		t.Fatalf("readAgentSignatureEmoji() after prompt = %q, want %q", current, "🦀")
	}
}

func TestPromptAgentSignatureEmojiSupportsComplexEmoji(t *testing.T) {
	targetDir := t.TempDir()

	if err := copyEmbeddedToTarget(targetDir); err != nil {
		t.Fatalf("copyEmbeddedToTarget() error = %v", err)
	}

	got, err := promptAgentSignatureEmoji(newLineReader("🧑‍💻\n"), targetDir, defaultAgentSignatureEmoji)
	if err != nil {
		t.Fatalf("promptAgentSignatureEmoji() error = %v", err)
	}
	if got != "🧑‍💻" {
		t.Fatalf("promptAgentSignatureEmoji() = %q, want %q", got, "🧑‍💻")
	}

	if current := readAgentSignatureEmoji(targetDir); current != "🧑‍💻" {
		t.Fatalf("readAgentSignatureEmoji() after prompt = %q, want %q", current, "🧑‍💻")
	}
}

func TestLoadOnboardSkillOptionsIncludesEmbeddedSkills(t *testing.T) {
	options := loadOnboardSkillOptions()
	if len(options) < 11 {
		t.Fatalf("loadOnboardSkillOptions() len = %d, want at least 11", len(options))
	}

	want := map[string]bool{
		"agent-browser": false,
		"gog":           false,
		"github":        false,
		"hardware":      false,
		"moltbook":      false,
		"security":      false,
		"session-logs":  false,
		"skill-creator": false,
		"summarize":     false,
		"twitter-x":     false,
		"tmux":          false,
		"weather":       false,
	}
	for _, option := range options {
		if _, ok := want[option.name]; ok {
			want[option.name] = true
		}
		if strings.TrimSpace(option.description) == "" {
			t.Fatalf("skill %q description should not be empty", option.name)
		}
	}
	for name, seen := range want {
		if !seen {
			t.Fatalf("expected skill %q to be loaded", name)
		}
	}
}

func TestPromptSkillSelectionDefaultsToPreselectedSkills(t *testing.T) {
	cfg := config.DefaultConfig()

	selected, err := promptSkillSelection(newLineReader("\n"), cfg)
	if err != nil {
		t.Fatalf("promptSkillSelection() error = %v", err)
	}

	want := defaultOnboardSkills(loadOnboardSkillOptions())
	if strings.Join(selected, ",") != strings.Join(want, ",") {
		t.Fatalf("selected = %#v, want %#v", selected, want)
	}
	agent := lookupOnboardAgent(cfg)
	if agent == nil {
		t.Fatal("expected default agent config to be created")
	}
	if strings.Join(agent.Skills, ",") != strings.Join(want, ",") {
		t.Fatalf("agent.Skills = %#v, want %#v", agent.Skills, want)
	}
}

func TestPromptSkillSelectionStoresExplicitSubset(t *testing.T) {
	cfg := config.DefaultConfig()
	options := loadOnboardSkillOptions()
	githubIndex := onboardSkillIndex(t, options, "github")
	skillCreatorIndex := onboardSkillIndex(t, options, "skill-creator")

	selected, err := promptSkillSelection(newLineReader(fmt.Sprintf("%d %d\n", githubIndex, skillCreatorIndex)), cfg)
	if err != nil {
		t.Fatalf("promptSkillSelection() error = %v", err)
	}

	want := []string{"github", "skill-creator"}
	if strings.Join(selected, ",") != strings.Join(want, ",") {
		t.Fatalf("selected = %#v, want %#v", selected, want)
	}

	agent := lookupOnboardAgent(cfg)
	if agent == nil {
		t.Fatal("expected default agent config to be created")
	}
	if strings.Join(agent.Skills, ",") != strings.Join(want, ",") {
		t.Fatalf("agent.Skills = %#v, want %#v", agent.Skills, want)
	}
	if !agent.Default || agent.ID != "main" {
		t.Fatalf("agent = %#v, want default main agent", *agent)
	}
}

func onboardSkillIndex(t *testing.T, options []onboardSkillOption, name string) int {
	t.Helper()
	for i, option := range options {
		if option.name == name {
			return i + 1
		}
	}
	t.Fatalf("skill %q not found in options %#v", name, options)
	return 0
}

func newLineReader(input string) *bufio.Reader {
	return bufio.NewReader(strings.NewReader(input))
}
