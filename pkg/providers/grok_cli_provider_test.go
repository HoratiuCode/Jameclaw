package providers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseGrokCLIResponse(t *testing.T) {
	response, err := parseGrokCLIResponse(`{"result":{"content":"Hello from Grok"}}`)
	if err != nil {
		t.Fatalf("parseGrokCLIResponse returned error: %v", err)
	}
	if response.Content != "Hello from Grok" || response.FinishReason != "stop" {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestResolveGrokCLICommandPrefersLocalInstall(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(grokCLIPathEnv, "")
	path := filepath.Join(home, ".local", "bin", "grok")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := resolveGrokCLICommand(); got != path {
		t.Fatalf("resolveGrokCLICommand() = %q, want %q", got, path)
	}
}
