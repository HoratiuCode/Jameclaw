package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/sipeed/jameclaw/cmd/jameclaw/internal"
	agentcore "github.com/sipeed/jameclaw/pkg/agent"
	"github.com/sipeed/jameclaw/pkg/commands"
	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/pkg/extensions"
	"github.com/sipeed/jameclaw/pkg/logger"
	"github.com/sipeed/jameclaw/pkg/providers"
	"github.com/sipeed/jameclaw/web/backend/launcherconfig"
)

type terminalEntry struct {
	id        string
	kind      string
	text      string
	reasoning string
	status    string
	tool      string
	started   time.Time
	duration  time.Duration
}

type terminalChat struct {
	app        *tview.Application
	loop       *agentcore.AgentLoop
	sessionKey string
	agentEmoji string
	chat       *tview.TextView
	header     *tview.TextView
	status     *tview.TextView
	footer     *tview.TextView
	input      *tview.TextArea

	mu             sync.Mutex
	entries        []terminalEntry
	links          map[string]string
	busy           bool
	busyMode       string
	showThinking   bool
	activity       string
	sessionStarted time.Time
	statusStarted  time.Time
	activeTurnID   string
	totalTokens    int
	promptTokens   int
	completion     int
	currentModel   string
	pendingInputs  []string
	localSearchOK  bool
	reasoningMode  reasoningMode
	inputHistory   []string
	historyIndex   int
	completionList []string
	suggestionText string
	backgroundSeq  int
	backgroundRuns map[string]string
	stopSpinner    chan struct{}
	spinnerFrame   string
	lastCtrlCAt    time.Time
}

func runTerminalChat(loop *agentcore.AgentLoop, sessionKey, agentEmoji string, reasoningDisplay reasoningMode) error {
	if loop == nil {
		return fmt.Errorf("agent loop is nil")
	}

	t := &terminalChat{
		app:            tview.NewApplication(),
		loop:           loop,
		sessionKey:     sessionKey,
		agentEmoji:     agentEmoji,
		links:          make(map[string]string),
		activity:       "idle",
		busyMode:       "interrupt",
		sessionStarted: time.Now(),
		backgroundRuns: make(map[string]string),
		stopSpinner:    make(chan struct{}),
		historyIndex:   -1,
		reasoningMode:  reasoningDisplay,
	}
	t.build()
	writeLastTerminalSession(sessionKey)
	t.loadHistory()
	t.renderHeaderFooter()
	t.renderChat()
	t.renderStatus()

	sub := loop.SubscribeEvents(256)
	defer loop.UnsubscribeEvents(sub.ID)
	go t.consumeEvents(sub.C)
	go t.animateStatus()
	defer close(t.stopSpinner)

	previousLogLevel := logger.GetLevel()
	logger.SetConsoleLevel(logger.FATAL)
	defer logger.SetConsoleLevel(previousLogLevel)

	return t.app.EnableMouse(true).EnablePaste(true).SetRoot(t.layout(), true).SetFocus(t.input).Run()
}

func (t *terminalChat) build() {
	t.header = tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter)
	t.status = tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter)
	t.footer = tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter)
	t.chat = tview.NewTextView().SetDynamicColors(true).SetRegions(true).SetScrollable(true).SetWrap(true)
	t.chat.SetBorder(true).SetTitle(" Conversation ")
	t.chat.SetHighlightedFunc(func(added, _, _ []string) {
		if len(added) == 0 {
			return
		}
		t.mu.Lock()
		url := t.links[added[0]]
		t.mu.Unlock()
		if url != "" {
			_ = openTerminalURL(url)
		}
		t.chat.Highlight()
	})

	t.completionList = terminalCompletions(t.loop)
	t.input = tview.NewTextArea().
		SetLabel(" You > ").
		SetPlaceholder("message or /command")
	t.input.SetBorder(true).SetTitle(" Enter/Ctrl-S: send | Ctrl-J: newline | Tab: complete | F1: help ")
	t.input.SetChangedFunc(t.updateSuggestions)

	t.app.SetInputCapture(t.handleKey)
}

func (t *terminalChat) layout() tview.Primitive {
	return tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(t.header, 1, 0, false).
		AddItem(t.chat, 0, 1, false).
		AddItem(t.status, 1, 0, false).
		AddItem(t.footer, 1, 0, false).
		AddItem(t.input, 5, 0, true)
}

func (t *terminalChat) loadHistory() {
	agent := t.loop.GetRegistry().GetDefaultAgent()
	if agent == nil {
		return
	}
	history := agent.Sessions.GetHistory(t.sessionKey)
	for i, message := range history {
		entry := terminalEntry{id: fmt.Sprintf("history-%d", i), kind: message.Role, text: message.Content}
		entry.reasoning = message.ReasoningContent
		if message.Role == "tool" {
			entry.kind = "tool"
			entry.status = "done"
		}
		t.entries = append(t.entries, entry)
	}
}

func (t *terminalChat) submit() {
	text := strings.TrimSpace(t.input.GetText())
	if text == "" {
		return
	}
	t.input.SetText("", true)
	t.inputHistory = append(t.inputHistory, text)
	t.historyIndex = -1
	if text == "/quit" || text == "/exit" {
		t.app.Stop()
		return
	}
	if t.handleLocalCommand(text) {
		return
	}

	t.mu.Lock()
	isBusy := t.busy
	t.entries = append(t.entries, terminalEntry{id: fmt.Sprintf("user-%d", time.Now().UnixNano()), kind: "user", text: text})
	t.trimEntriesLocked()
	t.mu.Unlock()
	t.refreshChat()

	if isBusy {
		if t.currentBusyMode() == "queue" {
			t.mu.Lock()
			t.pendingInputs = append(t.pendingInputs, text)
			depth := len(t.pendingInputs)
			t.mu.Unlock()
			t.addSystem(fmt.Sprintf("Queued for next turn (%d pending).", depth))
			t.setActivity("queued follow-up")
		} else if err := t.loop.Steer(providers.Message{Role: "user", Content: text}); err != nil {
			t.addSystem("Unable to queue message: " + err.Error())
		} else {
			t.setActivity("interrupt queued")
		}
		return
	}

	t.startForegroundTurn(text)
}

func (t *terminalChat) startForegroundTurn(text string) {
	t.mu.Lock()
	t.busy = true
	t.statusStarted = time.Now()
	t.activity = "waiting"
	t.mu.Unlock()
	t.refreshStatus()

	go func() {
		response, err := t.loop.ProcessDirect(context.Background(), text, t.sessionKey)
		if err != nil {
			t.addSystem("Run error: " + err.Error())
		}
		if response != "" {
			t.ensureFinalResponse(response)
		}
		t.mu.Lock()
		t.busy = false
		t.activity = "idle"
		t.activeTurnID = ""
		t.mu.Unlock()
		t.refreshAll()
		t.runPendingInputs()
	}()
}

