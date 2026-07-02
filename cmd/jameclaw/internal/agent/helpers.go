package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ergochat/readline"

	"github.com/sipeed/jameclaw/cmd/jameclaw/internal"
	"github.com/sipeed/jameclaw/pkg/agent"
	"github.com/sipeed/jameclaw/pkg/bus"
	"github.com/sipeed/jameclaw/pkg/commands"
	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/pkg/logger"
	"github.com/sipeed/jameclaw/pkg/providers"
)

type reasoningMode string

const (
	reasoningOff     reasoningMode = "off"
	reasoningSummary reasoningMode = "summary"
	reasoningDebug   reasoningMode = "debug"
)

func parseReasoningMode(raw string) (reasoningMode, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "off":
		return reasoningOff, nil
	case "summary":
		return reasoningSummary, nil
	case "debug":
		return reasoningDebug, nil
	default:
		return "", fmt.Errorf("invalid --reasoning value %q; use off, summary, or debug", raw)
	}
}

func agentCmd(message, sessionKey, model string, debug bool, reasoning string) error {
	sessionKey = resolveTerminalSessionKey(sessionKey)
	reasoningDisplay, err := parseReasoningMode(reasoning)
	if err != nil {
		return err
	}

	cfg, err := internal.LoadConfig()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	if debug {
		logger.SetLevel(logger.DEBUG)
		fmt.Println("🔍 Debug mode enabled")
	}

	if model != "" {
		cfg.Agents.Defaults.ModelName = model
	}
	agentEmoji := resolveAgentEmoji(cfg)

	provider, modelID, err := providers.CreateProvider(cfg)
	if err != nil {
		return fmt.Errorf("error creating provider: %w", err)
	}

	// Use the resolved model ID from provider creation
	if modelID != "" {
		cfg.Agents.Defaults.ModelName = modelID
	}

	msgBus := bus.NewMessageBus()
	defer msgBus.Close()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, provider)
	defer agentLoop.Close()

	// Print agent startup info (only for interactive mode)
	startupInfo := agentLoop.GetStartupInfo()
	logger.InfoCF("agent", "Agent initialized",
		map[string]any{
			"tools_count":      startupInfo["tools"].(map[string]any)["count"],
			"skills_total":     startupInfo["skills"].(map[string]any)["total"],
			"skills_available": startupInfo["skills"].(map[string]any)["available"],
		})

	if message != "" {
		cleanup := streamReasoningEvents(agentLoop, sessionKey, reasoningDisplay)
		defer cleanup()
		ctx := context.Background()
		response, err := agentLoop.ProcessDirect(ctx, message, sessionKey)
		if err != nil {
			return fmt.Errorf("error processing message: %w", err)
		}
		fmt.Printf("\n%s %s\n", agentEmoji, response)
		return nil
	}

	setTerminalAgentTitle(agentEmoji, sessionKey)

	if err := runTerminalChat(agentLoop, sessionKey, agentEmoji, reasoningDisplay); err != nil {
		fmt.Printf("Terminal UI unavailable: %v\nFalling back to readline mode.\n\n", err)
		interactiveMode(agentLoop, sessionKey, agentEmoji, reasoningDisplay)
	}

	return nil
}

func streamReasoningEvents(agentLoop *agent.AgentLoop, sessionKey string, mode reasoningMode) func() {
	if mode == reasoningOff || agentLoop == nil {
		return func() {}
	}
	sub := agentLoop.SubscribeEvents(128)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for event := range sub.C {
			if event.Meta.SessionKey != "" && event.Meta.SessionKey != sessionKey {
				continue
			}
			if text := formatReasoningEvent(event, mode); text != "" {
				fmt.Fprintln(os.Stderr, text)
			}
		}
	}()
	return func() {
		agentLoop.UnsubscribeEvents(sub.ID)
		<-done
	}
}

