package api

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCachedLocalFileIndexReusesSessionIndex(t *testing.T) {
	dir := t.TempDir()
	firstPath := filepath.Join(dir, "first.txt")
	if err := os.WriteFile(firstPath, []byte("first"), 0o644); err != nil {
		t.Fatalf("WriteFile(first) error = %v", err)
	}

	handler := NewHandler("")
	first := handler.cachedLocalFileIndex([]string{dir})
	if len(first) != 1 || first[0].Name != "first.txt" {
		t.Fatalf("first index = %#v, want first.txt", first)
	}

	secondPath := filepath.Join(dir, "second.txt")
	if err := os.WriteFile(secondPath, []byte("second"), 0o644); err != nil {
		t.Fatalf("WriteFile(second) error = %v", err)
	}

	second := handler.cachedLocalFileIndex([]string{dir})
	if len(second) != 1 || second[0].Name != "first.txt" {
		t.Fatalf("cached index = %#v, want original first.txt only", second)
	}
}

func TestSearchIndexedLocalFilesFiltersWithoutWalking(t *testing.T) {
	items := []fileSearchItem{
		{Name: "notes.md", Path: "/tmp/workspace/notes.md", Kind: "file"},
		{Name: "photo.png", Path: "/tmp/downloads/photo.png", Kind: "file"},
	}

	results := searchIndexedLocalFiles(items, "notes", 10)
	if len(results) != 1 || results[0].Name != "notes.md" {
		t.Fatalf("results = %#v, want notes.md", results)
	}
}