func (t *terminalChat) runPendingInputs() {
	for {
		t.mu.Lock()
		if len(t.pendingInputs) == 0 {
			t.mu.Unlock()
			return
		}
		next := t.pendingInputs[0]
		t.pendingInputs = t.pendingInputs[1:]
		t.entries = append(t.entries, terminalEntry{id: fmt.Sprintf("queued-%d", time.Now().UnixNano()), kind: "user", text: next})
		t.busy = true
		t.statusStarted = time.Now()
		t.activity = "queued turn"
		t.mu.Unlock()
		t.refreshAll()

		response, err := t.loop.ProcessDirect(context.Background(), next, t.sessionKey)
		if err != nil {
			t.addSystem("Queued run error: " + err.Error())
		}
		if response != "" {
			t.ensureFinalResponse(response)
		}
		t.mu.Lock()
		t.busy = false
		t.activity = "idle"
		t.activeTurnID = ""
		t.mu.Unlock()
		t.refreshAll()
	}
}

func (t *terminalChat) currentBusyMode() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.busyMode
}

type terminalPickerItem struct {
	Main      string
	Shortcut  rune
	Secondary string
	Action    func()
}

func (t *terminalChat) showListOverlay(title, hint string, items []terminalPickerItem) {
	list := tview.NewList().
		SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGreen).Bold(true)).
		SetHighlightFullLine(true)
	list.SetBorder(true).SetTitle(" " + title + " ")
	for _, item := range items {
		action := item.Action
		list.AddItem(item.Main, item.Secondary, item.Shortcut, func() {
			if action != nil {
				action()
			}
			t.restoreLayout()
		})
	}
	list.SetDoneFunc(t.restoreLayout)

	frame := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(tview.NewBox(), 0, 1, false).
			AddItem(list, 0, 4, true).
			AddItem(tview.NewBox(), 0, 1, false), 0, 4, true).
		AddItem(tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter).SetText("[gray]"+hint+"[-]"), 1, 0, false)
	t.app.SetRoot(frame, true).SetFocus(list)
}

func (t *terminalChat) restoreLayout() {
	t.app.SetRoot(t.layout(), true).SetFocus(t.input)
	t.refreshAll()
}

func (t *terminalChat) openModelPicker() {
	cfg, err := internal.LoadConfig()
	if err != nil {
		t.addSystem("Unable to load models: " + err.Error())
		return
	}
	if len(cfg.ModelList) == 0 {
		t.addSystem("No models configured. Add one with `jameclaw model add <provider> <preset>`.")
		return
	}

	items := make([]terminalPickerItem, 0, len(cfg.ModelList))
	defaultModel := cfg.Agents.Defaults.GetModelName()
	for _, model := range cfg.ModelList {
		if model == nil || strings.TrimSpace(model.ModelName) == "" {
			continue
		}
		name := model.ModelName
		label := name
		if name == defaultModel {
			label += "  (config default)"
		}
		secondary := model.Model
		if model.APIKey() == "" {
			secondary += "  no API key configured"
		}
		items = append(items, terminalPickerItem{
			Main:      label,
			Secondary: secondary,
			Action: func() {
				t.runAgentCommand("/switch model to " + name)
			},
		})
	}
	if len(items) == 0 {
		t.addSystem("No selectable models found in model_list.")
		return
	}
	t.showListOverlay("MODELS", "Enter selects model, Esc returns to chat", items)
}

func (t *terminalChat) runAgentCommand(text string) {
	t.addSystem("Running " + text)
	go func() {
		response, err := t.loop.ProcessDirect(context.Background(), text, t.sessionKey)
		if err != nil {
			t.addSystem("Command failed: " + err.Error())
			return
		}
		if response != "" {
			t.addSystem(response)
		}
		if strings.HasPrefix(text, "/switch model to ") {
			t.mu.Lock()
			t.currentModel = strings.TrimSpace(strings.TrimPrefix(text, "/switch model to "))
			t.mu.Unlock()
			t.refreshAll()
		}
	}()
}

func (t *terminalChat) openSettingsOverlay() {
	t.mu.Lock()
	showThinking := t.showThinking
	searchAllowed := t.localSearchOK
	busyMode := t.busyMode
	t.mu.Unlock()

	items := []terminalPickerItem{
		{
			Main:      "Busy mode: " + busyMode,
			Secondary: "Toggle interrupt/queue behavior for messages sent during a run",
			Action: func() {
				t.mu.Lock()
				if t.busyMode == "interrupt" {
					t.busyMode = "queue"
				} else {
					t.busyMode = "interrupt"
				}
				mode := t.busyMode
				t.mu.Unlock()
				t.addSystem("Busy mode set to " + mode + ".")
			},
		},
		{
			Main:      "Thinking: " + onOff(showThinking),
			Secondary: "Toggle reasoning display when available",
			Action: func() {
				t.mu.Lock()
				t.showThinking = !t.showThinking
				enabled := t.showThinking
				t.mu.Unlock()
				t.addSystem("Thinking display " + onOff(enabled) + ".")
			},
		},
		{
			Main:      "Computer search: " + onOff(searchAllowed),
			Secondary: "Toggle local /computer-search permission for this session",
			Action: func() {
				t.mu.Lock()
				t.localSearchOK = !t.localSearchOK
				enabled := t.localSearchOK
				t.mu.Unlock()
				t.addSystem("Computer search " + onOff(enabled) + ".")
			},
		},
		{
			Main:      "Reload history",
			Secondary: "Reload current session from disk",
			Action: func() {
				t.mu.Lock()
				t.entries = nil
				t.mu.Unlock()
				t.loadHistory()
				t.addSystem("History reloaded.")
			},
		},
		{
			Main:      "Clear view",
			Secondary: "Clear visible terminal chat entries",
			Action: func() {
				t.mu.Lock()
				t.entries = nil
				t.mu.Unlock()
				t.refreshChat()
			},
		},
	}
	t.showListOverlay("SETTINGS", "Enter toggles or runs action, Esc returns to chat", items)
}

