package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sipeed/jameclaw/pkg/config"
)

func TestResearchProviderConnectionPersistsSecretAndSelection(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	missingKey := httptest.NewRecorder()
	mux.ServeHTTP(missingKey, httptest.NewRequest(http.MethodPost, "/api/research-providers/tavily", bytes.NewBufferString(`{"enabled":true}`)))
	if missingKey.Code != http.StatusUnprocessableEntity {
		t.Fatalf("missing key status=%d body=%s", missingKey.Code, missingKey.Body.String())
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/research-providers/tavily", bytes.NewBufferString(`{"enabled":true,"api_key":"tvly-secret-test"}`))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "tvly-secret-test") {
		t.Fatal("research provider response exposed the API key")
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Tools.Web.Enabled || !cfg.Tools.Web.Tavily.Enabled || cfg.Tools.Web.PreferNative {
		t.Fatalf("web research config was not activated: %#v", cfg.Tools.Web)
	}
	if cfg.Tools.Web.Tavily.APIKey() != "tvly-secret-test" {
		t.Fatal("Tavily key was not persisted in secure configuration")
	}

	listRec := httptest.NewRecorder()
	mux.ServeHTTP(listRec, httptest.NewRequest(http.MethodGet, "/api/research-providers", nil))
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listRec.Code, listRec.Body.String())
	}
	var response struct {
		ActiveProvider string                   `json:"active_provider"`
		Providers      []researchProviderStatus `json:"providers"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.ActiveProvider != "tavily" {
		t.Fatalf("active provider=%q", response.ActiveProvider)
	}
	if strings.Contains(listRec.Body.String(), "tvly-secret-test") {
		t.Fatal("provider status response exposed the API key")
	}
}
