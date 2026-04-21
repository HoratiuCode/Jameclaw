package commands

import (
	"context"
	"fmt"
	"strings"
)

func personaCommand() Definition {
	return Definition{
		Name:        "persona",
		Aliases:     []string{"personality"},
		Description: "Change the assistant personality",
		Usage:       "/persona <text>",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			workspace := runtimeWorkspace(rt)
			if workspace == "" {
				return req.Reply(unavailableMsg)
			}

			args := strings.TrimSpace(req.Text)
			args = strings.TrimSpace(strings.TrimPrefix(args, "/persona"))
			args = strings.TrimSpace(strings.TrimPrefix(args, "/personality"))
			if args == "" || strings.EqualFold(args, "show") {
				persona := ReadAgentPersona(workspace)
				if persona == "" {
					return req.Reply("No custom personality is set.")
				}
				return req.Reply("Current personality:\n" + persona)
			}

			if err := UpdateAgentPersona(workspace, args); err != nil {
				return req.Reply(fmt.Sprintf("Failed to update personality: %v", err))
			}
			return req.Reply("Updated personality.")
		},
	}
}