func (t *terminalChat) handleLocalCommand(text string) bool {
	trimmed := strings.TrimSpace(text)
	lower := strings.ToLower(trimmed)
	switch {
	case lower == "/help":
		t.addSystem("Local TUI commands: /help, /models, /sessions, /settings, /gateway-status, /auth [provider], /webui, /background <prompt>, /busy interrupt|queue, /allow-computer-search on|off, /computer-search <query> [path], /clear, /status, /quit. Shortcuts: Enter/Ctrl-S send, Ctrl-J newline, Tab complete, Ctrl-C clear/warn/exit, Ctrl-X hard abort, Ctrl-G graceful stop, Ctrl-T reasoning, Ctrl-R reload history, Ctrl-L clear view, Ctrl-D exit.")
		return true
	case lower == "/models":
		t.openModelPicker()
		return true
	case lower == "/sessions":
		t.openSessionPicker()
		return true
	case lower == "/settings":
		t.openSettingsOverlay()
		return true
	case lower == "/gateway-status":
		t.showGatewayStatus()
		return true
	case lower == "/auth":
		t.openAuthProviderPicker()
		return true
	case strings.HasPrefix(lower, "/auth "):
		provider := strings.TrimSpace(trimmed[len("/auth "):])
		t.runAuth(provider)
		return true
	case lower == "/webui":
		t.openWebUI()
		return true
	case lower == "/clear":
		t.mu.Lock()
		t.entries = nil
		t.mu.Unlock()
		t.refreshChat()
		return true
	case lower == "/status":
		t.addSystem(t.statusSummary())
		return true
	case lower == "/busy":
		t.addSystem("Busy input mode is " + t.currentBusyMode() + ". Use /busy interrupt or /busy queue.")
		return true
	case strings.HasPrefix(lower, "/busy "):
		mode := strings.TrimSpace(strings.TrimPrefix(lower, "/busy "))
		if mode != "interrupt" && mode != "queue" {
			t.addSystem("Usage: /busy interrupt|queue")
			return true
		}
		t.mu.Lock()
		t.busyMode = mode
		t.mu.Unlock()
		t.addSystem("Busy input mode set to " + mode + ".")
		t.refreshAll()
		return true
	case lower == "/allow-computer-search":
		t.addSystem("Computer search is " + onOff(t.isLocalSearchAllowed()) + ". Use /allow-computer-search on or /allow-computer-search off.")
		return true
	case strings.HasPrefix(lower, "/allow-computer-search "):
		value := strings.TrimSpace(strings.TrimPrefix(lower, "/allow-computer-search "))
		switch value {
		case "on", "yes", "true", "allow":
			t.mu.Lock()
			t.localSearchOK = true
			t.mu.Unlock()
			t.addSystem("Computer search enabled for this terminal session. Use /computer-search <query> [path].")
		case "off", "no", "false", "deny":
			t.mu.Lock()
			t.localSearchOK = false
			t.mu.Unlock()
			t.addSystem("Computer search disabled.")
		default:
			t.addSystem("Usage: /allow-computer-search on|off")
		}
		return true
	case strings.HasPrefix(lower, "/computer-search "):
		args := strings.TrimSpace(trimmed[len("/computer-search "):])
		t.runComputerSearch(args)
		return true
	case strings.HasPrefix(lower, "/background "):
		prompt := strings.TrimSpace(trimmed[len("/background "):])
		if prompt == "" {
			t.addSystem("Usage: /background <prompt>")
			return true
		}
		t.startBackground(prompt)
		return true
	}
	return false
}

func (t *terminalChat) isLocalSearchAllowed() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.localSearchOK
}

func (t *terminalChat) openWebUI() {
	baseURL := terminalWebBaseURL()
	if webUIReachable(baseURL) {
		openURL := launcherAuthenticatedURL(baseURL)
		if err := openTerminalURL(openURL); err != nil {
			t.addSystem("WebUI is running, but failed to open browser: " + err.Error())
			return
		}
		t.addSystem("Opened running WebUI in browser: " + baseURL)
		return
	}

	binary, err := resolveCompanionBinary("JAMECLAW_WEB_BINARY", "jameclaw-web")
	if err != nil {
		t.addSystem(fmt.Sprintf("WebUI is not running and the launcher binary was not found: %v. Build/install jameclaw-web, then run /webui again.", err))
		return
	}
	cmd := exec.Command(binary, "-no-browser")
	if err := cmd.Start(); err != nil {
		t.addSystem("Failed to start WebUI: " + err.Error())
		return
	}
	t.addSystem(fmt.Sprintf("Started WebUI launcher (%s). Waiting for %s ...", binary, baseURL))
	go func() {
		if !waitForWebUI(baseURL, 8*time.Second) {
			t.addSystem("Started WebUI launcher, but it is not responding at " + baseURL + " yet. Try /webui again in a moment.")
			return
		}
		openURL := launcherAuthenticatedURL(baseURL)
		if err := openTerminalURL(openURL); err != nil {
			t.addSystem("WebUI started, but failed to open browser: " + err.Error())
			return
		}
		t.addSystem("WebUI started and opened in browser: " + baseURL)
	}()
}

func (t *terminalChat) openSessionPicker() {
	sessions, err := listTerminalSessions()
	if err != nil {
		t.addSystem("Unable to list sessions: " + err.Error())
		return
	}
	if len(sessions) == 0 {
		t.addSystem("No saved sessions found.")
		return
	}
	items := make([]terminalPickerItem, 0, len(sessions))
	for _, session := range sessions {
		sess := session
		label := sess.Key
		if sess.Title != "" {
			label = sess.Title
		}
		items = append(items, terminalPickerItem{
			Main:      label,
			Secondary: sess.Key + "  " + formatAge(sess.Updated),
			Action: func() {
				t.switchSession(sess.Key)
			},
		})
	}
	t.showListOverlay("SESSIONS", "Enter resumes session, Esc returns to chat", items)
}

func (t *terminalChat) switchSession(sessionKey string) {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return
	}
	t.mu.Lock()
	t.sessionKey = sessionKey
	t.sessionStarted = time.Now()
	t.entries = nil
	t.pendingInputs = nil
	t.mu.Unlock()
	writeLastTerminalSession(sessionKey)
	setTerminalAgentTitle(t.agentEmoji, sessionKey)
	t.loadHistory()
	t.addSystem("Switched to session " + sessionKey + ".")
	t.refreshAll()
}

func (t *terminalChat) showGatewayStatus() {
	go func() {
		status := fetchGatewayStatusSummary()
		t.addSystem(status)
	}()
}

func (t *terminalChat) openAuthProviderPicker() {
	cfg, _ := internal.LoadConfig()
	providers := extensions.ProviderCatalog(cfg)
	items := make([]terminalPickerItem, 0, len(providers))
	for _, provider := range providers {
		p := provider
		items = append(items, terminalPickerItem{
			Main:      p.Name,
			Secondary: p.ID,
			Action: func() {
				t.runAuth(p.ID)
			},
		})
	}
	t.showListOverlay("AUTH PROVIDERS", "Enter runs auth login for provider, Esc returns to chat", items)
}

func (t *terminalChat) runAuth(provider string) {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		t.openAuthProviderPicker()
		return
	}
	t.app.Suspend(func() {
		cmd := exec.Command("jameclaw", "auth", "login", "--provider", provider)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
	})
	t.addSystem("Auth flow finished for " + provider + ".")
}

func (t *terminalChat) runComputerSearch(args string) {
	if !t.isLocalSearchAllowed() {
		t.addSystem("Computer search is disabled. Run /allow-computer-search on first.")
		return
	}
	query, root := parseComputerSearchArgs(args)
	if query == "" {
		t.addSystem("Usage: /computer-search <query> [path]")
		return
	}
	if root == "" {
		agent := t.loop.GetRegistry().GetDefaultAgent()
		if agent != nil && strings.TrimSpace(agent.Workspace) != "" {
			root = agent.Workspace
		} else if cwd, err := os.Getwd(); err == nil {
			root = cwd
		}
	}
	if root == "" {
		t.addSystem("Computer search needs a path. Usage: /computer-search <query> [path]")
		return
	}
	t.addSystem(fmt.Sprintf("Searching %s for %q ...", root, query))
	go func() {
		results, err := searchComputerFiles(root, query, 20)
		if err != nil {
			t.addSystem("Computer search failed: " + err.Error())
			return
		}
		t.addSystem(formatComputerSearchResults(root, query, results))
	}()
}

