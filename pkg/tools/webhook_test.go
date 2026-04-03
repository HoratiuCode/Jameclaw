package tools

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestWebhookPostTool_Success(t *testing.T) {
	withPrivateWebFetchHostsAllowed(t)

	var gotContentType string
	var gotHeader string
	var gotBody map[string]any
	tool, err := NewWebhookPostTool("", nil)
	if err != nil {
		t.Fatalf("NewWebhookPostTool() error: %v", err)
	}
	tool.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotContentType = r.Header.Get("Content-Type")
		gotHeader = r.Header.Get("X-Test")
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		return &http.Response{
			StatusCode: http.StatusAccepted,
			Status:     "202 Accepted",
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
			Header:     make(http.Header),
		}, nil
	})}

	result := tool.Execute(context.Background(), map[string]any{
		"url": "https://example.com/hook",
		"payload": map[string]any{
			"event": "deploy",
		},
		"headers": map[string]any{
			"X-Test": "abc123",
		},
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if gotContentType != "application/json" {
		t.Fatalf("expected application/json content type, got %q", gotContentType)
	}
	if gotHeader != "abc123" {
		t.Fatalf("expected custom header to be forwarded, got %q", gotHeader)
	}
	if gotBody["event"] != "deploy" {
		t.Fatalf("expected payload to arrive, got %#v", gotBody)
	}
	if !strings.Contains(result.ForUser, "202 Accepted") {
		t.Fatalf("expected user summary to include response status, got %q", result.ForUser)
	}
}

func TestWebhookPostTool_RejectsPrivateHosts(t *testing.T) {
	tool, err := NewWebhookPostTool("", nil)
	if err != nil {
		t.Fatalf("NewWebhookPostTool() error: %v", err)
	}

	result := tool.Execute(context.Background(), map[string]any{
		"url": "http://127.0.0.1:8080/hook",
	})
	if !result.IsError {
		t.Fatal("expected private host request to be blocked")
	}
	if !strings.Contains(result.ForLLM, "private or local network hosts") {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
}

func TestWebhookPostTool_HTTPError(t *testing.T) {
	withPrivateWebFetchHostsAllowed(t)

	tool, err := NewWebhookPostTool("", nil)
	if err != nil {
		t.Fatalf("NewWebhookPostTool() error: %v", err)
	}
	tool.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Status:     "400 Bad Request",
			Body:       io.NopCloser(strings.NewReader("bad webhook\n")),
			Header:     make(http.Header),
		}, nil
	})}

	result := tool.Execute(context.Background(), map[string]any{
		"url": "https://example.com/hook",
	})
	if !result.IsError {
		t.Fatal("expected non-2xx response to be an error")
	}
	if !strings.Contains(result.ForLLM, "\"status_code\": 400") {
		t.Fatalf("expected structured error payload, got %s", result.ForLLM)
	}
}

func TestWebhookPostTool_InvalidHeaders(t *testing.T) {
	tool, err := NewWebhookPostTool("", nil)
	if err != nil {
		t.Fatalf("NewWebhookPostTool() error: %v", err)
	}

	result := tool.Execute(context.Background(), map[string]any{
		"url":     "https://example.com/hook",
		"headers": []any{"bad"},
	})
	if !result.IsError {
		t.Fatal("expected invalid headers to fail")
	}
	if !strings.Contains(result.ForLLM, "headers must be an object") {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
}
