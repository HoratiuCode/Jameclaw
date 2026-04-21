package commands

import (
	"context"
	"fmt"
	"strings"
)

func emojiCommand() Definition {
	return Definition{
		Name:        "emoji",
		Description: "Change the assistant signature emoji",
		Usage:       "/emoji [emoji|show]",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			workspace := runtimeWorkspace(rt)
			if workspace == "" {
				return req.Reply(unavailableMsg)
			}

			fields := strings.Fields(strings.TrimSpace(req.Text))
			if len(fields) <= 1 || strings.EqualFold(fields[1], "show") {
				return req.Reply(fmt.Sprintf("Current signature emoji: %s", ReadAgentSignatureEmoji(workspace)))
			}

			emoji := strings.TrimSpace(strings.Join(fields[1:], " "))
			if emoji == "" {
				emoji = defaultAgentSignatureEmoji
			}

			if err := UpdateAgentSignatureEmoji(workspace, emoji); err != nil {
				return req.Reply(fmt.Sprintf("Failed to update signature emoji: %v", err))
			}
			return req.Reply(fmt.Sprintf("Updated signature emoji to %s.", emoji))
		},
	}
}
