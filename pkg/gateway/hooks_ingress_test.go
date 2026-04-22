package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sipeed/jameclaw/pkg/bus"
	"github.com/sipeed/jameclaw/pkg/config"
)

type fakeHookRunner struct {
	wakeCalls    []hookWakeCall
	directCalls  []hookDirectCall
	lastChannel  string
	lastChatID   string
	wakeResult   string
	directResult string
	wakeErr      error
	directErr    error
}

type hookWakeCall struct {
	content string
	channel string
	chatID  string
}

type hookDirectCall struct {
	agentID    string
	content    string
	sessionKey string
	channel    string
	chatID     string
}

func (f *fakeHookRunner) ProcessHeartbeat(ctx context.Context, content, channel, chatID string) (string, error) {
	_ = ctx
	f.wakeCalls = append(f.wakeCalls, hookWakeCall{content: content, channel: channel, chatID: chatID})
	return f.wakeResult, f.wakeErr
}

func (f *fakeHookRunner) ProcessDirectOnAgent(
	ctx context.Context,
	agentID, content, sessionKey, channel, chatID string,
) (string, error) {
	_ = ctx
	f.directCalls = append(f.directCalls, hookDirectCall{
		agentID:    agentID,
		content:    content,
		sessionKey: sessionKey,
		channel:    channel,
		chatID:     chatID,
	})
	return f.directResult, f.directErr
}

func (f *fakeHookRunner) GetLastChannel() string { return f.lastChannel }

func (f *fakeHookRunner) GetLastChatID() string { return f.lastChatID }

func TestHookIngressWake(t *testing.T) {
	runner := &fakeHookRunner{wakeResult: "woken"}
	cfg := &config.Config{
		Hooks: config.HooksConfig{
			Ingress: config.WebhookIngressConfig{
				Enabled: true,
				Token:   "secret",
				Path:    "/hooks",
			},
		},
	}
	server, err := newHookIngressServer(cfg, runner, nil)
	if err != nil {
		t.Fatalf("newHookIngressServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/hooks/wake", bytes.NewBufferString(`{"text":"ping"}`))
	req.Header.Set("Authorization", "Bearer secret")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}
	if len(runner.wakeCalls) != 1 {
		t.Fatalf("wakeCalls = %d, want 1", len(runner.wakeCalls))
	}
	if runner.wakeCalls[0].content != "ping" || runner.wakeCalls[0].channel != "system" || runner.wakeCalls[0].chatID != "hooks:wake" {
		t.Fatalf("unexpected wake call: %+v", runner.wakeCalls[0])
	}

	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["ok"] != true {
		t.Fatalf("ok = %v, want true", body["ok"])
	}
	if body["result"] != "woken" {
		t.Fatalf("result = %v, want woken", body["result"])
	}
}

func TestHookIngressAgentDeliveryAndSessionPolicy(t *testing.T) {
	runner := &fakeHookRunner{
		lastChannel:  "telegram",
		lastChatID:   "chat-123",
		directResult: "done",
	}
	var published []bus.OutboundMessage
	cfg := &config.Config{
		Hooks: config.HooksConfig{
			Ingress: config.WebhookIngressConfig{
				Enabled:                   true,
				Token:                     "secret",
				Path:                      "/hooks",
				AllowRequestSessionKey:    true,
				AllowedSessionKeyPrefixes: config.FlexibleStringSlice{"hook:"},
				AllowedAgentIds:           config.FlexibleStringSlice{"main"},
			},
		},
	}
	server, err := newHookIngressServer(cfg, runner, func(_ context.Context, msg bus.OutboundMessage) error {
		published = append(published, msg)
		return nil
	})
	if err != nil {
		t.Fatalf("newHookIngressServer: %v", err)
	}

	reqBody := `{"message":"run","agentId":"main","sessionKey":"hook:abc","deliver":true,"channel":"last"}`
	req := httptest.NewRequest(http.MethodPost, "/hooks/agent", bytes.NewBufferString(reqBody))
	req.Header.Set("Authorization", "Bearer secret")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}
	if len(runner.directCalls) != 1 {
		t.Fatalf("directCalls = %d, want 1", len(runner.directCalls))
	}
	call := runner.directCalls[0]
	if call.agentID != "main" || call.sessionKey != "hook:abc" {
		t.Fatalf("unexpected direct call: %+v", call)
	}
	if call.channel != "telegram" || call.chatID != "chat-123" {
		t.Fatalf("unexpected delivery target: %+v", call)
	}
	if len(published) != 1 {
		t.Fatalf("published = %d, want 1", len(published))
	}
	if published[0].Channel != "telegram" || published[0].ChatID != "chat-123" || published[0].Content != "done" {
		t.Fatalf("unexpected published message: %+v", published[0])
	}

	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["delivered"] != true {
		t.Fatalf("delivered = %v, want true", body["delivered"])
	}
	if body["channel"] != "telegram" || body["to"] != "chat-123" {
		t.Fatalf("unexpected response routing: %+v", body)
	}
}