func (t *terminalChat) startBackground(prompt string) {
	t.mu.Lock()
	t.backgroundSeq++
	id := fmt.Sprintf("bg-%d", t.backgroundSeq)
	sessionKey := fmt.Sprintf("%s:%s", t.sessionKey, id)
	t.backgroundRuns[id] = "running"
	t.entries = append(t.entries, terminalEntry{kind: "system", text: fmt.Sprintf("Background %s started: %s", id, truncateText(prompt, 96))})
	t.trimEntriesLocked()
	t.mu.Unlock()
	t.refreshAll()

	go func() {
		started := time.Now()
		response, err := t.loop.ProcessDirect(context.Background(), prompt, sessionKey)
		t.mu.Lock()
		defer t.mu.Unlock()
		if err != nil {
			t.backgroundRuns[id] = "error"
			t.entries = append(t.entries, terminalEntry{kind: "system", text: fmt.Sprintf("Background %s failed after %s: %v", id, formatDuration(time.Since(started)), err)})
		} else {
			t.backgroundRuns[id] = "done"
			t.entries = append(t.entries, terminalEntry{kind: "assistant", id: id, text: fmt.Sprintf("Background %s finished in %s\n\n%s", id, formatDuration(time.Since(started)), response), status: "done"})
		}
		t.trimEntriesLocked()
		go t.refreshAll()
	}()
}

func (t *terminalChat) consumeEvents(events <-chan agentcore.Event) {
	for event := range events {
		if event.Meta.SessionKey != "" && event.Meta.SessionKey != t.sessionKey {
			continue
		}
		t.handleEvent(event)
	}
}

func (t *terminalChat) handleEvent(event agentcore.Event) {
	t.mu.Lock()
	switch payload := event.Payload.(type) {
	case agentcore.TurnStartPayload:
		t.busy = true
		t.activeTurnID = event.Meta.TurnID
		t.activity = "running"
		t.statusStarted = event.Time
	case agentcore.LLMRequestPayload:
		t.activity = "waiting"
		if payload.Model != "" {
			t.currentModel = payload.Model
		}
	case agentcore.LLMDeltaPayload:
		t.activity = "streaming"
		t.upsertAssistantLocked(event.Meta.TurnID, payload.Content, "streaming", "")
	case agentcore.LLMResponsePayload:
		t.promptTokens = payload.PromptTokens
		t.completion = payload.CompletionTokens
		if payload.TotalTokens > 0 {
			t.totalTokens = payload.TotalTokens
		}
		if payload.Content != "" {
			t.upsertAssistantLocked(event.Meta.TurnID, payload.Content, "done", payload.ReasoningContent)
		}
	case agentcore.ToolExecStartPayload:
		t.activity = "running tool"
		t.entries = append(t.entries, terminalEntry{
			id: event.Meta.TurnID + fmt.Sprintf("-tool-%d", len(t.entries)), kind: "tool",
			tool: payload.Tool, text: formatToolArgs(payload.Arguments), status: "running", started: event.Time,
		})
	case agentcore.ToolExecEndPayload:
		t.finishToolLocked(payload.Tool, payload.IsError, payload.Duration)
	case agentcore.ToolExecSkippedPayload:
		t.entries = append(t.entries, terminalEntry{kind: "system", text: "Tool skipped: " + payload.Tool + " - " + payload.Reason})
	case agentcore.ReasoningStepPayload:
		if t.reasoningMode != reasoningOff {
			t.activity = payload.Step
			text := "[reasoning] " + payload.Step + ": " + payload.Summary
			if t.reasoningMode == reasoningDebug {
				text = debugReasoningLine(event.Kind.String(), event.Meta, payload)
			}
			t.entries = append(t.entries, terminalEntry{kind: "system", text: text})
		}
	case agentcore.VerificationStartPayload:
		if t.reasoningMode != reasoningOff {
			t.activity = "verifying"
			text := "[verify] " + payload.Command + " (" + payload.Reason + ")"
			if t.reasoningMode == reasoningDebug {
				text = debugReasoningLine(event.Kind.String(), event.Meta, payload)
			}
			t.entries = append(t.entries, terminalEntry{kind: "system", text: text})
		}
	case agentcore.VerificationEndPayload:
		if t.reasoningMode != reasoningOff {
			status := "passed"
			if payload.IsError {
				status = "failed"
			}
			text := fmt.Sprintf("[verify] %s %s in %s", payload.Command, status, payload.Duration.Round(time.Millisecond))
			if t.reasoningMode == reasoningDebug {
				text = debugReasoningLine(event.Kind.String(), event.Meta, payload)
			}
			t.entries = append(t.entries, terminalEntry{kind: "system", text: text})
		}
	case agentcore.LLMRetryPayload:
		t.activity = "retrying"
	case agentcore.TurnEndPayload:
		t.busy = false
		t.activity = string(payload.Status)
		if payload.Status == agentcore.TurnEndStatusCompleted {
			t.activity = "idle"
		}
	case agentcore.ErrorPayload:
		t.activity = "error"
		t.entries = append(t.entries, terminalEntry{kind: "system", text: payload.Stage + ": " + payload.Message})
	}
	t.trimEntriesLocked()
	t.mu.Unlock()
	t.refreshAll()
}

func (t *terminalChat) upsertAssistantLocked(id, text, status, reasoning string) {
	if id == "" {
		id = "active"
	}
	for i := len(t.entries) - 1; i >= 0; i-- {
		if t.entries[i].kind == "assistant" && t.entries[i].id == id {
			t.entries[i].text = text
			t.entries[i].status = status
			if reasoning != "" {
				t.entries[i].reasoning = reasoning
			}
			return
		}
	}
	t.entries = append(t.entries, terminalEntry{id: id, kind: "assistant", text: text, status: status, reasoning: reasoning})
}

func (t *terminalChat) ensureFinalResponse(response string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for i := len(t.entries) - 1; i >= 0; i-- {
		if t.entries[i].kind == "assistant" && t.entries[i].text == response {
			return
		}
	}
	t.upsertAssistantLocked(t.activeTurnID, response, "done", "")
}

func (t *terminalChat) finishToolLocked(tool string, failed bool, duration time.Duration) {
	for i := len(t.entries) - 1; i >= 0; i-- {
		if t.entries[i].kind == "tool" && t.entries[i].tool == tool && t.entries[i].status == "running" {
			if failed {
				t.entries[i].status = "error"
			} else {
				t.entries[i].status = "done"
			}
			t.entries[i].duration = duration
			return
		}
	}
}

