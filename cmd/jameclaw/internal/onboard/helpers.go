package onboard

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"

	"github.com/sipeed/jameclaw/cmd/jameclaw/internal"
	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/pkg/credential"
)

const (
	onboardANSIReset    = "\033[0m"
	onboardANSITitle    = "\033[1;38;2;224;90;71m"
	onboardANSIDim      = "\033[38;2;140;146;125m"
	onboardANSIActive   = "\033[1;38;2;120;196;88m"
	onboardANSIInactive = "\033[1;38;2;240;98;83m"
	onboardANSIRail     = "\033[38;2;140;146;125m"
	onboardANSIStrong   = "\033[1;38;2;224;90;71m"
	onboardANSIBgClear  = "\033[H\033[2J"
)

var onboardInput io.Reader = os.Stdin
var onboardOutput io.Writer = os.Stdout

type onboardModelOption struct {
	key            string
	label          string
	description    string
	modelName      string
	requiresAPIKey bool
	keyLabel       string
}

type onboardSelection struct {
	modelName       string
	modelConfigured bool
	signatureEmoji  string
	telegramEnabled bool
}

const (
	defaultAgentSignatureEmoji = "🦐"
	agentNameLinePrefix        = "Your name is JameClaw"
)

var onboardModelOptions = []onboardModelOption{
	{
		key:            "1",
		label:          "OpenAI GPT-5.4",
		description:    "Use GPT-5.4 with your OpenAI API key.",
		modelName:      "gpt-5.4",
		requiresAPIKey: true,
		keyLabel:       "OpenAI API key",
	},
	{
		key:            "2",
		label:          "Anthropic Claude Sonnet 4.6",
		description:    "Use Claude Sonnet 4.6 with your Anthropic API key.",
		modelName:      "claude-sonnet-4.6",
		requiresAPIKey: true,
		keyLabel:       "Anthropic API key",
	},
	{
		key:            "3",
		label:          "OpenRouter Auto",
		description:    "Use OpenRouter and let it pick the best route.",
		modelName:      "openrouter-auto",
		requiresAPIKey: true,
		keyLabel:       "OpenRouter API key",
	},
	{
		key:            "4",
		label:          "Local Ollama llama3",
		description:    "Use a local Ollama model at http://localhost:11434/v1.",
		modelName:      "llama3",
		requiresAPIKey: false,
	},
}

func Run(encrypt bool) bool {
	return onboard(encrypt)
}

func IsComplete() bool {
	configPath := internal.GetConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return false
	}

	if cfg == nil || cfg.WorkspacePath() == "" {
		return false
	}

	if _, err := os.Stat(configPath); err != nil {
		return false
	}

	return configReadyForChat(cfg)
}

