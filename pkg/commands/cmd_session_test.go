package commands

import (
	"context"
	"strings"
	"testing"
)

func TestSessionManagementCommands(t *testing.T) {
	rt := &Runtime{
		SessionStats:    func() (int, int, int, string, error) { return 12, 345, 4096, "summary", nil },
		UndoLastTurn:    func() (int, error) { return 3, nil },
		CompressSession: func() (int, int, bool, error) { return 8, 4, true, nil },
	}
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	cases := []struct {
		command string
		want    string
	}{
		{command: "/stats", want: "Session: 12 messages"},
		{command: "/undo", want: "Removed the last turn (3 messages)."},
		{command: "/compact", want: "Compacted the chat: removed 8 older messages, 4 remain."},
	}
	for _, tc := range cases {
		t.Run(strings.TrimPrefix(tc.command, "/"), func(t *testing.T) {
			var reply string
			result := ex.Execute(context.Background(), Request{Text: tc.command, Reply: func(text string) error {
				reply = text
				return nil
			}})
			if result.Outcome != OutcomeHandled {
				t.Fatalf("outcome=%v, want handled", result.Outcome)
			}
			if !strings.Contains(reply, tc.want) {
				t.Fatalf("reply=%q, want %q", reply, tc.want)
			}
		})
	}
}

func TestSessionManagementCommandsUnavailableWithoutRuntime(t *testing.T) {
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), &Runtime{})
	for _, command := range []string{"/stats", "/undo", "/compact"} {
		var reply string
		result := ex.Execute(context.Background(), Request{Text: command, Reply: func(text string) error {
			reply = text
			return nil
		}})
		if result.Outcome != OutcomeHandled || reply != unavailableMsg {
			t.Fatalf("%s: outcome=%v reply=%q", command, result.Outcome, reply)
		}
	}
}
