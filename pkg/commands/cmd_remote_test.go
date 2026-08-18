package commands

import (
	"context"
	"strings"
	"testing"
)

func TestRemoteStatusAndAutomationCommands(t *testing.T) {
	rt := &Runtime{
		GetModelInfo:       func() (string, string) { return "model-x", "provider-y" },
		GetEnabledChannels: func() []string { return []string{"telegram", "jame"} },
		GetActiveTurn:      func() any { return nil },
		ListAutomations: func() []AutomationSummary {
			return []AutomationSummary{{ID: "job-1", Name: "Daily digest", Enabled: true, Schedule: "0 9 * * *"}}
		},
		RunAutomation: func(identifier string) error {
			if identifier != "Daily digest" {
				t.Fatalf("run identifier=%q", identifier)
			}
			return nil
		},
		SetAutomationState: func(identifier string, enabled bool) error {
			if identifier != "job-1" || enabled {
				t.Fatalf("state=%q enabled=%v", identifier, enabled)
			}
			return nil
		},
	}
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	for _, tc := range []struct{ command, want string }{
		{"/status", "Model: model-x (provider-y)"},
		{"/automations", "Daily digest"},
		{"/run Daily digest", "Automation run: Daily digest."},
		{"/pause job-1", "Automation pause: job-1."},
	} {
		t.Run(tc.command, func(t *testing.T) {
			var reply string
			result := ex.Execute(context.Background(), Request{Channel: "telegram", Text: tc.command, Reply: func(text string) error { reply = text; return nil }})
			if result.Outcome != OutcomeHandled || !strings.Contains(reply, tc.want) {
				t.Fatalf("outcome=%v reply=%q want=%q", result.Outcome, reply, tc.want)
			}
		})
	}
}
