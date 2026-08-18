package commands

import (
	"context"
	"fmt"
)

func sessionStatsCommand() Definition {
	return Definition{
		Name:        "stats",
		Description: "Show session context and token usage",
		Usage:       "/stats",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.SessionStats == nil {
				return req.Reply(unavailableMsg)
			}
			messages, tokens, window, summary, err := rt.SessionStats()
			if err != nil {
				return req.Reply("Failed to read session stats: " + err.Error())
			}
			reply := fmt.Sprintf("Session: %d messages\nContext: ~%d / %d tokens", messages, tokens, window)
			if summary != "" {
				reply += "\nA compacted summary is active."
			}
			return req.Reply(reply)
		},
	}
}

func undoCommand() Definition {
	return Definition{
		Name:        "undo",
		Description: "Remove the last user turn from this chat",
		Usage:       "/undo",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.UndoLastTurn == nil {
				return req.Reply(unavailableMsg)
			}
			removed, err := rt.UndoLastTurn()
			if err != nil {
				return req.Reply("Failed to undo the last turn: " + err.Error())
			}
			if removed == 0 {
				return req.Reply("Nothing to undo in this chat.")
			}
			return req.Reply(fmt.Sprintf("Removed the last turn (%d messages).", removed))
		},
	}
}

func compactCommand() Definition {
	return Definition{
		Name:        "compact",
		Description: "Compact older context to make room for new work",
		Usage:       "/compact",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.CompressSession == nil {
				return req.Reply(unavailableMsg)
			}
			dropped, remaining, compressed, err := rt.CompressSession()
			if err != nil {
				return req.Reply("Failed to compact this chat: " + err.Error())
			}
			if !compressed {
				return req.Reply("This chat does not need compacting yet.")
			}
			return req.Reply(fmt.Sprintf("Compacted the chat: removed %d older messages, %d remain.", dropped, remaining))
		},
	}
}
