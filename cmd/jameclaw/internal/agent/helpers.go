package agent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ergochat/readline"

	"github.com/sipeed/jameclaw/cmd/jameclaw/internal"
	"github.com/sipeed/jameclaw/pkg/agent"
	"github.com/sipeed/jameclaw/pkg/bus"
	"github.com/sipeed/jameclaw/pkg/commands"
	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/pkg/logger"
	"github.com/sipeed/jameclaw/pkg/providers"
)

func agentCmd(message, sessionKey, model string, debug bool) error {
	if sessionKey == "" {
		sessionKey = "cli:default"
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
		ctx := context.Background()
		response, err := agentLoop.ProcessDirect(ctx, message, sessionKey)
		if err != nil {
			return fmt.Errorf("error processing message: %w", err)
		}
		fmt.Printf("\n%s %s\n", agentEmoji, response)
		return nil
	}

	fmt.Printf("%s Interactive mode (Ctrl+C to exit)\n\n", agentEmoji)
	fmt.Println("Tip: use /emoji <emoji>, /persona <text>, or /skills add <skill> to customize JameClaw.")
	interactiveMode(agentLoop, sessionKey, agentEmoji)

	return nil
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

func interactiveMode(agentLoop *agent.AgentLoop, sessionKey, agentEmoji string) {
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