func (t *terminalChat) trimEntriesLocked() {
	const maxEntries = 300
	if len(t.entries) > maxEntries {
		t.entries = append([]terminalEntry(nil), t.entries[len(t.entries)-maxEntries:]...)
	}
}

func (t *terminalChat) addSystem(text string) {
	t.mu.Lock()
	t.entries = append(t.entries, terminalEntry{kind: "system", text: text})
	t.trimEntriesLocked()
	t.mu.Unlock()
	t.refreshChat()
}

func (t *terminalChat) setActivity(activity string) {
	t.mu.Lock()
	t.activity = activity
	t.mu.Unlock()
	t.refreshStatus()
}

func (t *terminalChat) refreshAll() {
	go t.app.QueueUpdateDraw(func() {
		t.renderHeaderFooter()
		t.renderChat()
		t.renderStatus()
	})
}

func (t *terminalChat) refreshChat() {
	go t.app.QueueUpdateDraw(t.renderChat)
}

func (t *terminalChat) refreshStatus() {
	go t.app.QueueUpdateDraw(t.renderStatus)
}

func (t *terminalChat) renderHeaderFooter() {
	agent := t.loop.GetRegistry().GetDefaultAgent()
	agentID, model := "main", "unknown"
	contextWindow := 0
	if agent != nil {
		agentID, model, contextWindow = agent.ID, agent.Model, agent.ContextWindow
	}
	if t.currentModel != "" {
		model = t.currentModel
	}
	t.mu.Lock()
	busyMode, pending, background, activity := t.busyMode, len(t.pendingInputs), len(t.backgroundRuns), t.activity
	started := t.sessionStarted
	t.mu.Unlock()
	t.header.SetText(fmt.Sprintf("[::b]%s JameClaw Terminal Chat[-:-:-]  [gray]agent %s | session %s | connected[-]", tview.Escape(t.agentEmoji), tview.Escape(agentID), tview.Escape(t.sessionKey)))
	usage := formatUsage(t.totalTokens, contextWindow)
	t.footer.SetText(fmt.Sprintf("[gray]model %s | session %s | activity %s | %s | duration %s | busy %s | pending %d | bg %d | thinking %s | F1 help[-]",
		tview.Escape(model), tview.Escape(t.sessionKey), tview.Escape(activity), usage, formatDuration(time.Since(started)), busyMode, pending, background, onOff(t.showThinking)))
}

func (t *terminalChat) renderStatus() {
	t.mu.Lock()
	activity, busy, started, spinner, suggestion := t.activity, t.busy, t.statusStarted, t.spinnerFrame, t.suggestionText
	t.mu.Unlock()
	if busy && spinner != "" {
		activity = spinner + " " + activity
	} else if !busy && suggestion != "" {
		activity = suggestion
	}
	color := "green"
	if activity == "error" || activity == "aborted" {
		color = "red"
	} else if busy {
		color = "yellow"
	}
	elapsed := ""
	if busy && !started.IsZero() {
		elapsed = fmt.Sprintf(" • %.1fs", time.Since(started).Seconds())
	}
	t.status.SetText(fmt.Sprintf("[%s]%s%s[-]", color, tview.Escape(activity), elapsed))
}

func (t *terminalChat) renderChat() {
	t.mu.Lock()
	entries := append([]terminalEntry(nil), t.entries...)
	showThinking := t.showThinking
	t.links = make(map[string]string)
	t.mu.Unlock()

	var out strings.Builder
	for i, entry := range entries {
		if i > 0 {
			out.WriteString("\n\n")
		}
		switch entry.kind {
		case "user":
			out.WriteString("[deepskyblue::b]You[-:-:-]\n")
			out.WriteString(t.renderMarkdown(entry.text))
		case "assistant":
			label := t.agentEmoji
			if entry.status == "streaming" {
				label += " streaming"
			}
			out.WriteString("[green::b]" + tview.Escape(label) + "[-:-:-]\n")
			if showThinking && entry.reasoning != "" {
				out.WriteString("[gray::i]Reasoning\n" + tview.Escape(entry.reasoning) + "[-:-:-]\n\n")
			}
			out.WriteString(t.renderMarkdown(entry.text))
		case "tool":
			color := "yellow"
			if entry.status == "done" {
				color = "green"
			} else if entry.status == "error" {
				color = "red"
			}
			dur := entry.duration
			if entry.status == "running" && !entry.started.IsZero() {
				dur = time.Since(entry.started)
			}
			duration := ""
			if dur > 0 {
				duration = " " + formatDuration(dur)
			}
			out.WriteString(fmt.Sprintf("[%s::b]Tool: %s (%s%s)[-:-:-]\n[gray]%s[-]", color, tview.Escape(entry.tool), entry.status, duration, tview.Escape(entry.text)))
		default:
			out.WriteString("[gray::i]" + tview.Escape(entry.text) + "[-:-:-]")
		}
	}
	t.chat.SetText(out.String()).ScrollToEnd()
}

func (t *terminalChat) renderMarkdown(text string) string {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	var out strings.Builder
	inCode := false
	language := ""
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			if !inCode {
				language = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "```"))
				out.WriteString("[gray::b]" + tview.Escape(language) + "[-:-:-]")
			}
			inCode = !inCode
			if i < len(lines)-1 {
				out.WriteByte('\n')
			}
			continue
		}
		if inCode {
			out.WriteString(highlightCode(line, language))
		} else {
			trimmed := strings.TrimSpace(line)
			switch {
			case strings.HasPrefix(trimmed, "### "):
				out.WriteString("[purple::b]" + t.renderInline(strings.TrimPrefix(trimmed, "### ")) + "[-:-:-]")
			case strings.HasPrefix(trimmed, "## "):
				out.WriteString("[purple::b]" + t.renderInline(strings.TrimPrefix(trimmed, "## ")) + "[-:-:-]")
			case strings.HasPrefix(trimmed, "# "):
				out.WriteString("[purple::b]" + t.renderInline(strings.TrimPrefix(trimmed, "# ")) + "[-:-:-]")
			case strings.HasPrefix(trimmed, "> "):
				out.WriteString("[gray::i]│ " + t.renderInline(strings.TrimPrefix(trimmed, "> ")) + "[-:-:-]")
			case strings.HasPrefix(trimmed, "- "), strings.HasPrefix(trimmed, "* "):
				out.WriteString("[yellow]•[-] " + t.renderInline(trimmed[2:]))
			default:
				out.WriteString(t.renderInline(line))
			}
		}
		if i < len(lines)-1 {
			out.WriteByte('\n')
		}
	}
	return out.String()
}

var markdownLinkPattern = regexp.MustCompile(`\[([^\]]+)\]\((https?://[^\s)]+)\)`)
var bareURLPattern = regexp.MustCompile(`https?://[^\s<>()]+`)
var inlineCodePattern = regexp.MustCompile("`([^`]+)`")

