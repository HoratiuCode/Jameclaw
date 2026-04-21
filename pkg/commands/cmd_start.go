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
			if workspace := runtimeWorkspace(rt); workspace != "" {
				emoji = ReadAgentSignatureEmoji(workspace)
			}
			return req.Reply(fmt.Sprintf("Hello! I am JameClaw %s", emoji))
		},
	}
}
