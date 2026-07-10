package commands

import (
	"context"
	"fmt"
	"strings"
)

func nameCommand() Definition {
	return Definition{
		Name:        "name",
		Description: "Change the assistant default name",
		Usage:       "/name [name|show]",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			workspace := runtimeWorkspace(rt)
			if workspace == "" {
				return req.Reply(unavailableMsg)
			}

			args := strings.TrimSpace(req.Text)
			args = strings.TrimSpace(strings.TrimPrefix(args, "/name"))
			if args == "" || strings.EqualFold(args, "show") {
				return req.Reply(fmt.Sprintf("Current assistant name: %s", ReadAgentDisplayName(workspace)))
			}

			if err := UpdateAgentDisplayName(workspace, args); err != nil {
				return req.Reply(fmt.Sprintf("Failed to update assistant name: %v", err))
			}
			return req.Reply(fmt.Sprintf("Updated assistant name to %s.", args))
		},
	}
}