func (t *terminalChat) renderInline(text string) string {
	var out strings.Builder
	position := 0
	for _, match := range markdownLinkPattern.FindAllStringSubmatchIndex(text, -1) {
		out.WriteString(t.renderBareInline(text[position:match[0]]))
		label := text[match[2]:match[3]]
		url := text[match[4]:match[5]]
		out.WriteString(t.linkTag(label, url))
		position = match[1]
	}
	out.WriteString(t.renderBareInline(text[position:]))
	return out.String()
}

func (t *terminalChat) renderBareInline(text string) string {
	var out strings.Builder
	position := 0
	for _, match := range bareURLPattern.FindAllStringIndex(text, -1) {
		out.WriteString(renderInlineCode(text[position:match[0]]))
		url := text[match[0]:match[1]]
		out.WriteString(t.linkTag(url, url))
		position = match[1]
	}
	out.WriteString(renderInlineCode(text[position:]))
	return out.String()
}

func renderInlineCode(text string) string {
	escaped := tview.Escape(text)
	return inlineCodePattern.ReplaceAllString(escaped, "[orange]$1[-]")
}

func (t *terminalChat) linkTag(label, url string) string {
	t.mu.Lock()
	id := fmt.Sprintf("link-%d", len(t.links)+1)
	t.links[id] = url
	t.mu.Unlock()
	return fmt.Sprintf("[blue::u][\"%s\"]%s[\"\"][-:-:-]", id, tview.Escape(label))
}

var codeKeywordPattern = regexp.MustCompile(`\b(func|package|import|return|if|else|for|range|type|struct|interface|const|var|let|class|def|async|await|try|catch|throw|switch|case|break|continue|true|false|null|nil)\b`)
var codeStringPattern = regexp.MustCompile(`("[^"\n]*"|'[^'\n]*')`)

func highlightCode(line, _ string) string {
	escaped := tview.Escape(line)
	escaped = codeStringPattern.ReplaceAllString(escaped, "[green]$1[-]")
	escaped = codeKeywordPattern.ReplaceAllString(escaped, "[purple::b]$1[-:-:-]")
	return "[gray]│[-] " + escaped
}

func (t *terminalChat) autocomplete(current string) []string {
	if !strings.HasPrefix(strings.TrimSpace(current), "/") {
		return nil
	}
	prefix := strings.ToLower(strings.TrimSpace(current))
	var matches []string
	for _, candidate := range t.completionList {
		if strings.HasPrefix(strings.ToLower(candidate), prefix) {
			matches = append(matches, candidate)
		}
	}
	if len(matches) > 12 {
		matches = matches[:12]
	}
	return matches
}

func terminalCompletions(loop *agentcore.AgentLoop) []string {
	values := []string{
		"/quit",
		"/exit",
		"/help",
		"/models",
		"/sessions",
		"/settings",
		"/gateway-status",
		"/auth ",
		"/webui",
		"/background ",
		"/busy interrupt",
		"/busy queue",
		"/allow-computer-search on",
		"/allow-computer-search off",
		"/computer-search ",
		"/clear",
		"/status",
	}
	for _, definition := range commands.BuiltinDefinitions() {
		values = append(values, definition.EffectiveUsage())
		for _, sub := range definition.SubCommands {
			value := "/" + definition.Name + " " + sub.Name
			if sub.ArgsUsage != "" {
				value += " " + sub.ArgsUsage
			}
			values = append(values, value)
		}
	}
	if loop != nil && loop.GetRegistry() != nil {
		for _, id := range loop.GetRegistry().ListAgentIDs() {
			values = append(values, "/use agent "+id)
		}
	}
	sort.Strings(values)
	return values
}

func (t *terminalChat) handleKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEnter, tcell.KeyCtrlS:
		t.submit()
		return nil
	case tcell.KeyCtrlJ:
		t.input.SetText(t.input.GetText()+"\n", true)
		return nil
	case tcell.KeyTab:
		if t.completeInput() {
			return nil
		}
	case tcell.KeyCtrlD:
		t.app.Stop()
		return nil
	case tcell.KeyCtrlC:
		if action := resolveCtrlCAction(t.input.GetText(), t.lastCtrlCAt, time.Now()); action == "clear" {
			t.input.SetText("", true)
			t.mu.Lock()
			t.suggestionText = "input cleared"
			t.lastCtrlCAt = time.Now()
			t.mu.Unlock()
			t.refreshStatus()
		} else if action == "warn" {
			t.mu.Lock()
			t.suggestionText = "press Ctrl-C again to exit"
			t.lastCtrlCAt = time.Now()
			t.mu.Unlock()
			t.refreshStatus()
		} else {
			t.app.Stop()
		}
		return nil
	case tcell.KeyCtrlX:
		if err := t.loop.HardAbort(t.sessionKey); err != nil {
			t.addSystem(err.Error())
		} else {
			t.setActivity("aborted")
		}
		return nil
	case tcell.KeyCtrlG:
		if err := t.loop.InterruptGraceful("Finish safely and return the best partial result."); err != nil {
			t.addSystem(err.Error())
		} else {
			t.setActivity("stopping gracefully")
		}
		return nil
	case tcell.KeyCtrlT:
		t.mu.Lock()
		t.showThinking = !t.showThinking
		t.mu.Unlock()
		t.refreshAll()
		return nil
	case tcell.KeyCtrlL:
		t.mu.Lock()
		t.entries = nil
		t.mu.Unlock()
		t.refreshChat()
		return nil
	case tcell.KeyCtrlR:
		t.mu.Lock()
		t.entries = nil
		t.mu.Unlock()
		t.loadHistory()
		t.refreshChat()
		return nil
	case tcell.KeyUp:
		if t.input.GetText() == "" || t.historyIndex >= 0 {
			t.navigateInputHistory(-1)
			return nil
		}
	case tcell.KeyDown:
		if t.historyIndex >= 0 {
			t.navigateInputHistory(1)
			return nil
		}
	case tcell.KeyPgUp:
		row, col := t.chat.GetScrollOffset()
		t.chat.ScrollTo(max(0, row-10), col)
		return nil
	case tcell.KeyPgDn:
		row, col := t.chat.GetScrollOffset()
		t.chat.ScrollTo(row+10, col)
		return nil
	case tcell.KeyF1:
		t.addSystem("Shortcuts: Enter/Ctrl-S send; Ctrl-J newline; submit while busy interrupts or queues based on /busy; Tab autocomplete; Ctrl-C clear/warn/exit; Ctrl-X hard abort; Ctrl-G graceful stop; Ctrl-T reasoning; Ctrl-R reload history; Ctrl-L clear view; PageUp/PageDown scroll; Ctrl-D exit. Local commands: /models, /sessions, /settings, /gateway-status, /auth [provider], /background <prompt>, /busy interrupt|queue, /status, /clear.")
		return nil
	}
	return event
}

func resolveCtrlCAction(input string, lastCtrlCAt, now time.Time) string {
	if strings.TrimSpace(input) != "" {
		return "clear"
	}
	if !lastCtrlCAt.IsZero() && now.Sub(lastCtrlCAt) <= time.Second {
		return "exit"
	}
	return "warn"
}

