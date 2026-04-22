package gateway

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/sipeed/jameclaw/pkg/config"
)

func TestHookIngressMappedRoute(t *testing.T) {
	runner := &fakeHookRunner{directResult: "done"}
	cfg := &config.Config{
		Hooks: config.HooksConfig{
			Enabled: true,
			Ingress: config.WebhookIngressConfig{
				Enabled: true,
				Token:   "secret",
				Path:    "/hooks",
			},
			Mappings: []config.WebhookMappingConfig{
				{
					ID:              "demo",
					Match:           config.WebhookMappingMatchConfig{Path: "custom"},
					Action:          "agent",
					MessageTemplate: "Hello {{path \"title\"}}",
					Deliver:         boolPtr(false),
				},
			},
		},
	}
	server, err := newHookIngressServer(cfg, "", runner, nil)
	if err != nil {
		t.Fatalf("newHookIngressServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/hooks/custom", bytes.NewBufferString(`{"title":"Ada"}`))
	req.Header.Set("Authorization", "Bearer secret")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}
	if len(runner.directCalls) != 1 {
		t.Fatalf("directCalls = %d, want 1", len(runner.directCalls))
	}
	if runner.directCalls[0].content != "Hello Ada" {
		t.Fatalf("content = %q, want %q", runner.directCalls[0].content, "Hello Ada")
	}

	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["delivered"] != false {
		t.Fatalf("delivered = %v, want false", body["delivered"])
	}
}

func TestHookMappingsTransformTemplate(t *testing.T) {
	configDir := t.TempDir()
	transformsDir := filepath.Join(configDir, "hooks", "transforms")
	if err := os.MkdirAll(transformsDir, 0o755); err != nil {
		t.Fatalf("mkdir transforms: %v", err)
	}
	if err := os.WriteFile(filepath.Join(transformsDir, "rewrite.tmpl"), []byte(`{"kind":"wake","text":"{{path "title"}}","mode":"now"}`), 0o600); err != nil {
		t.Fatalf("write transform: %v", err)
	}

	mappings, err := resolveHookMappings(config.HooksConfig{
		Enabled: true,
		Mappings: []config.WebhookMappingConfig{
			{
				ID:     "rewrite",
				Match:  config.WebhookMappingMatchConfig{Path: "custom"},
				Action: "agent",
				Transform: &config.WebhookMappingTransformConfig{
					Module: "rewrite.tmpl",
				},
			},
		},
	}, configDir)
	if err != nil {
		t.Fatalf("resolveHookMappings: %v", err)
	}

	outcome, err := applyHookMappings(mappings, hookTemplateContext{
		Payload: map[string]any{"title": "Launch"},
		Headers: map[string]string{},
		URL:     "http://127.0.0.1/hooks/custom",
		Path:    "custom",
	})
	if err != nil {
		t.Fatalf("applyHookMappings: %v", err)
	}
	if !outcome.Matched || outcome.Skipped || outcome.Action == nil {
		t.Fatalf("unexpected outcome: %+v", outcome)
	}
	if outcome.Action.kind != hookMappingActionWake || outcome.Action.text != "Launch" {
		t.Fatalf("unexpected action: %+v", outcome.Action)
	}
}

func TestHookMappingsPresetGmail(t *testing.T) {
	mappings, err := resolveHookMappings(config.HooksConfig{
		Enabled: true,
		Presets: config.FlexibleStringSlice{"gmail"},
	}, t.TempDir())
	if err != nil {
		t.Fatalf("resolveHookMappings: %v", err)
	}
	if len(mappings) == 0 {
		t.Fatal("expected gmail preset mapping")
	}

	outcome, err := applyHookMappings(mappings, hookTemplateContext{
		Payload: map[string]any{
			"messages": []any{
				map[string]any{
					"id":      "msg-1",
					"from":    "Ada",
					"subject": "Hello",
					"snippet": "Ping",
					"body":    "Body text",
				},
			},
		},
		Headers: map[string]string{},
		URL:     "http://127.0.0.1/hooks/gmail",
		Path:    "gmail",
	})
	if err != nil {
		t.Fatalf("applyHookMappings: %v", err)
	}
	if !outcome.Matched || outcome.Skipped || outcome.Action == nil {
		t.Fatalf("unexpected outcome: %+v", outcome)
	}
	if outcome.Action.kind != hookMappingActionAgent {
		t.Fatalf("kind = %q, want agent", outcome.Action.kind)
	}
	if outcome.Action.message == "" {
		t.Fatal("gmail preset should render a message")
	}
}

func TestHookMappingsRejectsTransformTraversal(t *testing.T) {
	_, err := resolveHookMappings(config.HooksConfig{
		Enabled: true,
		Mappings: []config.WebhookMappingConfig{
			{
				Match:  config.WebhookMappingMatchConfig{Path: "custom"},
				Action: "agent",
				Transform: &config.WebhookMappingTransformConfig{
					Module: "../evil.tmpl",
				},
			},
		},
	}, t.TempDir())
	if err == nil {
		t.Fatal("expected traversal error")
	}
}
