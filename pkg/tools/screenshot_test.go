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
	tool.runner = func(ctx context.Context, req screenshotRequest) error {
		return os.WriteFile(req.OutputPath, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, 0o600)
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
	tool.runner = func(ctx context.Context, req screenshotRequest) error {
		return os.WriteFile(req.OutputPath, []byte("png"), 0o600)
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

func TestScreenshotTool_TargetedRequest(t *testing.T) {
	store := media.NewFileMediaStore()
	tool := NewScreenshotTool(0, store)
	var captured screenshotRequest
	tool.runner = func(ctx context.Context, req screenshotRequest) error {
		captured = req
		return os.WriteFile(req.OutputPath, []byte("png"), 0o600)
	}

	ctx := WithToolContext(context.Background(), "telegram", "chat123")
	result := tool.Execute(ctx, map[string]any{
		"app":          "Safari",
		"window_title": "Release Notes",
		"window_index": float64(2),
		"retina":       true,
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if captured.Mode != "window" {
		t.Fatalf("Mode = %q, want window", captured.Mode)
	}
	if captured.App != "Safari" || captured.WindowTitle != "Release Notes" {
		t.Fatalf("unexpected target: %#v", captured)
	}
	if captured.WindowIndex == nil || *captured.WindowIndex != 2 {
		t.Fatalf("WindowIndex = %#v, want 2", captured.WindowIndex)
	}
	if !captured.Retina {
		t.Fatal("Retina = false, want true")
	}
}

func TestScreenshotTool_EmptyCapture(t *testing.T) {
	store := media.NewFileMediaStore()
	tool := NewScreenshotTool(0, store)
	tool.runner = func(ctx context.Context, req screenshotRequest) error {
		return os.WriteFile(req.OutputPath, nil, 0o600)
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

func TestScreenRecordingTool_Success(t *testing.T) {
	t.Setenv(envJameclawScreenBinary, "/no/such/jameclaw-screen")
	store := media.NewFileMediaStore()
	tool := NewScreenRecordingTool(0, store)
	tool.now = func() time.Time { return time.Date(2026, 7, 10, 12, 34, 56, 0, time.UTC) }
	var captured recordingRequest
	tool.runner = func(ctx context.Context, req recordingRequest) error {
		captured = req
		return os.WriteFile(req.OutputPath, []byte("video"), 0o600)
	}

	ctx := WithToolContext(context.Background(), "telegram", "chat123")
	result := tool.Execute(ctx, map[string]any{
		"duration_seconds": float64(3),
		"region":           "100,120,640,360",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if len(result.Media) != 1 {
		t.Fatalf("expected 1 media ref, got %d", len(result.Media))
	}
	if captured.DurationSeconds != 3 {
		t.Fatalf("DurationSeconds = %d, want 3", captured.DurationSeconds)
	}
	if captured.Mode != "area" || captured.Region != "100,120,640,360" {
		t.Fatalf("unexpected recording request: %#v", captured)
	}

	_, meta, err := store.ResolveWithMeta(result.Media[0])
	if err != nil {
		t.Fatalf("ResolveWithMeta failed: %v", err)
	}
	if meta.Filename != "recording-20260710-123456.mov" {
		t.Fatalf("Filename = %q, want default recording filename", meta.Filename)
	}
	if meta.Source != "tool:screen_recording" {
		t.Fatalf("Source = %q, want tool:screen_recording", meta.Source)
	}
}

func TestScreenRecordingTool_DurationClamped(t *testing.T) {
	store := media.NewFileMediaStore()
	tool := NewScreenRecordingTool(0, store)
	var captured recordingRequest
	tool.runner = func(ctx context.Context, req recordingRequest) error {
		captured = req
		return os.WriteFile(req.OutputPath, []byte("video"), 0o600)
	}

	ctx := WithToolContext(context.Background(), "telegram", "chat123")
	result := tool.Execute(ctx, map[string]any{"duration_seconds": float64(999)})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if captured.DurationSeconds != 180 {
		t.Fatalf("DurationSeconds = %d, want clamp to 180", captured.DurationSeconds)
	}
}

func TestScreenshotTool_FileTooLarge(t *testing.T) {
	store := media.NewFileMediaStore()
	tool := NewScreenshotTool(2, store)
	tool.runner = func(ctx context.Context, req screenshotRequest) error {
		return os.WriteFile(req.OutputPath, []byte("large"), 0o600)
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