func (t *terminalChat) navigateInputHistory(direction int) {
	if len(t.inputHistory) == 0 {
		return
	}
	if t.historyIndex < 0 {
		t.historyIndex = len(t.inputHistory)
	}
	t.historyIndex += direction
	if t.historyIndex < 0 {
		t.historyIndex = 0
	}
	if t.historyIndex >= len(t.inputHistory) {
		t.historyIndex = -1
		t.input.SetText("", true)
		return
	}
	t.input.SetText(t.inputHistory[t.historyIndex], true)
}

func (t *terminalChat) completeInput() bool {
	text := strings.TrimSpace(t.input.GetText())
	matches := t.autocomplete(text)
	if len(matches) == 0 {
		return false
	}
	t.input.SetText(matches[0], true)
	t.mu.Lock()
	t.suggestionText = "completed: " + matches[0]
	t.mu.Unlock()
	t.refreshStatus()
	return true
}

func (t *terminalChat) updateSuggestions() {
	matches := t.autocomplete(t.input.GetText())
	t.mu.Lock()
	if len(matches) == 0 {
		t.suggestionText = ""
	} else {
		if len(matches) > 4 {
			matches = matches[:4]
		}
		t.suggestionText = "suggest: " + strings.Join(matches, "  ")
	}
	t.mu.Unlock()
	t.refreshStatus()
}

func (t *terminalChat) animateStatus() {
	ticker := time.NewTicker(120 * time.Millisecond)
	defer ticker.Stop()
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frame := 0
	for {
		select {
		case <-ticker.C:
			t.mu.Lock()
			busy := t.busy
			if busy {
				t.spinnerFrame = frames[frame%len(frames)]
				frame++
			} else {
				t.spinnerFrame = ""
			}
			t.mu.Unlock()
			if busy {
				t.refreshStatus()
			}
		case <-t.stopSpinner:
			return
		}
	}
}

func formatToolArgs(arguments map[string]any) string {
	if len(arguments) == 0 {
		return ""
	}
	parts := make([]string, 0, len(arguments))
	for key, value := range arguments {
		parts = append(parts, fmt.Sprintf("%s=%v", key, value))
	}
	sort.Strings(parts)
	return strings.Join(parts, " ")
}

func formatTokenCount(value int) string {
	if value >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(value)/1_000_000)
	}
	if value >= 1_000 {
		return fmt.Sprintf("%.1fk", float64(value)/1_000)
	}
	return fmt.Sprintf("%d", value)
}

func formatUsage(totalTokens, contextWindow int) string {
	if contextWindow <= 0 {
		return fmt.Sprintf("tokens %s / n/a", formatTokenCount(totalTokens))
	}
	percent := int(float64(totalTokens) / float64(contextWindow) * 100)
	return fmt.Sprintf("tokens %s / %s (%d%%)", formatTokenCount(totalTokens), formatTokenCount(contextWindow), percent)
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
}

func truncateText(text string, maxLen int) string {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
	if maxLen <= 0 || len(text) <= maxLen {
		return text
	}
	if maxLen <= 3 {
		return text[:maxLen]
	}
	return text[:maxLen-3] + "..."
}

type computerSearchResult struct {
	Path    string
	Line    int
	Snippet string
	Kind    string
}

func parseComputerSearchArgs(args string) (string, string) {
	parts := strings.Fields(strings.TrimSpace(args))
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return strings.Join(parts[:len(parts)-1], " "), parts[len(parts)-1]
}

func searchComputerFiles(root, query string, limit int) ([]computerSearchResult, error) {
	if limit <= 0 {
		limit = 20
	}
	expandedRoot := expandUserPath(root)
	absRoot, err := filepath.Abs(expandedRoot)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(absRoot)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", absRoot)
	}

	queryLower := strings.ToLower(query)
	var results []computerSearchResult
	err = filepath.WalkDir(absRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if len(results) >= limit {
			return filepath.SkipAll
		}
		name := d.Name()
		if d.IsDir() {
			if shouldSkipSearchDir(name) && path != absRoot {
				return filepath.SkipDir
			}
			return nil
		}
		if shouldSkipSearchFile(name) {
			return nil
		}
		rel, _ := filepath.Rel(absRoot, path)
		if strings.Contains(strings.ToLower(rel), queryLower) {
			results = append(results, computerSearchResult{Path: path, Kind: "path", Snippet: rel})
			return nil
		}
		match, line, snippet := searchFileContent(path, queryLower)
		if match {
			results = append(results, computerSearchResult{Path: path, Line: line, Kind: "content", Snippet: snippet})
		}
		return nil
	})
	return results, err
}

func searchFileContent(path, queryLower string) (bool, int, string) {
	file, err := os.Open(path)
	if err != nil {
		return false, 0, ""
	}
	defer file.Close()

	buf := make([]byte, 8192)
	n, _ := file.Read(buf)
	if n > 0 && strings.Contains(string(buf[:n]), "\x00") {
		return false, 0, ""
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return false, 0, ""
	}

	reader := io.LimitReader(file, 512*1024)
	data, err := io.ReadAll(reader)
	if err != nil {
		return false, 0, ""
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), queryLower) {
			return true, i + 1, strings.TrimSpace(line)
		}
	}
	return false, 0, ""
}

func formatComputerSearchResults(root, query string, results []computerSearchResult) string {
	if len(results) == 0 {
		return fmt.Sprintf("No computer search results for %q in %s.", query, root)
	}
	var out strings.Builder
	fmt.Fprintf(&out, "Computer search results for %q in %s:\n", query, root)
	for i, result := range results {
		if result.Line > 0 {
			fmt.Fprintf(&out, "%d. %s:%d\n   %s\n", i+1, result.Path, result.Line, truncateText(result.Snippet, 180))
		} else {
			fmt.Fprintf(&out, "%d. %s\n   %s\n", i+1, result.Path, truncateText(result.Snippet, 180))
		}
	}
	return strings.TrimSpace(out.String())
}

func shouldSkipSearchDir(name string) bool {
	switch name {
	case ".git", ".hg", ".svn", "node_modules", "vendor", "dist", "build", ".next", ".turbo", ".cache", ".venv", "venv", "__pycache__":
		return true
	default:
		return strings.HasPrefix(name, ".")
	}
}

func shouldSkipSearchFile(name string) bool {
	lower := strings.ToLower(name)
	if strings.HasPrefix(lower, ".") {
		return true
	}
	for _, suffix := range []string{
		".png", ".jpg", ".jpeg", ".gif", ".webp", ".ico", ".icns", ".pdf", ".zip", ".gz", ".tar", ".mp3", ".mp4", ".mov", ".wav", ".woff", ".woff2", ".ttf", ".otf",
	} {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}

func expandUserPath(path string) string {
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func resolveCompanionBinary(envVar, binaryName string) (string, error) {
	if custom := os.Getenv(envVar); custom != "" {
		if info, err := os.Stat(custom); err == nil && !info.IsDir() {
			return custom, nil
		}
	}

	name := binaryName
	if runtime.GOOS == "windows" {
		name += ".exe"
	}

	var candidates []string
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates, filepath.Join(exeDir, name), filepath.Join(exeDir, "build", name))
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, "build", name))
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("%s binary not found", binaryName)
}

