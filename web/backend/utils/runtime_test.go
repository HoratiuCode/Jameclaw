package utils

import (
	"reflect"
	"testing"
)

func TestOpenBrowserCommandDarwinBackground(t *testing.T) {
	command, args, err := openBrowserCommand("darwin", "http://localhost:18800", true)
	if err != nil {
		t.Fatalf("openBrowserCommand() error = %v", err)
	}
	if command != "open" {
		t.Fatalf("command = %q, want open", command)
	}
	if !reflect.DeepEqual(args, []string{"-g", "http://localhost:18800"}) {
		t.Fatalf("args = %#v", args)
	}
}

func TestOpenBrowserCommandDarwinForeground(t *testing.T) {
	command, args, err := openBrowserCommand("darwin", "http://localhost:18800", false)
	if err != nil {
		t.Fatalf("openBrowserCommand() error = %v", err)
	}
	if command != "open" {
		t.Fatalf("command = %q, want open", command)
	}
	if !reflect.DeepEqual(args, []string{"http://localhost:18800"}) {
		t.Fatalf("args = %#v", args)
	}
}