func onboard(encrypt bool) bool {
	configPath := internal.GetConfigPath()
	renderOnboardIntro()
	reader := bufio.NewReader(onboardInput)

	configExists := false
	if _, err := os.Stat(configPath); err == nil {
		configExists = true
		if encrypt {
			// Only ask for confirmation when *both* config and SSH key already exist,
			// indicating a full re-onboard that would reset the config to defaults.
			sshKeyPath, _ := credential.DefaultSSHKeyPath()
			if _, err := os.Stat(sshKeyPath); err == nil {
				// Both exist — confirm a full reset.
				onboardWriteLine("Config already exists at %s", configPath)
				overwriteConfig, promptErr := promptYesNo(reader, "Overwrite config with defaults?", false)
				if promptErr != nil {
					fmt.Fprintf(onboardOutput, "Error: %v\n", promptErr)
					os.Exit(1)
				}
				if !overwriteConfig {
					onboardWriteLine("Aborted.")
					return true
				}
				configExists = false // user agreed to reset; treat as fresh
			}
			// Config exists but SSH key is missing — keep existing config, only add SSH key.
		}
	}

	var err error
	if encrypt {
		onboardWriteLine("")
		onboardWriteLine("Set up credential encryption")
		onboardWriteLine("-----------------------------")
		passphrase, pErr := promptPassphrase()
		if pErr != nil {
			fmt.Fprintf(onboardOutput, "Error: %v\n", pErr)
			os.Exit(1)
		}
		// Expose the passphrase to credential.PassphraseProvider (which calls
		// os.Getenv by default) so that SaveConfig can encrypt api_keys.
		// This process is a one-shot CLI tool; the env var is never exposed outside
		// the current process and disappears when it exits.
		os.Setenv(credential.PassphraseEnvVar, passphrase)

		if err = setupSSHKey(); err != nil {
			fmt.Fprintf(onboardOutput, "Error generating SSH key: %v\n", err)
			os.Exit(1)
		}
	}

	var cfg *config.Config
	if configExists {
		// Preserve the existing config; SaveConfig will re-encrypt api_keys with the new passphrase.
		cfg, err = config.LoadConfig(configPath)
		if err != nil {
			fmt.Fprintf(onboardOutput, "Error loading existing config: %v\n", err)
			os.Exit(1)
		}
	} else {
		cfg = config.DefaultConfig()
	}

	workspace := cfg.WorkspacePath()
	existingSignature := readAgentSignatureEmoji(workspace)
	createWorkspaceTemplates(workspace)
	if err := applyAgentSignatureEmoji(workspace, existingSignature); err != nil {
		fmt.Fprintf(onboardOutput, "Error applying agent signature: %v\n", err)
	}

	selection, wizardErr := runOnboardWizard(cfg, configExists, encrypt, configPath, workspace, existingSignature)
	if wizardErr != nil {
		fmt.Fprintf(onboardOutput, "Error during onboarding: %v\n", wizardErr)
		os.Exit(1)
	}

	if err := config.SaveConfig(configPath, cfg); err != nil {
		fmt.Fprintf(onboardOutput, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	renderOnboardSummary(configExists, encrypt, configPath, workspace, selection)
	return false
}

func renderOnboardIntro() {
	fmt.Fprint(onboardOutput, onboardANSIBgClear)
	onboardWriteLine("%sJameClaw Onboard%s", onboardANSITitle, onboardANSIReset)
	onboardWriteLine("%sPrepare your config, workspace, and chat flow.%s", onboardANSIDim, onboardANSIReset)
	onboardWriteLine("")
}

func renderOnboardSummary(configExists, encrypt bool, configPath, workspace string, selection onboardSelection) {
	fmt.Fprint(onboardOutput, onboardANSIBgClear)
	onboardWriteLine("%sJameClaw Onboard%s", onboardANSITitle, onboardANSIReset)
	if selection.modelConfigured {
		onboardWriteLine("%sSetup complete. The next time you run `jameclaw`, you can choose terminal agent, dashboard, or web console.%s", onboardANSIDim, onboardANSIReset)
	} else {
		onboardWriteLine("%sSetup is saved, but chat is not ready yet. Run `jameclaw onboard` again to finish model setup.%s", onboardANSIDim, onboardANSIReset)
	}
	onboardWriteLine("")

	statusLine := "Created a fresh configuration and workspace scaffold."
	if configExists {
		statusLine = "Using your existing configuration and refreshing workspace templates."
	}

	renderOnboardStep("◆", onboardANSIActive, "Config Ready", statusLine)
	onboardWriteLine("  %s│%s %sConfig:%s %s", onboardANSIRail, onboardANSIReset, onboardANSIStrong, onboardANSIReset, configPath)
	onboardWriteLine("  %s│%s %sWorkspace:%s %s", onboardANSIRail, onboardANSIReset, onboardANSIStrong, onboardANSIReset, workspace)

	modelMarker := "◇"
	modelColor := onboardANSIInactive
	modelCopy := "Choose a model and rerun onboarding to enable chat."
	if selection.modelConfigured {
		modelMarker = "◆"
		modelColor = onboardANSIActive
		modelCopy = fmt.Sprintf("Default chat model: %s", selection.modelName)
	}
	renderOnboardStep(modelMarker, modelColor, "Model Setup", modelCopy)
	if encrypt {
		onboardWriteLine("  %s│%s export JAMECLAW_KEY_PASSPHRASE=<your-passphrase>", onboardANSIRail, onboardANSIReset)
	}
	if selection.modelConfigured {
		onboardWriteLine("  %s│%s Terminal chat is ready.", onboardANSIRail, onboardANSIReset)
	} else {
		onboardWriteLine("  %s│%s OpenRouter: https://openrouter.ai/keys", onboardANSIRail, onboardANSIReset)
		onboardWriteLine("  %s│%s Ollama:     https://ollama.com", onboardANSIRail, onboardANSIReset)
	}

	renderOnboardStep("◆", onboardANSIActive, "Personalization", fmt.Sprintf("Agent signature emoji: %s", selection.signatureEmoji))
	onboardWriteLine("  %s│%s Applied to %s/AGENT.md.", onboardANSIRail, onboardANSIReset, workspace)

	telegramMarker := "◇"
	telegramColor := onboardANSIInactive
	telegramCopy := "Telegram is optional. You can connect it later in config.json."
	if selection.telegramEnabled {
		telegramMarker = "◆"
		telegramColor = onboardANSIActive
		telegramCopy = "Telegram is enabled and ready for your bot token."
	}
	renderOnboardStep(telegramMarker, telegramColor, "Telegram", telegramCopy)

	startCopy := "Run `jameclaw onboard` again after you choose a model."
	startMarker := "◇"
	startColor := onboardANSIInactive
	if selection.modelConfigured {
		startCopy = "Run `jameclaw` and choose terminal agent, web console, or dashboard."
		startMarker = "◆"
		startColor = onboardANSIActive
	}
	renderOnboardStep(startMarker, startColor, "Start Chat", startCopy)
	onboardWriteLine("  %s│%s Optional web console: jameclaw-web", onboardANSIRail, onboardANSIReset)
	onboardWriteLine("  %s│%s Optional TUI dashboard: jameclaw-launcher-tui", onboardANSIRail, onboardANSIReset)
	onboardWriteLine("  %s│%s", onboardANSIRail, onboardANSIReset)
	onboardWriteLine("")
	nextCommand := "jameclaw onboard"
	if selection.modelConfigured {
		nextCommand = "jameclaw"
	}
	onboardWriteLine("%sNext command:%s %s", onboardANSIStrong, onboardANSIReset, nextCommand)
}

func renderOnboardStep(marker, markerColor, title, description string) {
	onboardWriteLine("  %s│%s", onboardANSIRail, onboardANSIReset)
	onboardWriteLine(
		"  %s%s%s %s%s%s",
		markerColor,
		marker,
		onboardANSIReset,
		onboardANSIStrong,
		title,
		onboardANSIReset,
	)
	onboardWriteLine("  %s│%s %s%s%s", onboardANSIRail, onboardANSIReset, onboardANSIDim, description, onboardANSIReset)
}

func onboardWriteLine(format string, args ...any) {
	fmt.Fprintf(onboardOutput, format+"\n", args...)
}

// promptPassphrase reads the encryption passphrase twice from the terminal
// (with echo disabled) and returns it. Returns an error if the passphrase is
// empty or if the two inputs do not match.
func promptPassphrase() (string, error) {
	reader := bufio.NewReader(onboardInput)
	p1, err := promptSecret(reader, "Enter passphrase for credential encryption: ")
	if err != nil {
		return "", fmt.Errorf("reading passphrase: %w", err)
	}
	if len(p1) == 0 {
		return "", fmt.Errorf("passphrase must not be empty")
	}

	p2, err := promptSecret(reader, "Confirm passphrase: ")
	if err != nil {
		return "", fmt.Errorf("reading passphrase confirmation: %w", err)
	}

	if p1 != p2 {
		return "", fmt.Errorf("passphrases do not match")
	}
	return p1, nil
}

// setupSSHKey generates the jameclaw-specific SSH key at ~/.ssh/jameclaw_ed25519.key.
// If the key already exists the user is warned and asked to confirm overwrite.
// Answering anything other than "y" keeps the existing key (not an error).
func setupSSHKey() error {
	reader := bufio.NewReader(onboardInput)
	keyPath, err := credential.DefaultSSHKeyPath()
	if err != nil {
		return fmt.Errorf("cannot determine SSH key path: %w", err)
	}

	if _, err := os.Stat(keyPath); err == nil {
		fmt.Fprintf(onboardOutput, "\nWARNING: %s already exists.\n", keyPath)
		onboardWriteLine("Overwriting will invalidate any credentials previously encrypted with this key.")
		confirmed, promptErr := promptYesNo(reader, "Overwrite it now?", false)
		if promptErr != nil {
			return promptErr
		}
		if !confirmed {
			onboardWriteLine("Keeping existing SSH key.")
			return nil
		}
	}

	if err := credential.GenerateSSHKey(keyPath); err != nil {
		return err
	}
	onboardWriteLine("SSH key generated: %s", keyPath)
	return nil
}

func runOnboardWizard(cfg *config.Config, configExists, encrypt bool, configPath, workspace, currentSignature string) (onboardSelection, error) {
	reader := bufio.NewReader(onboardInput)
	selection := onboardSelection{
		modelConfigured: configReadyForChat(cfg),
		signatureEmoji:  normalizeAgentSignatureEmoji(currentSignature),
		telegramEnabled: cfg.Channels.Telegram.Enabled && cfg.Channels.Telegram.Token() != "",
	}
	selection.modelName = cfg.Agents.Defaults.ModelName

	renderOnboardWizard(configExists, encrypt, configPath, workspace, selection)

	modelOption, err := promptModelChoice(reader, cfg)
	if err != nil {
		return selection, err
	}
	if modelOption != nil {
		if err := applyModelChoice(reader, cfg, *modelOption); err != nil {
			return selection, err
		}
	}

	selection.modelConfigured = configReadyForChat(cfg)
	selection.modelName = cfg.Agents.Defaults.ModelName

	signatureEmoji, err := promptAgentSignatureEmoji(reader, workspace, selection.signatureEmoji)
	if err != nil {
		return selection, err
	}
	selection.signatureEmoji = signatureEmoji

	if err := promptTelegramSetup(reader, cfg); err != nil {
		return selection, err
	}
	selection.telegramEnabled = cfg.Channels.Telegram.Enabled && cfg.Channels.Telegram.Token() != ""
	return selection, nil
}

func renderOnboardWizard(configExists, encrypt bool, configPath, workspace string, selection onboardSelection) {
	fmt.Fprint(onboardOutput, onboardANSIBgClear)
	onboardWriteLine("%sJameClaw Onboard%s", onboardANSITitle, onboardANSIReset)
	onboardWriteLine("%sSet your model, finish setup, and go straight into chat.%s", onboardANSIDim, onboardANSIReset)
	onboardWriteLine("")

	statusLine := "Fresh config and workspace ready."
	if configExists {
		statusLine = "Existing config loaded and workspace refreshed."
	}
	renderOnboardStep("◆", onboardANSIActive, "Config Ready", statusLine)
	onboardWriteLine("  %s│%s %sConfig:%s %s", onboardANSIRail, onboardANSIReset, onboardANSIStrong, onboardANSIReset, configPath)
	onboardWriteLine("  %s│%s %sWorkspace:%s %s", onboardANSIRail, onboardANSIReset, onboardANSIStrong, onboardANSIReset, workspace)
	if encrypt {
		onboardWriteLine("  %s│%s %sPassphrase:%s export JAMECLAW_KEY_PASSPHRASE before chat", onboardANSIRail, onboardANSIReset, onboardANSIStrong, onboardANSIReset)
	}

	modelDescription := "Choose the default model used by `jameclaw`."
	if selection.modelConfigured {
		modelDescription = fmt.Sprintf("Current default model: %s", selection.modelName)
	}
	renderOnboardStep("◇", onboardANSIInactive, "Model Setup", modelDescription)
	for _, option := range onboardModelOptions {
		onboardWriteLine("  %s│%s %s.%s %s", onboardANSIRail, onboardANSIReset, option.key, onboardANSIReset, option.label)
		onboardWriteLine("  %s│%s %s%s%s", onboardANSIRail, onboardANSIReset, onboardANSIDim, option.description, onboardANSIReset)
	}
	onboardWriteLine("  %s│%s 5. Skip for now", onboardANSIRail, onboardANSIReset)
	onboardWriteLine("  %s│%s %sKeep the current config and finish later.%s", onboardANSIRail, onboardANSIReset, onboardANSIDim, onboardANSIReset)

	renderOnboardStep("◇", onboardANSIInactive, "Personalization", "Choose any emoji used by the default agent identity.")
	onboardWriteLine("  %s│%s Current signature: %s", onboardANSIRail, onboardANSIReset, selection.signatureEmoji)
	onboardWriteLine("  %s│%s Press Enter to keep it, or type any emoji you want, such as 🦐, 🤖, 🐙, 🧑‍💻, or 🏴‍☠️.", onboardANSIRail, onboardANSIReset)

	renderOnboardStep("◇", onboardANSIInactive, "Telegram", "Optionally connect your Telegram bot right now.")
	onboardWriteLine("  %s│%s Paste your bot token and optional allowed user ID or username.", onboardANSIRail, onboardANSIReset)
	onboardWriteLine("  %s│%s", onboardANSIRail, onboardANSIReset)
}

func promptModelChoice(reader *bufio.Reader, cfg *config.Config) (*onboardModelOption, error) {
	defaultChoice := "5"
	current := lookupModelConfig(cfg, cfg.Agents.Defaults.ModelName)
	if current != nil {
		for _, option := range onboardModelOptions {
			if option.modelName == current.ModelName {
				defaultChoice = option.key
				break
			}
		}
	}

	line, err := promptLine(reader, fmt.Sprintf("Select model [1-5] (default %s)", defaultChoice))
	if err != nil {
		return nil, err
	}
	if line == "" {
		line = defaultChoice
	}
	if line == "5" {
		return nil, nil
	}

	for _, option := range onboardModelOptions {
		if line == option.key || strings.EqualFold(line, option.modelName) {
			return &option, nil
		}
	}

	return nil, fmt.Errorf("unknown model selection %q", line)
}

func applyModelChoice(reader *bufio.Reader, cfg *config.Config, option onboardModelOption) error {
	modelCfg := lookupModelConfig(cfg, option.modelName)
	if modelCfg == nil {
		return fmt.Errorf("model %q not found in config", option.modelName)
	}

	if option.requiresAPIKey {
		currentValue := ""
		if modelCfg.APIKey() != "" {
			currentValue = " (press Enter to keep current key)"
		}
		apiKey, err := promptSecret(reader, fmt.Sprintf("%s%s: ", option.keyLabel, currentValue))
		if err != nil {
			return err
		}
		if apiKey == "" && modelCfg.APIKey() == "" {
			return fmt.Errorf("%s is required", option.keyLabel)
		}
		if apiKey != "" {
			modelCfg.SetAPIKey(apiKey)
		}
	}

	cfg.Agents.Defaults.ModelName = option.modelName
	return nil
}

func promptTelegramSetup(reader *bufio.Reader, cfg *config.Config) error {
	enableDefault := cfg.Channels.Telegram.Enabled && cfg.Channels.Telegram.Token() != ""
	enableTelegram, err := promptYesNo(reader, "Connect Telegram now?", enableDefault)
	if err != nil {
		return err
	}
	if !enableTelegram {
		return nil
	}

	tokenPrompt := "Telegram bot token"
	if cfg.Channels.Telegram.Token() != "" {
		tokenPrompt += " (press Enter to keep current token)"
	}
	token, err := promptSecret(reader, tokenPrompt+": ")
	if err != nil {
		return err
	}
	if token == "" && cfg.Channels.Telegram.Token() == "" {
		return fmt.Errorf("telegram bot token is required when Telegram is enabled")
	}
	if token != "" {
		cfg.Channels.Telegram.SetToken(token)
	}
	cfg.Channels.Telegram.Enabled = true

	allowed := strings.Join(cfg.Channels.Telegram.AllowFrom, ",")
	allowPrompt := "Allowed Telegram user ID or username (optional, press Enter to skip)"
	if allowed != "" {
		allowPrompt = fmt.Sprintf("Allowed Telegram user ID or username (current %s, press Enter to keep)", allowed)
	}
	userID, err := promptLine(reader, allowPrompt)
	if err != nil {
		return err
	}
	switch {
	case userID == "" && len(cfg.Channels.Telegram.AllowFrom) > 0:
	case userID == "":
		cfg.Channels.Telegram.AllowFrom = nil
	default:
		cfg.Channels.Telegram.AllowFrom = config.FlexibleStringSlice{userID}
	}

	return nil
}

func promptAgentSignatureEmoji(reader *bufio.Reader, workspace, current string) (string, error) {
	current = normalizeAgentSignatureEmoji(current)
	value, err := promptLine(reader, fmt.Sprintf("Agent signature emoji, any emoji allowed (default %s)", current))
	if err != nil {
		return current, err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		value = current
	}
	value = normalizeAgentSignatureEmoji(value)
	if err := applyAgentSignatureEmoji(workspace, value); err != nil {
		return current, err
	}
	return value, nil
}

func promptLine(reader *bufio.Reader, label string) (string, error) {
	fmt.Fprintf(onboardOutput, "%s%s%s: ", onboardANSIStrong, label, onboardANSIReset)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func promptSecret(reader *bufio.Reader, label string) (string, error) {
	fmt.Fprint(onboardOutput, label)
	if file, ok := onboardInput.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		value, err := term.ReadPassword(int(file.Fd()))
		fmt.Fprintln(onboardOutput)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(value)), nil
	}

	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func promptYesNo(reader *bufio.Reader, label string, defaultYes bool) (bool, error) {
	suffix := "y/N"
	if defaultYes {
		suffix = "Y/n"
	}
	line, err := promptLine(reader, fmt.Sprintf("%s [%s]", label, suffix))
	if err != nil {
		return false, err
	}
	if line == "" {
		return defaultYes, nil
	}
	switch strings.ToLower(line) {
	case "y", "yes":
		return true, nil
	case "n", "no":
		return false, nil
	default:
		return false, fmt.Errorf("please answer y or n")
	}
}

func configReadyForChat(cfg *config.Config) bool {
	if cfg == nil || cfg.WorkspacePath() == "" || cfg.Agents.Defaults.ModelName == "" {
		return false
	}

	modelCfg := lookupModelConfig(cfg, cfg.Agents.Defaults.ModelName)
	return modelReadyForChat(modelCfg)
}

func modelReadyForChat(modelCfg *config.ModelConfig) bool {
	if modelCfg == nil {
		return false
	}
	if strings.HasPrefix(modelCfg.Model, "ollama/") {
		return true
	}
	return modelCfg.APIKey() != ""
}

func lookupModelConfig(cfg *config.Config, modelName string) *config.ModelConfig {
	if cfg == nil || modelName == "" {
		return nil
	}
	for _, modelCfg := range cfg.ModelList {
		if modelCfg.ModelName == modelName {
			return modelCfg
		}
	}
	return nil
}

func createWorkspaceTemplates(workspace string) {
	err := copyEmbeddedToTarget(workspace)
	if err != nil {
		fmt.Fprintf(onboardOutput, "Error copying workspace templates: %v\n", err)
	}
}

func copyEmbeddedToTarget(targetDir string) error {
	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("Failed to create target directory: %w", err)
	}

	// Walk through all files in embed.FS
	err := fs.WalkDir(embeddedFiles, "workspace", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Read embedded file
		data, err := embeddedFiles.ReadFile(path)
		if err != nil {
			return fmt.Errorf("Failed to read embedded file %s: %w", path, err)
		}

		new_path, err := filepath.Rel("workspace", path)
		if err != nil {
			return fmt.Errorf("Failed to get relative path for %s: %v\n", path, err)
		}

		// Build target file path
		targetPath := filepath.Join(targetDir, new_path)

		// Ensure target file's directory exists
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("Failed to create directory %s: %w", filepath.Dir(targetPath), err)
		}

		// Write file
		if err := os.WriteFile(targetPath, data, 0o644); err != nil {
			return fmt.Errorf("Failed to write file %s: %w", targetPath, err)
		}

		return nil
	})

	return err
}

