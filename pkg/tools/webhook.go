package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sipeed/jameclaw/pkg/utils"
)

const webhookTimeout = 30 * time.Second

type WebhookPostTool struct {
	client    *http.Client
	whitelist *privateHostWhitelist
}

func NewWebhookPostTool(proxy string, privateHostWhitelist []string) (*WebhookPostTool, error) {
	whitelist, err := newPrivateHostWhitelist(privateHostWhitelist)
	if err != nil {
		return nil, fmt.Errorf("failed to parse webhook private host whitelist: %w", err)
	}
	client, err := utils.CreateHTTPClient(proxy, webhookTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client for webhook_post: %w", err)
	}
	return &WebhookPostTool{
		client:    client,
		whitelist: whitelist,
	}, nil
}

func (t *WebhookPostTool) Name() string {
	return "webhook_post"
}

func (t *WebhookPostTool) Description() string {
	return "Send a POST request to an external webhook URL with optional JSON payload and headers."
}

func (t *WebhookPostTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "Webhook URL to call. Can be copied from workspace TOOLS.md.",
			},
			"payload": map[string]any{
				"description": "Webhook payload. Prefer a JSON object; strings are sent as-is.",
			},
			"headers": map[string]any{
				"type":        "object",
				"description": "Optional HTTP headers to include. Values must be strings.",
				"additionalProperties": map[string]any{
					"type": "string",
				},
			},
		},
		"required": []string{"url"},
	}
}

func (t *WebhookPostTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	urlStr, ok := args["url"].(string)
	if !ok || strings.TrimSpace(urlStr) == "" {
		return ErrorResult("url is required")
	}

	parsedURL, err := url.Parse(strings.TrimSpace(urlStr))
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid URL: %v", err))
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return ErrorResult("only http/https URLs are allowed")
	}
	if parsedURL.Host == "" {
		return ErrorResult("missing domain in URL")
	}
	if isObviousPrivateHost(parsedURL.Hostname(), t.whitelist) {
		return ErrorResult("posting to private or local network hosts is not allowed")
	}

	bodyBytes, contentType, err := buildWebhookPayload(args["payload"])
	if err != nil {
		return ErrorResult(err.Error())
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, parsedURL.String(), bytes.NewReader(bodyBytes))
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to create request: %v", err))
	}
	req.Header.Set("User-Agent", "jameclaw-webhook-post/1.0")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	if headersRaw, ok := args["headers"]; ok {
		headers, err := normalizeWebhookHeaders(headersRaw)
		if err != nil {
			return ErrorResult(err.Error())
		}
		for key, value := range headers {
			if strings.EqualFold(key, "Host") || strings.EqualFold(key, "Content-Length") {
				continue
			}
			req.Header.Set(key, value)
		}
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return ErrorResult(fmt.Sprintf("request failed: %v", err))
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to read response: %v", err))
	}

	responseText := strings.TrimSpace(string(respBody))
	resultPayload := map[string]any{
		"url":         parsedURL.String(),
		"status_code": resp.StatusCode,
		"status":      resp.Status,
	}
	if responseText != "" {
		resultPayload["response_body"] = responseText
	}
	resultJSON, _ := json.MarshalIndent(resultPayload, "", "  ")

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ErrorResult(string(resultJSON))
	}

	return &ToolResult{
		ForLLM:  string(resultJSON),
		ForUser: fmt.Sprintf("Webhook sent: %s", resp.Status),
	}
}

func buildWebhookPayload(raw any) ([]byte, string, error) {
	if raw == nil {
		return []byte("{}"), "application/json", nil
	}

	switch v := raw.(type) {
	case string:
		return []byte(v), "text/plain; charset=utf-8", nil
	case map[string]any:
		bodyBytes, err := json.Marshal(v)
		if err != nil {
			return nil, "", fmt.Errorf("failed to encode payload as JSON: %w", err)
		}
		return bodyBytes, "application/json", nil
	case []any:
		bodyBytes, err := json.Marshal(v)
		if err != nil {
			return nil, "", fmt.Errorf("failed to encode payload as JSON: %w", err)
		}
		return bodyBytes, "application/json", nil
	default:
		bodyBytes, err := json.Marshal(v)
		if err != nil {
			return nil, "", fmt.Errorf("unsupported payload type: %T", raw)
		}
		return bodyBytes, "application/json", nil
	}
}

func normalizeWebhookHeaders(raw any) (map[string]string, error) {
	if raw == nil {
		return nil, nil
	}

	headersMap, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("headers must be an object with string values")
	}

	headers := make(map[string]string, len(headersMap))
	for key, value := range headersMap {
		strValue, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("header %q must be a string", key)
		}
		headers[key] = strValue
	}

	return headers, nil
}
