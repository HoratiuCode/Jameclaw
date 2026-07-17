package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/pkg/media"
)

func TestImageGenerationTool_GeneratesAndStoresImage(t *testing.T) {
	imageBytes := []byte("not-a-real-png-but-a-valid-payload")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/images/generations" {
			t.Fatalf("path = %q, want /images/generations", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization = %q", got)
		}
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request["model"] != "gpt-image-1" || request["prompt"] != "a friendly shrimp" {
			t.Fatalf("unexpected request: %#v", request)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"b64_json": base64.StdEncoding.EncodeToString(imageBytes)}}})
	}))
	defer server.Close()

	model := &config.ModelConfig{ModelName: "images", Model: "openai/gpt-image-1", APIBase: server.URL}
	model.SetAPIKey("test-key")
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ImageModel = "images"
	cfg.ModelList = []*config.ModelConfig{model}
	store := media.NewFileMediaStore()
	tool := NewImageGenerationTool(cfg, 1024, store)
	result := tool.Execute(WithToolContext(context.Background(), "telegram", "chat-1"), map[string]any{"prompt": "a friendly shrimp"})
	if result.IsError {
		t.Fatalf("Execute() error = %s", result.ForLLM)
	}
	if len(result.Media) != 1 {
		t.Fatalf("media refs = %v, want one", result.Media)
	}
	path, meta, err := store.ResolveWithMeta(result.Media[0])
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(imageBytes) {
		t.Fatalf("stored image = %q, want %q", data, imageBytes)
	}
	if meta.ContentType != "image/png" || !strings.HasSuffix(meta.Filename, ".png") {
		t.Fatalf("media metadata = %#v", meta)
	}
}

func TestImageGenerationTool_RequiresConfiguredImageModel(t *testing.T) {
	tool := NewImageGenerationTool(config.DefaultConfig(), 1024, media.NewFileMediaStore())
	result := tool.Execute(WithToolContext(context.Background(), "telegram", "chat-1"), map[string]any{"prompt": "a shrimp"})
	if !result.IsError || !strings.Contains(result.ForLLM, "no image model is configured") {
		t.Fatalf("result = %#v", result)
	}
}
