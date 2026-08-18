package commands

import (
	"context"
	"fmt"
	"strings"
)

func stopCommand() Definition {
	return Definition{
		Name:        "stop",
		Aliases:     []string{"abort", "cancel"},
		Description: "Stop the active agent task immediately",
		Usage:       "/stop",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.StopAgent == nil {
				return req.Reply(unavailableMsg)
			}
			if err := rt.StopAgent(true); err != nil {
				return req.Reply("No active task to stop.")
			}
			return req.Reply("Stopped the active task. The unfinished turn was rolled back.")
		},
	}
}

func queueCommand() Definition {
	return Definition{
		Name:        "queue",
		Description: "Show messages waiting for the agent",
		Usage:       "/queue [clear]",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.PendingQueue == nil {
				return req.Reply(unavailableMsg)
			}
			fields := strings.Fields(req.Text)
			if len(fields) > 1 && strings.EqualFold(fields[1], "clear") {
				if rt.ClearQueue == nil {
					return req.Reply(unavailableMsg)
				}
				return req.Reply(fmt.Sprintf("Cleared %d queued message(s).", rt.ClearQueue()))
			}
			depth := rt.PendingQueue()
			if depth == 0 {
				return req.Reply("The agent queue is empty.")
			}
			return req.Reply(fmt.Sprintf("The agent has %d message(s) queued. They will run after the current task.", depth))
		},
	}
}

func modelCommand() Definition {
	return Definition{
		Name:        "model",
		Description: "Show or switch the active model",
		Usage:       "/model [name]",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.GetModelInfo == nil {
				return req.Reply(unavailableMsg)
			}
			fields := strings.Fields(req.Text)
			if len(fields) == 1 {
				name, provider := rt.GetModelInfo()
				return req.Reply(fmt.Sprintf("Active model: %s (%s)\nUse /model <name> to switch.", name, provider))
			}
			if rt.SwitchModel == nil {
				return req.Reply(unavailableMsg)
			}
			value := strings.Join(fields[1:], " ")
			old, err := rt.SwitchModel(value)
			if err != nil {
				return req.Reply(err.Error())
			}
			return req.Reply(fmt.Sprintf("Switched model from %s to %s.", old, value))
		},
	}
}

func newSessionCommand() Definition {
	return Definition{
		Name:        "new",
		Aliases:     []string{"reset"},
		Description: "Start a fresh conversation in this channel",
		Usage:       "/new",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.ClearHistory == nil {
				return req.Reply(unavailableMsg)
			}
			if err := rt.ClearHistory(); err != nil {
				return req.Reply("Failed to start a new conversation: " + err.Error())
			}
			return req.Reply("Started a new conversation. The previous context is no longer active.")
		},
	}
}

func sessionsCommand() Definition {
	return Definition{
		Name:        "sessions",
		Description: "Show the current persistent conversation session",
		Usage:       "/sessions",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.SessionStats == nil {
				return req.Reply(unavailableMsg)
			}
			messages, _, _, _, err := rt.SessionStats()
			if err != nil {
				return req.Reply("Failed to read sessions: " + err.Error())
			}
			return req.Reply(fmt.Sprintf("Current session: %s\nMessages: %d\nUse /new to start a fresh session, or open History in the app to resume another one.", req.ChatID, messages))
		},
	}
}

func approvalsCommand() Definition {
	return Definition{
		Name:        "approvals",
		Description: "Explain the current safety boundary",
		Usage:       "/approvals",
		Handler: func(_ context.Context, req Request, _ *Runtime) error {
			return req.Reply("Safety controls are active. Destructive tool actions, external delivery, and sensitive integrations require their configured approval boundary. Review the action before confirming it.")
		},
	}
}

func usageCommand() Definition {
	return Definition{
		Name:        "usage",
		Description: "Show context and usage for this conversation",
		Usage:       "/usage",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.SessionStats == nil {
				return req.Reply(unavailableMsg)
			}
			messages, tokens, window, summary, err := rt.SessionStats()
			if err != nil {
				return req.Reply("Failed to read usage: " + err.Error())
			}
			percent := 0
			if window > 0 {
				percent = tokens * 100 / window
			}
			result := fmt.Sprintf("Usage\nMessages: %d\nEstimated context: %d / %d tokens (%d%%)", messages, tokens, window, percent)
			if summary != "" {
				result += "\nA compacted summary is active."
			}
			return req.Reply(result)
		},
	}
}

func insightsCommand() Definition {
	return Definition{
		Name:        "insights",
		Description: "Summarize the current conversation state",
		Usage:       "/insights",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.SessionStats == nil {
				return req.Reply(unavailableMsg)
			}
			messages, tokens, window, summary, err := rt.SessionStats()
			if err != nil {
				return req.Reply("Failed to read insights: " + err.Error())
			}
			state := "healthy"
			if window > 0 && tokens*100/window >= 80 {
				state = "near context limit — use /compact"
			}
			return req.Reply(fmt.Sprintf("Conversation insight\nState: %s\nMessages: %d\nContext estimate: %d / %d tokens\nSummary: %s", state, messages, tokens, window, map[bool]string{true: "active", false: "not needed"}[summary != ""]))
		},
	}
}