func webUIReachable(url string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 700*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < http.StatusInternalServerError
}

func waitForWebUI(url string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if webUIReachable(url) {
			return true
		}
		time.Sleep(250 * time.Millisecond)
	}
	return webUIReachable(url)
}

func launcherAuthenticatedURL(baseURL string) string {
	token := readLauncherAccessToken()
	if token == "" {
		return baseURL
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}
	query := parsed.Query()
	query.Set("access_token", token)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func readLauncherAccessToken() string {
	data, err := os.ReadFile(launcherAccessTokenPath())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func launcherAccessTokenPath() string {
	if home := strings.TrimSpace(os.Getenv("JAMECLAW_HOME")); home != "" {
		return filepath.Join(home, "launcher_access_token")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".jameclaw", "launcher_access_token")
	}
	return "launcher_access_token"
}

func terminalLastSessionPath() string {
	return filepath.Join(internal.GetJameclawHome(), "terminal-last-session")
}

func readLastTerminalSession() string {
	data, err := os.ReadFile(terminalLastSessionPath())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func writeLastTerminalSession(sessionKey string) {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return
	}
	path := terminalLastSessionPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return
	}
	_ = os.WriteFile(path, []byte(sessionKey+"\n"), 0o600)
}

type terminalSessionSummary struct {
	Key     string
	Title   string
	Updated time.Time
}

func listTerminalSessions() ([]terminalSessionSummary, error) {
	cfg, err := internal.LoadConfig()
	if err != nil {
		return nil, err
	}
	workspace := cfg.WorkspacePath()
	if workspace == "" {
		workspace = filepath.Join(internal.GetJameclawHome(), "workspace")
	}
	dir := filepath.Join(expandUserPath(workspace), "sessions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	seen := make(map[string]struct{})
	var sessions []terminalSessionSummary
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		switch {
		case strings.HasSuffix(name, ".json") && !strings.HasSuffix(name, ".meta.json"):
			session, ok := readTerminalJSONSession(filepath.Join(dir, name))
			if ok {
				if _, exists := seen[session.Key]; !exists {
					seen[session.Key] = struct{}{}
					sessions = append(sessions, session)
				}
			}
		case strings.HasSuffix(name, ".meta.json"):
			session, ok := readTerminalMetaSession(filepath.Join(dir, name))
			if ok {
				if _, exists := seen[session.Key]; !exists {
					seen[session.Key] = struct{}{}
					sessions = append(sessions, session)
				}
			}
		}
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Updated.After(sessions[j].Updated)
	})
	if len(sessions) > 50 {
		sessions = sessions[:50]
	}
	return sessions, nil
}

func readTerminalJSONSession(path string) (terminalSessionSummary, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return terminalSessionSummary{}, false
	}
	var raw struct {
		Key      string              `json:"key"`
		Summary  string              `json:"summary"`
		Messages []providers.Message `json:"messages"`
		Updated  time.Time           `json:"updated"`
	}
	if err := json.Unmarshal(data, &raw); err != nil || strings.TrimSpace(raw.Key) == "" {
		return terminalSessionSummary{}, false
	}
	title := strings.TrimSpace(raw.Summary)
	if title == "" {
		for _, msg := range raw.Messages {
			if msg.Role == "user" && strings.TrimSpace(msg.Content) != "" {
				title = truncateText(msg.Content, 80)
				break
			}
		}
	}
	return terminalSessionSummary{Key: raw.Key, Title: title, Updated: raw.Updated}, true
}

func readTerminalMetaSession(path string) (terminalSessionSummary, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return terminalSessionSummary{}, false
	}
	var raw struct {
		Key       string    `json:"key"`
		Summary   string    `json:"summary"`
		UpdatedAt time.Time `json:"updated_at"`
	}
	if err := json.Unmarshal(data, &raw); err != nil || strings.TrimSpace(raw.Key) == "" {
		return terminalSessionSummary{}, false
	}
	return terminalSessionSummary{Key: raw.Key, Title: truncateText(raw.Summary, 80), Updated: raw.UpdatedAt}, true
}

func formatAge(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
}

func terminalWebBaseURL() string {
	launcherCfg, err := launcherconfig.Load(
		launcherconfig.PathForAppConfig(internal.GetConfigPath()),
		launcherconfig.Default(),
	)
	if err != nil || launcherCfg.Port <= 0 {
		return "http://localhost:18800"
	}
	return "http://localhost:" + strconv.Itoa(launcherCfg.Port)
}

func fetchGatewayStatusSummary() string {
	webURL := strings.TrimRight(terminalWebBaseURL(), "/") + "/api/gateway/status"
	if data, ok := fetchJSONMap(webURL, 2*time.Second); ok {
		return formatGatewayStatusMap("Web Console API", data)
	}

	cfg, err := config.LoadConfig(internal.GetConfigPath())
	if err != nil {
		return "Gateway status unavailable: " + err.Error()
	}
	port := cfg.Gateway.Port
	if port == 0 {
		port = 18790
	}
	healthURL := "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(port)) + "/health"
	if data, ok := fetchJSONMap(healthURL, 2*time.Second); ok {
		return formatGatewayStatusMap("Gateway health", data)
	}
	return "Gateway status unavailable: Web Console and gateway health endpoint are not reachable."
}

func fetchJSONMap(rawURL string, timeout time.Duration) (map[string]any, bool) {
	client := http.Client{Timeout: timeout}
	resp, err := client.Get(rawURL)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusInternalServerError {
		return nil, false
	}
	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, false
	}
	return data, true
}

func formatGatewayStatusMap(source string, data map[string]any) string {
	keys := []string{
		"gateway_status",
		"status",
		"pid",
		"config_default_model",
		"boot_default_model",
		"gateway_restart_required",
		"gateway_start_allowed",
		"gateway_start_reason",
	}
	var parts []string
	for _, key := range keys {
		if value, ok := data[key]; ok {
			parts = append(parts, fmt.Sprintf("%s=%v", key, value))
		}
	}
	if len(parts) == 0 {
		return source + ": reachable"
	}
	return source + ": " + strings.Join(parts, ", ")
}

func (t *terminalChat) statusSummary() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return fmt.Sprintf("Status: activity=%s, busy=%t, busy_mode=%s, tokens=%s, prompt=%s, completion=%s, pending=%d, background=%d, session_duration=%s",
		t.activity,
		t.busy,
		t.busyMode,
		formatTokenCount(t.totalTokens),
		formatTokenCount(t.promptTokens),
		formatTokenCount(t.completion),
		len(t.pendingInputs),
		len(t.backgroundRuns),
		formatDuration(time.Since(t.sessionStarted)),
	)
}

func onOff(value bool) string {
	if value {
		return "on"
	}
	return "off"
}

func openTerminalURL(url string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("open", url)
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		command = exec.Command("xdg-open", url)
	}
	return command.Start()
}
