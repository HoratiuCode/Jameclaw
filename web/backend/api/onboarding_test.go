package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/sipeed/jameclaw/pkg/config"
)

func TestOnboardingStatusReportsSevenSteps(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("JAMECLAW_HOME", tempDir)

	workspace := filepath.Join(tempDir, "workspace")
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = workspace
	cfg.Agents.Defaults.ModelName = "remote"
	cfg.ModelList = []*config.ModelConfig{{
		ModelName: "remote",
		Model:     "vllm/test-model",
		APIBase:   "https://models.example.com/v1",
	}}
	cfg.ModelList[0].SetAPIKey("test-key")

	configPath := filepath.Join(tempDir, "config.json")
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	status := h.onboardingStatus()

	if status.TotalCount != 7 {
		t.Fatalf("TotalCount = %d, want 7", status.TotalCount)
	}
	if len(status.Steps) != 7 {
		t.Fatalf("len(Steps) = %d, want 7", len(status.Steps))
	}
	if status.Complete {
		t.Fatal("Complete = true, want false before completion")
	}

	ids := make(map[string]bool, len(status.Steps))
	for _, step := range status.Steps {
		ids[step.ID] = true
	}
	for _, id := range []string{"config", "workspace", "model", "credentials", "gateway", "channels", "chat"} {
		if !ids[id] {
			t.Fatalf("missing onboarding step %q", id)
		}
	}
}

func TestOnboardingCompletePersistsState(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("JAMECLAW_HOME", tempDir)

	configPath := filepath.Join(tempDir, "config.json")
	if err := config.SaveConfig(configPath, config.DefaultConfig()); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/onboarding/complete", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/onboarding/status", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var status onboardingStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !status.Complete {
		t.Fatal("Complete = false, want true after completion")
	}
	if status.ShouldShow {
		t.Fatal("ShouldShow = true, want false after completion")
	}
}
