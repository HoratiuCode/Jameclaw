package browserbridge

import (
	"context"
	"testing"
	"time"
)

func TestBridgeDispatchesAndReturnsExtensionResult(t *testing.T) {
	bridge := New()
	commands := make(chan *Command, 1)
	go func() {
		command, err := bridge.Next(context.Background(), "tab-1")
		if err == nil {
			commands <- command
		}
	}()
	time.Sleep(10 * time.Millisecond)
	resultCh := make(chan Result, 1)
	go func() {
		result, err := bridge.Dispatch(context.Background(), "inspect", map[string]any{})
		if err == nil {
			resultCh <- result
		}
	}()
	command := <-commands
	if command.Action != "inspect" || command.ID == "" {
		t.Fatalf("command = %#v", command)
	}
	bridge.Complete(command.ID, Result{Content: "page details"})
	select {
	case result := <-resultCh:
		if result.Content != "page details" {
			t.Fatalf("result = %#v", result)
		}
	case <-time.After(time.Second):
		t.Fatal("dispatch did not complete")
	}
}
