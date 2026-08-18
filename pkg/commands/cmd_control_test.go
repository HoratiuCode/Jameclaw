package commands

import (
	"context"
	"strings"
	"testing"
)

func TestControlCommands_StopAndQueue(t *testing.T) {
	stopped := false
	cleared := false
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), &Runtime{
		StopAgent:    func(bool) error { stopped = true; return nil },
		PendingQueue: func() int { return 2 },
		ClearQueue:   func() int { cleared = true; return 2 },
	})

	var reply string
	result := ex.Execute(context.Background(), Request{Text: "/stop", Reply: func(value string) error { reply = value; return nil }})
	if result.Outcome != OutcomeHandled || !stopped || !strings.Contains(reply, "Stopped") {
		t.Fatalf("stop result=%+v stopped=%v reply=%q", result, stopped, reply)
	}

	reply = ""
	result = ex.Execute(context.Background(), Request{Text: "/queue", Reply: func(value string) error { reply = value; return nil }})
	if result.Outcome != OutcomeHandled || !strings.Contains(reply, "2 message") {
		t.Fatalf("queue result=%+v reply=%q", result, reply)
	}

	reply = ""
	result = ex.Execute(context.Background(), Request{Text: "/queue clear", Reply: func(value string) error { reply = value; return nil }})
	if result.Outcome != OutcomeHandled || !cleared || !strings.Contains(reply, "Cleared 2") {
		t.Fatalf("queue clear result=%+v cleared=%v reply=%q", result, cleared, reply)
	}
}

func TestControlCommands_ModelUsageAndInsights(t *testing.T) {
	model := "old-model"
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), &Runtime{
		GetModelInfo: func() (string, string) { return model, "test-provider" },
		SwitchModel:  func(value string) (string, error) { old := model; model = value; return old, nil },
		SessionStats: func() (int, int, int, string, error) { return 4, 800, 1000, "summary", nil },
	})

	for _, command := range []string{"/model new-model", "/usage", "/insights"} {
		var reply string
		result := ex.Execute(context.Background(), Request{Text: command, Reply: func(value string) error { reply = value; return nil }})
		if result.Outcome != OutcomeHandled || reply == "" {
			t.Fatalf("command=%q result=%+v reply=%q", command, result, reply)
		}
	}
	if model != "new-model" {
		t.Fatalf("model=%q", model)
	}
}
