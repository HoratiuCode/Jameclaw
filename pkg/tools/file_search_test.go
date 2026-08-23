package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindFilesToolFindsWorkspaceRelativeFiles(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, "src", "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(workspace, "node_modules", "package"), 0o755); err != nil {
		t.Fatal(err)
	}
	for path, content := range map[string]string{
		"src/main.go":                  "package main",
		"src/nested/main_test.go":      "package main",
		"node_modules/package/main.go": "generated",
	} {
		fullPath := filepath.Join(workspace, filepath.FromSlash(path))
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	tool := NewFindFilesTool(workspace, true)
	result := tool.Execute(context.Background(), map[string]any{
		"query": "main.go",
		"path":  "src",
	})
	if result.IsError {
		t.Fatalf("unexpected search error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "src/main.go") || strings.Contains(result.ForLLM, "node_modules") {
		t.Fatalf("unexpected search result: %s", result.ForLLM)
	}
}

func TestFindFilesToolSupportsWindowsSeparatorsAndKinds(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, "Documents", "Reports"), 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(workspace, "Documents", "Reports", "Q3-Report.TXT")
	if err := os.WriteFile(file, []byte("report"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := NewFindFilesTool(workspace, true)
	result := tool.Execute(context.Background(), map[string]any{
		"query": "documents\\reports\\q3-report.txt",
		"kind":  "file",
	})
	if result.IsError || !strings.Contains(result.ForLLM, "Documents/Reports/Q3-Report.TXT") {
		t.Fatalf("Windows-style search did not find the file: %s", result.ForLLM)
	}

	directoryResult := tool.Execute(context.Background(), map[string]any{
		"query": "reports",
		"kind":  "directory",
	})
	if directoryResult.IsError || !strings.Contains(directoryResult.ForLLM, "Documents/Reports") {
		t.Fatalf("directory search did not find the folder: %s", directoryResult.ForLLM)
	}
}

func TestFindFilesToolRejectsOutsideWorkspace(t *testing.T) {
	workspace := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := NewFindFilesTool(workspace, true)
	result := tool.Execute(context.Background(), map[string]any{
		"query": "secret",
		"path":  outside,
	})
	if !result.IsError || !strings.Contains(result.ForLLM, "outside the workspace") {
		t.Fatalf("expected outside-workspace rejection, got: %s", result.ForLLM)
	}
}
