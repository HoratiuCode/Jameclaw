package onboard

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/jameclaw/cmd/jameclaw/internal"
	"github.com/sipeed/jameclaw/pkg/config"
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
	if len(options) != 8 {
		t.Fatalf("loadOnboardSkillOptions() len = %d, want 8", len(options))
	}

	want := map[string]bool{
		"agent-browser": false,
		"github":        false,
		"hardware":      false,
		"moltbook":      false,
		"skill-creator": false,
		"summarize":     false,
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

func TestPromptSkillSelectionDefaultsToAllSkills(t *testing.T) {
	cfg := config.DefaultConfig()

	selected, err := promptSkillSelection(newLineReader("\n"), cfg)
	if err != nil {
		t.Fatalf("promptSkillSelection() error = %v", err)
	}

	options := loadOnboardSkillOptions()
	if len(selected) != len(options) {
		t.Fatalf("selected len = %d, want %d", len(selected), len(options))
	}
	agent := lookupOnboardAgent(cfg)
	if agent == nil {
		t.Fatal("expected default agent config to be created")
	}
	if len(agent.Skills) != len(options) {
		t.Fatalf("agent.Skills len = %d, want %d", len(agent.Skills), len(options))
	}
}

func TestPromptSkillSelectionStoresExplicitSubset(t *testing.T) {
	cfg := config.DefaultConfig()

	selected, err := promptSkillSelection(newLineReader("2 5\n"), cfg)
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

func newLineReader(input string) *bufio.Reader {
	return bufio.NewReader(strings.NewReader(input))
}
