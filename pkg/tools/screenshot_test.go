package tools

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/jameclaw/pkg/media"
)

func TestScreenshotTool_NoContext(t *testing.T) {
	store := media.NewFileMediaStore()
	tool := NewScreenshotTool(0, store)

	result := tool.Execute(context.Background(), map[string]any{})
	if !result.IsError {
		t.Fatal("expected error when no channel context")
	}
}

func TestScreenshotTool_NoMediaStore(t *testing.T) {
	tool := NewScreenshotTool(0, nil)
	ctx := WithToolContext(context.Background(), "telegram", "chat123")

	result := tool.Execute(ctx, map[string]any{})
	if !result.IsError {
		t.Fatal("expected error when no media store")
	}
}

func TestScreenshotTool_Success(t *testing.T) {
	store := media.NewFileMediaStore()
	tool := NewScreenshotTool(0, store)
	tool.now = func() time.Time { return time.Date(2026, 7, 10, 12, 34, 56, 0, time.UTC) }
	tool.runner = func(ctx context.Context, outputPath string) error {
		return os.WriteFile(outputPath, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, 0o600)
	}

	ctx := WithToolContext(context.Background(), "telegram", "chat123")
	result := tool.Execute(ctx, map[string]any{})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if len(result.Media) != 1 {
		t.Fatalf("expected 1 media ref, got %d", len(result.Media))
	}

	path, meta, err := store.ResolveWithMeta(result.Media[0])
	if err != nil {
		t.Fatalf("ResolveWithMeta failed: %v", err)
	}
	if path == "" {
		t.Fatalf("expected stored screenshot path, got %q", path)
	}
	if meta.Filename != "screenshot-20260710-123456.png" {
		t.Fatalf("Filename = %q, want default timestamp filename", meta.Filename)
	}
	if meta.ContentType != "image/png" {
		t.Fatalf("ContentType = %q, want image/png", meta.ContentType)
	}
	if meta.Source != "tool:screenshot" {
		t.Fatalf("Source = %q, want tool:screenshot", meta.Source)
	}
	if meta.CleanupPolicy != media.CleanupPolicyDeleteOnCleanup {
		t.Fatalf("CleanupPolicy = %q, want %q", meta.CleanupPolicy, media.CleanupPolicyDeleteOnCleanup)
	}
}

func TestScreenshotTool_CustomFilename(t *testing.T) {
	store := media.NewFileMediaStore()
	tool := NewScreenshotTool(0, store)
	tool.runner = func(ctx context.Context, outputPath string) error {
		return os.WriteFile(outputPath, []byte("png"), 0o600)
	}

	ctx := WithToolContext(context.Background(), "discord", "chat456")
	result := tool.Execute(ctx, map[string]any{"filename": "../screen:now.jpg"})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}

	_, meta, err := store.ResolveWithMeta(result.Media[0])
	if err != nil {
		t.Fatalf("ResolveWithMeta failed: %v", err)
	}
	if meta.Filename != "screen_now.png" {
		t.Fatalf("Filename = %q, want sanitized png filename", meta.Filename)
	}
}

func TestScreenshotTool_EmptyCapture(t *testing.T) {
	store := media.NewFileMediaStore()
	tool := NewScreenshotTool(0, store)
	tool.runner = func(ctx context.Context, outputPath string) error {
		return os.WriteFile(outputPath, nil, 0o600)
	}

	ctx := WithToolContext(context.Background(), "telegram", "chat123")
	result := tool.Execute(ctx, map[string]any{})
	if !result.IsError {
		t.Fatal("expected empty capture error")
	}
	if !strings.Contains(result.ForLLM, "empty") {
		t.Fatalf("expected empty error, got %q", result.ForLLM)
	}
}

func TestScreenshotTool_FileTooLarge(t *testing.T) {
	store := media.NewFileMediaStore()
	tool := NewScreenshotTool(2, store)
	tool.runner = func(ctx context.Context, outputPath string) error {
		return os.WriteFile(outputPath, []byte("large"), 0o600)
	}

	ctx := WithToolContext(context.Background(), "telegram", "chat123")
	result := tool.Execute(ctx, map[string]any{})
	if !result.IsError {
		t.Fatal("expected oversized screenshot error")
	}
	if !strings.Contains(result.ForLLM, "too large") {
		t.Fatalf("expected too large error, got %q", result.ForLLM)
	}
}
