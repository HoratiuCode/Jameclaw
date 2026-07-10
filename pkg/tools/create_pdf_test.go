package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/jameclaw/pkg/media"
)

func TestCreatePDFToolCreatesAndRegistersPDF(t *testing.T) {
	workspace := t.TempDir()
	store := media.NewFileMediaStore()
	tool := NewCreatePDFTool(workspace, true, 0, store)

	ctx := WithToolContext(context.Background(), "jame", "jame:sess")
	result := tool.Execute(ctx, map[string]any{
		"title":    "Quarterly Report",
		"content":  "Revenue increased.\nNext steps are listed here.",
		"filename": "report.pdf",
	})

	if result.IsError {
		t.Fatalf("Execute returned error: %s", result.ForLLM)
	}
	if len(result.Media) != 1 || !strings.HasPrefix(result.Media[0], "media://") {
		t.Fatalf("media refs = %v, want one media:// ref", result.Media)
	}

	path, meta, err := store.ResolveWithMeta(result.Media[0])
	if err != nil {
		t.Fatalf("ResolveWithMeta: %v", err)
	}
	if meta.ContentType != "application/pdf" {
		t.Fatalf("ContentType = %q, want application/pdf", meta.ContentType)
	}
	if meta.Filename != "report.pdf" {
		t.Fatalf("Filename = %q, want report.pdf", meta.Filename)
	}
	if filepath.Dir(path) != filepath.Join(workspace, defaultPDFSubdir) {
		t.Fatalf("path = %q, want under generated dir", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.HasPrefix(string(data), "%PDF-1.4\n") {
		t.Fatal("created file does not look like a PDF")
	}
	if !strings.Contains(string(data), "Revenue increased.") {
		t.Fatal("created PDF does not include content text")
	}
}

func TestCreatePDFToolRequiresChatContext(t *testing.T) {
	tool := NewCreatePDFTool(t.TempDir(), true, 0, media.NewFileMediaStore())
	result := tool.Execute(context.Background(), map[string]any{
		"content": "hello",
	})
	if !result.IsError || !strings.Contains(result.ForLLM, "no target channel/chat") {
		t.Fatalf("result = %+v, want no target channel/chat error", result)
	}
}

func TestCreatePDFToolRejectsOutsideWorkspace(t *testing.T) {
	tool := NewCreatePDFTool(t.TempDir(), true, 0, media.NewFileMediaStore())
	ctx := WithToolContext(context.Background(), "telegram", "telegram:1")

	result := tool.Execute(ctx, map[string]any{
		"content": "hello",
		"path":    "../outside.pdf",
	})
	if !result.IsError || !strings.Contains(result.ForLLM, "outside the workspace") {
		t.Fatalf("result = %+v, want outside workspace error", result)
	}
}

func TestBuildSimplePDFUsesCompactXref(t *testing.T) {
	data := buildSimplePDF("Title", "Body")
	text := string(data)
	if !strings.Contains(text, "xref\n0 6\n") {
		t.Fatalf("xref table was not compact:\n%s", text)
	}
	if strings.Contains(text, "999 0 obj") {
		t.Fatal("PDF should not use sparse object numbers")
	}
}