func formatReasoningEvent(event agent.Event, mode reasoningMode) string {
	switch payload := event.Payload.(type) {
	case agent.ReasoningStepPayload:
		if mode == reasoningDebug {
			return debugReasoningLine(event.Kind.String(), event.Meta, payload)
		}
		return "[reasoning] " + payload.Step + ": " + payload.Summary
	case agent.VerificationStartPayload:
		if mode == reasoningDebug {
			return debugReasoningLine(event.Kind.String(), event.Meta, payload)
		}
		return "[verify] " + payload.Command + " (" + payload.Reason + ")"
	case agent.VerificationEndPayload:
		if mode == reasoningDebug {
			return debugReasoningLine(event.Kind.String(), event.Meta, payload)
		}
		status := "passed"
		if payload.IsError {
			status = "failed"
		}
		return fmt.Sprintf("[verify] %s %s in %s", payload.Command, status, payload.Duration.Round(time.Millisecond))
	}
	return ""
}

func debugReasoningLine(kind string, meta agent.EventMeta, payload any) string {
	body, err := json.Marshal(map[string]any{
		"kind":       kind,
		"turn_id":    meta.TurnID,
		"session":    meta.SessionKey,
		"iteration":  meta.Iteration,
		"trace_path": meta.TracePath,
		"payload":    payload,
	})
	if err != nil {
		return "[reasoning:debug] " + kind
	}
	return "[reasoning:debug] " + string(body)
}

func resolveTerminalSessionKey(explicit string) string {
	if strings.TrimSpace(explicit) != "" {
		return strings.TrimSpace(explicit)
	}
	if remembered := readLastTerminalSession(); remembered != "" {
		return remembered
	}
	return "cli:default"
}

func resolveAgentEmoji(cfg *config.Config) string {
	if cfg == nil {
		return internal.Logo
	}

	workspace := strings.TrimSpace(cfg.WorkspacePath())
	if workspace == "" {
		return internal.Logo
	}

	emoji := strings.TrimSpace(commands.ReadAgentSignatureEmoji(workspace))
	if emoji == "" {
		return internal.Logo
	}
	return emoji
}

func setTerminalAgentTitle(agentEmoji, sessionKey string) {
	title := fmt.Sprintf("%s JameClaw Agent", strings.TrimSpace(agentEmoji))
	if strings.TrimSpace(sessionKey) != "" {
		title += " - " + strings.TrimSpace(sessionKey)
	}
	title = strings.Map(func(r rune) rune {
		switch r {
		case '\x1b', '\a':
			return -1
		default:
			return r
		}
	}, title)
	fmt.Printf("\x1b]0;%s\a", title)
}

func interactiveMode(agentLoop *agent.AgentLoop, sessionKey, agentEmoji string, mode reasoningMode) {
	cleanup := streamReasoningEvents(agentLoop, sessionKey, mode)
	defer cleanup()
	prompt := fmt.Sprintf("%s You: ", agentEmoji)

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          prompt,
		HistoryFile:     filepath.Join(os.TempDir(), ".jameclaw_history"),
		HistoryLimit:    100,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		fmt.Printf("Error initializing readline: %v\n", err)
		fmt.Println("Falling back to simple input mode...")
		simpleInteractiveMode(agentLoop, sessionKey, agentEmoji)
		return
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt || err == io.EOF {
				fmt.Println("\nGoodbye!")
				return
			}
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		if input == "exit" || input == "quit" {
			fmt.Println("Goodbye!")
			return
		}

		ctx := context.Background()
		response, err := agentLoop.ProcessDirect(ctx, input, sessionKey)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("\n%s %s\n\n", agentEmoji, response)
	}
}

func simpleInteractiveMode(agentLoop *agent.AgentLoop, sessionKey, agentEmoji string) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(fmt.Sprintf("%s You: ", agentEmoji))
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("\nGoodbye!")
				return
			}
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		if input == "exit" || input == "quit" {
			fmt.Println("Goodbye!")
			return
		}

		ctx := context.Background()
		response, err := agentLoop.ProcessDirect(ctx, input, sessionKey)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("\n%s %s\n\n", agentEmoji, response)
	}
}