func normalizeAgentSignatureEmoji(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultAgentSignatureEmoji
	}
	return value
}

func readAgentSignatureEmoji(workspace string) string {
	agentPath := filepath.Join(workspace, "AGENT.md")
	data, err := os.ReadFile(agentPath)
	if err != nil {
		return defaultAgentSignatureEmoji
	}

	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, agentNameLinePrefix) {
			continue
		}
		signature := strings.TrimSpace(strings.TrimPrefix(trimmed, agentNameLinePrefix))
		signature = strings.TrimSuffix(signature, ".")
		return normalizeAgentSignatureEmoji(signature)
	}

	return defaultAgentSignatureEmoji
}

func applyAgentSignatureEmoji(workspace, signature string) error {
	agentPath := filepath.Join(workspace, "AGENT.md")
	data, err := os.ReadFile(agentPath)
	if err != nil {
		return err
	}

	signature = normalizeAgentSignatureEmoji(signature)
	replacementLine := fmt.Sprintf("%s %s.", agentNameLinePrefix, signature)

	lines := strings.Split(string(data), "\n")
	replaced := false
	insertAfter := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "You are Jame, the default assistant for this workspace.") {
			insertAfter = i
		}
		if strings.HasPrefix(trimmed, agentNameLinePrefix) {
			lines[i] = replacementLine
			replaced = true
			break
		}
	}

	if !replaced {
		if insertAfter >= 0 {
			lines = append(lines[:insertAfter+1], append([]string{replacementLine}, lines[insertAfter+1:]...)...)
		} else {
			lines = append([]string{replacementLine}, lines...)
		}
	}

	output := strings.Join(lines, "\n")
	if !strings.HasSuffix(output, "\n") {
		output += "\n"
	}

	return os.WriteFile(agentPath, []byte(output), 0o644)
}
