package commands

import (
	"context"
	"fmt"
)

func startCommand() Definition {
	return Definition{
		Name:        "start",
		Description: "Start the bot",
		Usage:       "/start",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			emoji := defaultAgentSignatureEmoji
			name := defaultAgentDisplayName
			if workspace := runtimeWorkspace(rt); workspace != "" {
				emoji = ReadAgentSignatureEmoji(workspace)
				name = ReadAgentDisplayName(workspace)
			}
			return req.Reply(fmt.Sprintf("Hello! I am %s %s", name, emoji))
		},
	}
}
