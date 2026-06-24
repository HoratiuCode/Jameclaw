package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleListChannelCatalog_IncludesBuiltInChannels(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/channels/catalog", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Channels []channelCatalogItem `json:"channels"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if len(resp.Channels) < 10 {
		t.Fatalf("len(channels) = %d, want expanded built-in channel catalog", len(resp.Channels))
	}

	channelsByName := make(map[string]channelCatalogItem, len(resp.Channels))
	for _, channel := range resp.Channels {
		channelsByName[channel.Name] = channel
	}

	expected := map[string]string{
		"telegram":        "telegram",
		"whatsapp":        "whatsapp",
		"whatsapp_native": "whatsapp",
		"discord":         "discord",
		"slack":           "slack",
		"matrix":          "matrix",
		"line":            "line",
		"feishu":          "feishu",
		"dingtalk":        "dingtalk",
		"jame":            "jame",
		"jame_client":     "jame_client",
	}
	for name, configKey := range expected {
		channel, ok := channelsByName[name]
		if !ok {
			t.Fatalf("channel %q missing from catalog", name)
		}
		if channel.ConfigKey != configKey {
			t.Fatalf("channel %q config_key = %q, want %q", name, channel.ConfigKey, configKey)
		}
	}
}
