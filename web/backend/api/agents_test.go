package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sipeed/jameclaw/pkg/config"
)

func TestHandleCreateAgent_AddsSubagentToParent(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/agents", bytes.NewBufferString(`{
		"id": "research-helper",
		"name": "Research Helper",
		"model": "custom-default",
		"workspace": "/tmp/research",
		"parent_id": "main"
	}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	var created *config.AgentConfig
	var main *config.AgentConfig
	for i := range cfg.Agents.List {
		switch cfg.Agents.List[i].ID {
		case "research-helper":
			created = &cfg.Agents.List[i]
		case "main":
			main = &cfg.Agents.List[i]
		}
	}
	if created == nil {
		t.Fatal("created agent not found")
	}
	if created.Name != "Research Helper" {
		t.Fatalf("created.Name = %q, want %q", created.Name, "Research Helper")
	}
	if created.Workspace != "/tmp/research" {
		t.Fatalf("created.Workspace = %q, want %q", created.Workspace, "/tmp/research")
	}
	if created.Model == nil || created.Model.Primary != "custom-default" {
		t.Fatalf("created.Model.Primary = %#v, want custom-default", created.Model)
	}
	if main == nil || main.Subagents == nil {
		t.Fatal("main subagent allow-list was not created")
	}
	if got := main.Subagents.AllowAgents; len(got) != 1 || got[0] != "research-helper" {
		t.Fatalf("main.Subagents.AllowAgents = %#v, want [research-helper]", got)
	}
}
