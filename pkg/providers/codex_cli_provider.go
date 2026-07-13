package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const codexCliPathEnv = "JAMECLAW_CODEX_CLI_PATH"

// CodexCliProvider implements LLMProvider by wrapping the codex CLI as a subprocess.
type CodexCliProvider struct {
	command   string
	workspace string
}

// NewCodexCliProvider creates a new Codex CLI provider.
func NewCodexCliProvider(workspace string) *CodexCliProvider {
	return &CodexCliProvider{
		command:   resolveCodexCLICommand(),
		workspace: workspace,
	}
}

// Chat implements LLMProvider.Chat by executing the codex CLI in non-interactive mode.
func (p *CodexCliProvider) Chat(
	ctx context.Context, messages []Message, tools []ToolDefinition, model string, options map[string]any,
) (*LLMResponse, error) {
	if p.command == "" {
		return nil, fmt.Errorf("codex command not configured")
	}

	preparedMessages, cleanup, extraDirs, err := p.prepareMediaMessages(messages)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	prompt := p.buildPrompt(preparedMessages, tools)

	args := []string{
		"exec",
		"--json",
		"--dangerously-bypass-approvals-and-sandbox",
		"--skip-git-repo-check",
		"--color", "never",
	}
	if model != "" && model != "codex-cli" {
		args = append(args, "-m", model)
	}
	if p.workspace != "" {
		args = append(args, "-C", p.workspace)
	}
	for _, dir := range extraDirs {
		args = append(args, "--add-dir", dir)
	}
	args = append(args, "-") // read prompt from stdin

	cmd := exec.CommandContext(ctx, p.command, args...)
	cmd.Env = codexCLIEnv(p.command)
	cmd.Stdin = bytes.NewReader([]byte(prompt))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	// Parse JSONL from stdout even if exit code is non-zero,
	// because codex writes diagnostic noise to stderr (e.g. rollout errors)
	// but still produces valid JSONL output.
	if stdoutStr := stdout.String(); stdoutStr != "" {
		resp, parseErr := p.parseJSONLEvents(stdoutStr)
		if parseErr == nil && resp != nil && (resp.Content != "" || len(resp.ToolCalls) > 0) {
			return resp, nil
		}
	}

	if err != nil {
		if ctx.Err() == context.Canceled {
			return nil, ctx.Err()
		}
		if stderrStr := stderr.String(); stderrStr != "" {
			return nil, fmt.Errorf("codex cli error: %s", stderrStr)
		}
		return nil, fmt.Errorf("codex cli error: %w", err)
	}

	return p.parseJSONLEvents(stdout.String())
}

func resolveCodexCLICommand() string {
	if custom := strings.TrimSpace(os.Getenv(codexCliPathEnv)); custom != "" {
		if isExecutableFile(custom) {
			return custom
		}
	}

	if home, err := os.UserHomeDir(); err == nil && home != "" {
		candidates := nvmCodexCandidates(filepath.Join(home, ".nvm", "versions", "node"))
		for _, candidate := range candidates {
			if isExecutableFile(candidate) {
				return candidate
			}
		}
	}

	if path, err := exec.LookPath("codex"); err == nil && path != "" {
		return path
	}
	return "codex"
}

func nvmCodexCandidates(nodeVersionsDir string) []string {
	matches, err := filepath.Glob(filepath.Join(nodeVersionsDir, "v*", "bin", "codex"))
	if err != nil || len(matches) == 0 {
		return nil
	}
	sort.Slice(matches, func(i, j int) bool {
		return compareNodeVersion(filepath.Base(filepath.Dir(filepath.Dir(matches[i]))), filepath.Base(filepath.Dir(filepath.Dir(matches[j])))) > 0
	})
	return matches
}

func compareNodeVersion(a, b string) int {
	aa := parseNodeVersion(a)
	bb := parseNodeVersion(b)
	for i := range aa {
		if aa[i] > bb[i] {
			return 1
		}
		if aa[i] < bb[i] {
			return -1
		}
	}
	return strings.Compare(a, b)
}

func parseNodeVersion(version string) [3]int {
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	parts := strings.Split(version, ".")
	var parsed [3]int
	for i := 0; i < len(parsed) && i < len(parts); i++ {
		n, _ := strconv.Atoi(parts[i])
		parsed[i] = n
	}
	return parsed
}

func isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Mode()&0o111 != 0
}

func codexCLIEnv(command string) []string {
	env := os.Environ()
	if command == "" || !filepath.IsAbs(command) {
		return env
	}
	binDir := filepath.Dir(command)
	if binDir == "." || binDir == "" {
		return env
	}
	pathValue := os.Getenv("PATH")
	nextPath := binDir
	if pathValue != "" {
		nextPath += string(os.PathListSeparator) + pathValue
	}
	replaced := false
	for i, item := range env {
		if strings.HasPrefix(item, "PATH=") {
			env[i] = "PATH=" + nextPath
			replaced = true
			break
		}
	}
	if !replaced {
		env = append(env, "PATH="+nextPath)
	}
	return env
}

type stagedCodexMedia struct {
	MediaType string
	Path      string
}

// prepareMediaMessages materializes data URL media into local files so the Codex
// CLI can inspect them during the run. The current CLI only has native --image
// attachments; file staging gives audio transcription requests a usable path.
func (p *CodexCliProvider) prepareMediaMessages(messages []Message) ([]Message, func(), []string, error) {
	prepared := make([]Message, len(messages))
	copy(prepared, messages)

	var tempDirs []string
	for i := range prepared {
		if len(prepared[i].Media) == 0 {
			continue
		}

		staged := make([]stagedCodexMedia, 0, len(prepared[i].Media))
		for mediaIndex, mediaURL := range prepared[i].Media {
			mediaType, data, ok := parseDataMediaURL(mediaURL)
			if !ok {
				staged = append(staged, stagedCodexMedia{MediaType: "media", Path: mediaURL})
				continue
			}

			tmpDir, err := os.MkdirTemp("", "jameclaw-codex-cli-media-*")
			if err != nil {
				cleanupTempDirs(tempDirs)
				return nil, func() {}, nil, fmt.Errorf("staging codex cli media: %w", err)
			}
			tempDirs = append(tempDirs, tmpDir)

			ext := mediaExtension(mediaType)
			path := filepath.Join(tmpDir, fmt.Sprintf("media-%d%s", mediaIndex+1, ext))
			if err := os.WriteFile(path, data, 0o600); err != nil {
				cleanupTempDirs(tempDirs)
				return nil, func() {}, nil, fmt.Errorf("writing codex cli media: %w", err)
			}
			staged = append(staged, stagedCodexMedia{MediaType: mediaType, Path: path})
		}

		prepared[i].Content = appendCodexMediaReferences(prepared[i].Content, staged)
		prepared[i].Media = nil
	}

	cleanup := func() {
		cleanupTempDirs(tempDirs)
	}
	return prepared, cleanup, tempDirs, nil
}

func parseDataMediaURL(mediaURL string) (string, []byte, bool) {
	if !strings.HasPrefix(mediaURL, "data:") {
		return "", nil, false
	}

	header, encoded, found := strings.Cut(strings.TrimPrefix(mediaURL, "data:"), ",")
	if !found || !strings.Contains(header, ";base64") {
		return "", nil, false
	}

	mediaType, _, _ := strings.Cut(header, ";")
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", nil, false
	}
	return mediaType, data, true
}

func mediaExtension(mediaType string) string {
	switch strings.ToLower(mediaType) {
	case "audio/wav", "audio/wave", "audio/x-wav":
		return ".wav"
	case "audio/mpeg", "audio/mp3":
		return ".mp3"
	case "audio/ogg":
		return ".ogg"
	case "audio/webm":
		return ".webm"
	case "audio/mp4", "audio/m4a":
		return ".m4a"
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	default:
		return ".bin"
	}
}

func appendCodexMediaReferences(content string, media []stagedCodexMedia) string {
	if len(media) == 0 {
		return content
	}

	var sb strings.Builder
	sb.WriteString(strings.TrimSpace(content))
	if sb.Len() > 0 {
		sb.WriteString("\n\n")
	}
	sb.WriteString("Attached media files:\n")
	for _, item := range media {
		sb.WriteString("- ")
		sb.WriteString(item.MediaType)
		sb.WriteString(": ")
		sb.WriteString(item.Path)
		sb.WriteString("\n")
	}
	sb.WriteString("\nUse these attached media files when answering.")
	return sb.String()
}