func TestHookIngressRejectsSessionKeyOverride(t *testing.T) {
	runner := &fakeHookRunner{directResult: "done"}
	cfg := &config.Config{
		Hooks: config.HooksConfig{
			Ingress: config.WebhookIngressConfig{
				Enabled: true,
				Token:   "secret",
				Path:    "/hooks",
			},
		},
	}
	server, err := newHookIngressServer(cfg, runner, nil)
	if err != nil {
		t.Fatalf("newHookIngressServer: %v", err)
	}

	reqBody := `{"message":"run","sessionKey":"hook:abc"}`
	req := httptest.NewRequest(http.MethodPost, "/hooks/agent", bytes.NewBufferString(reqBody))
	req.Header.Set("Authorization", "Bearer secret")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", res.Code)
	}
	if len(runner.directCalls) != 0 {
		t.Fatalf("directCalls = %d, want 0", len(runner.directCalls))
	}
}

func TestHookIngressRejectsInvalidToken(t *testing.T) {
	server, err := newHookIngressServer(&config.Config{
		Hooks: config.HooksConfig{
			Ingress: config.WebhookIngressConfig{
				Enabled: true,
				Token:   "secret",
			},
		},
	}, &fakeHookRunner{}, nil)
	if err != nil {
		t.Fatalf("newHookIngressServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/hooks/wake", bytes.NewBufferString(`{"text":"ping"}`))
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", res.Code)
	}
}

func TestHookIngressRejectsUnknownAgentId(t *testing.T) {
	server, err := newHookIngressServer(&config.Config{
		Hooks: config.HooksConfig{
			Ingress: config.WebhookIngressConfig{
				Enabled:         true,
				Token:           "secret",
				AllowedAgentIds: config.FlexibleStringSlice{"main"},
			},
		},
	}, &fakeHookRunner{}, nil)
	if err != nil {
		t.Fatalf("newHookIngressServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/hooks/agent", bytes.NewBufferString(`{"message":"run","agentId":"other"}`))
	req.Header.Set("Authorization", "Bearer secret")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", res.Code)
	}
}

func TestHookIngressDeliveryRequiresLastChannel(t *testing.T) {
	server, err := newHookIngressServer(&config.Config{
		Hooks: config.HooksConfig{
			Ingress: config.WebhookIngressConfig{
				Enabled: true,
				Token:   "secret",
			},
		},
	}, &fakeHookRunner{directResult: "done"}, nil)
	if err != nil {
		t.Fatalf("newHookIngressServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/hooks/agent", bytes.NewBufferString(`{"message":"run","deliver":true,"channel":"last"}`))
	req.Header.Set("Authorization", "Bearer secret")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", res.Code)
	}
}

func TestHookIngressRunnerError(t *testing.T) {
	server, err := newHookIngressServer(&config.Config{
		Hooks: config.HooksConfig{
			Ingress: config.WebhookIngressConfig{
				Enabled: true,
				Token:   "secret",
			},
		},
	}, &fakeHookRunner{wakeErr: errors.New("boom")}, nil)
	if err != nil {
		t.Fatalf("newHookIngressServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/hooks/wake", bytes.NewBufferString(`{"text":"ping"}`))
	req.Header.Set("Authorization", "Bearer secret")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", res.Code)
	}
}
