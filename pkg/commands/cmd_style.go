package commands

import (
	"context"
	"fmt"
	"strings"
)

func styleCommand() Definition {
	return Definition{
		Name:        "style",
		Aliases:     []string{"voice"},
		Description: "Change the assistant speaking style memory",
		Usage:       "/style [text|show]",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			workspace := runtimeWorkspace(rt)
			if workspace == "" {
				return req.Reply(unavailableMsg)
			}

			args := strings.TrimSpace(req.Text)
			args = strings.TrimSpace(strings.TrimPrefix(args, "/style"))
			args = strings.TrimSpace(strings.TrimPrefix(args, "/voice"))
			if args == "" || strings.EqualFold(args, "show") {
				style := ReadAgentStyle(workspace)
				if style == "" {
					return req.Reply("No custom speaking style is set.")
				}
				return req.Reply("Current speaking style:\n" + style)
			}

			if err := UpdateAgentStyle(workspace, args); err != nil {
				return req.Reply(fmt.Sprintf("Failed to update speaking style: %v", err))
			}
			return req.Reply("Updated speaking style.")
		},
	}
}