func cleanupTempDirs(dirs []string) {
	for _, dir := range dirs {
		_ = os.RemoveAll(dir)
	}
}

// GetDefaultModel returns the default model identifier.
func (p *CodexCliProvider) GetDefaultModel() string {
	return "codex-cli"
}

// buildPrompt converts messages to a prompt string for the Codex CLI.
// System messages are prepended as instructions since Codex CLI has no --system-prompt flag.
func (p *CodexCliProvider) buildPrompt(messages []Message, tools []ToolDefinition) string {
	var systemParts []string
	var conversationParts []string

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			systemParts = append(systemParts, msg.Content)
		case "user":
			conversationParts = append(conversationParts, msg.Content)
		case "assistant":
			conversationParts = append(conversationParts, "Assistant: "+msg.Content)
		case "tool":
			conversationParts = append(conversationParts,
				fmt.Sprintf("[Tool Result for %s]: %s", msg.ToolCallID, msg.Content))
		}
	}

	var sb strings.Builder

	if len(systemParts) > 0 {
		sb.WriteString("## System Instructions\n\n")
		sb.WriteString(strings.Join(systemParts, "\n\n"))
		sb.WriteString("\n\n## Task\n\n")
	}

	if len(tools) > 0 {
		sb.WriteString(buildCLIToolsPrompt(tools))
		sb.WriteString("\n\n")
	}

	// Simplify single user message (no prefix)
	if len(conversationParts) == 1 && len(systemParts) == 0 && len(tools) == 0 {
		return conversationParts[0]
	}

	sb.WriteString(strings.Join(conversationParts, "\n"))
	return sb.String()
}

// codexEvent represents a single JSONL event from `codex exec --json`.
type codexEvent struct {
	Type     string          `json:"type"`
	ThreadID string          `json:"thread_id,omitempty"`
	Message  string          `json:"message,omitempty"`
	Item     *codexEventItem `json:"item,omitempty"`
	Usage    *codexUsage     `json:"usage,omitempty"`
	Error    *codexEventErr  `json:"error,omitempty"`
}

type codexEventItem struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Command  string `json:"command,omitempty"`
	Status   string `json:"status,omitempty"`
	ExitCode *int   `json:"exit_code,omitempty"`
	Output   string `json:"output,omitempty"`
}

type codexUsage struct {
	InputTokens       int `json:"input_tokens"`
	CachedInputTokens int `json:"cached_input_tokens"`
	OutputTokens      int `json:"output_tokens"`
}

type codexEventErr struct {
	Message string `json:"message"`
}

// parseJSONLEvents processes the JSONL output from codex exec --json.
func (p *CodexCliProvider) parseJSONLEvents(output string) (*LLMResponse, error) {
	var contentParts []string
	var usage *UsageInfo
	var lastError string

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var event codexEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue // skip malformed lines
		}

		switch event.Type {
		case "item.completed":
			if event.Item != nil && event.Item.Type == "agent_message" && event.Item.Text != "" {
				contentParts = append(contentParts, event.Item.Text)
			}
		case "turn.completed":
			if event.Usage != nil {
				promptTokens := event.Usage.InputTokens + event.Usage.CachedInputTokens
				usage = &UsageInfo{
					PromptTokens:     promptTokens,
					CompletionTokens: event.Usage.OutputTokens,
					TotalTokens:      promptTokens + event.Usage.OutputTokens,
				}
			}
		case "error":
			lastError = event.Message
		case "turn.failed":
			if event.Error != nil {
				lastError = event.Error.Message
			}
		}
	}

	if lastError != "" && len(contentParts) == 0 {
		return nil, fmt.Errorf("codex cli: %s", lastError)
	}

	content := strings.Join(contentParts, "\n")

	// Extract tool calls from response text (same pattern as ClaudeCliProvider)
	toolCalls := extractToolCallsFromText(content)

	finishReason := "stop"
	if len(toolCalls) > 0 {
		finishReason = "tool_calls"
		content = stripToolCallsFromText(content)
	}

	return &LLMResponse{
		Content:      strings.TrimSpace(content),
		ToolCalls:    toolCalls,
		FinishReason: finishReason,
		Usage:        usage,
	}, nil
}
