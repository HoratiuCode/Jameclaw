package commands

import (
	"context"
	"fmt"
	"strings"
)

func statusCommand() Definition {
	return Definition{
		Name:        "status",
		Description: "Show agent, model, channel, and task status",
		Usage:       "/status",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil {
				return req.Reply(unavailableMsg)
			}
			lines := []string{"JameClaw status", fmt.Sprintf("Channel: %s", req.Channel)}
			if rt.GetModelInfo != nil {
				model, provider := rt.GetModelInfo()
				lines = append(lines, fmt.Sprintf("Model: %s (%s)", model, provider))
			}
			if rt.GetEnabledChannels != nil {
				channels := rt.GetEnabledChannels()
				if len(channels) == 0 {
					lines = append(lines, "Connected channels: none")
				} else {
					lines = append(lines, "Connected channels: "+strings.Join(channels, ", "))
				}
			}
			if rt.GetActiveTurn != nil && rt.GetActiveTurn() != nil {
				lines = append(lines, "Task: working")
			} else {
				lines = append(lines, "Task: ready")
			}
			return req.Reply(strings.Join(lines, "\n"))
		},
	}
}

func automationsCommand() Definition {
	return Definition{
		Name:        "automations",
		Aliases:     []string{"automation"},
		Description: "List scheduled automations",
		Usage:       "/automations",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.ListAutomations == nil {
				return req.Reply(unavailableMsg)
			}
			items := rt.ListAutomations()
			if len(items) == 0 {
				return req.Reply("No automations scheduled.")
			}
			lines := []string{"Automations:"}
			for _, item := range items {
				state := "paused"
				if item.Running {
					state = "running"
				} else if item.Enabled {
					state = "scheduled"
				}
				lines = append(lines, fmt.Sprintf("• %s — %s [%s] (%s)", item.Name, item.ID, state, item.Schedule))
			}
			lines = append(lines, "Use /run <name-or-id>, /pause <name-or-id>, or /resume <name-or-id>.")
			return req.Reply(strings.Join(lines, "\n"))
		},
	}
}

func automationActionCommand(name, description, verb string, enabled *bool) Definition {
	return Definition{
		Name:        name,
		Description: description,
		Usage:       "/" + name + " <name-or-id>",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			fields := strings.Fields(strings.TrimSpace(req.Text))
			identifier := ""
			if len(fields) > 1 {
				identifier = strings.Join(fields[1:], " ")
			}
			if identifier == "" {
				return req.Reply("Usage: /" + name + " <name-or-id>")
			}
			if rt == nil {
				return req.Reply(unavailableMsg)
			}
			var err error
			if enabled == nil {
				if rt.RunAutomation == nil {
					return req.Reply(unavailableMsg)
				}
				err = rt.RunAutomation(identifier)
			} else {
				if rt.SetAutomationState == nil {
					return req.Reply(unavailableMsg)
				}
				err = rt.SetAutomationState(identifier, *enabled)
			}
			if err != nil {
				return req.Reply(fmt.Sprintf("Could not %s automation: %v", verb, err))
			}
			return req.Reply(fmt.Sprintf("Automation %s: %s.", verb, identifier))
		},
	}
}
