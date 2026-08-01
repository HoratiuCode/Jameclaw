package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/sipeed/jameclaw/pkg/config"
)

func TestTeamOperationsEnforcesDependenciesAndVerification(t *testing.T) {
	mux, workspace := setupTeamOperationsTest(t)
	putTeamGoal(t, mux)

	first := createTeamTask(t, mux, map[string]any{
		"title":               "Implement backend",
		"owner_agent_id":      "builder",
		"file_scopes":         []string{"pkg/backend"},
		"acceptance_criteria": []string{"Focused tests pass"},
		"time_budget_minutes": 45,
		"token_budget":        12000,
	})
	second := createTeamTask(t, mux, map[string]any{
		"title":          "Verify implementation",
		"owner_agent_id": "reviewer",
		"depends_on":     []string{first.ID},
		"file_scopes":    []string{"pkg/review"},
	})

	response := teamRequest(t, mux, http.MethodPost, "/api/agents/team-operations/tasks/"+second.ID+"/action", map[string]any{"action": "start"})
	if response.Code != http.StatusConflict {
		t.Fatalf("dependent start status=%d body=%s", response.Code, response.Body.String())
	}
	response = teamRequest(t, mux, http.MethodPost, "/api/agents/team-operations/tasks/"+first.ID+"/action", map[string]any{"action": "start"})
	if response.Code != http.StatusOK {
		t.Fatalf("start status=%d body=%s", response.Code, response.Body.String())
	}
	response = teamRequest(t, mux, http.MethodPost, "/api/agents/team-operations/tasks/"+first.ID+"/action", map[string]any{"action": "submit_review", "result": "Backend and focused tests implemented."})
	if response.Code != http.StatusOK {
		t.Fatalf("review status=%d body=%s", response.Code, response.Body.String())
	}
	response = teamRequest(t, mux, http.MethodPost, "/api/agents/team-operations/tasks/"+first.ID+"/action", map[string]any{"action": "complete"})
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("complete without evidence status=%d body=%s", response.Code, response.Body.String())
	}
	response = teamRequest(t, mux, http.MethodPost, "/api/agents/team-operations/tasks/"+first.ID+"/action", map[string]any{"action": "complete", "verification": "go test ./pkg/backend passed"})
	if response.Code != http.StatusOK {
		t.Fatalf("complete status=%d body=%s", response.Code, response.Body.String())
	}
	response = teamRequest(t, mux, http.MethodPost, "/api/agents/team-operations/tasks/"+second.ID+"/action", map[string]any{"action": "start"})
	if response.Code != http.StatusOK {
		t.Fatalf("dependent start after completion status=%d body=%s", response.Code, response.Body.String())
	}

	path := filepath.Join(workspace, "state", "team-operations.json")
	snapshot, err := loadTeamOperations(path)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Tasks[0].Status != "done" || snapshot.Tasks[1].Status != "working" {
		t.Fatalf("unexpected persisted states: %#v", snapshot.Tasks)
	}
}

func TestTeamOperationsBlocksConcurrentFileScopes(t *testing.T) {
	mux, _ := setupTeamOperationsTest(t)
	putTeamGoal(t, mux)
	first := createTeamTask(t, mux, map[string]any{
		"title":          "Edit native app",
		"owner_agent_id": "builder",
		"file_scopes":    []string{"macos/JameClawHome"},
	})
	second := createTeamTask(t, mux, map[string]any{
		"title":          "Review Swift screen",
		"owner_agent_id": "reviewer",
		"file_scopes":    []string{"macos/JameClawHome/JameClawHome.swift"},
	})
	if response := teamRequest(t, mux, http.MethodPost, "/api/agents/team-operations/tasks/"+first.ID+"/action", map[string]any{"action": "start"}); response.Code != http.StatusOK {
		t.Fatalf("first start status=%d body=%s", response.Code, response.Body.String())
	}
	response := teamRequest(t, mux, http.MethodPost, "/api/agents/team-operations/tasks/"+second.ID+"/action", map[string]any{"action": "start"})
	if response.Code != http.StatusConflict {
		t.Fatalf("overlapping scope status=%d body=%s", response.Code, response.Body.String())
	}
}

func setupTeamOperationsTest(t *testing.T) (*http.ServeMux, string) {
	t.Helper()
	configPath, cleanup := setupOAuthTestEnv(t)
	t.Cleanup(cleanup)
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	workspace := t.TempDir()
	cfg.Agents.Defaults.Workspace = workspace
	cfg.Agents.List = append(cfg.Agents.List,
		config.AgentConfig{ID: "builder", Name: "Builder"},
		config.AgentConfig{ID: "reviewer", Name: "Reviewer"},
	)
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(configPath)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	return mux, workspace
}

func putTeamGoal(t *testing.T, mux *http.ServeMux) {
	t.Helper()
	response := teamRequest(t, mux, http.MethodPut, "/api/agents/team-operations/goal", map[string]any{
		"title":         "Ship Team Operations",
		"outcome":       "A dependency-aware team workflow passes verification.",
		"lead_agent_id": "main",
	})
	if response.Code != http.StatusOK {
		t.Fatalf("goal status=%d body=%s", response.Code, response.Body.String())
	}
}

func createTeamTask(t *testing.T, mux *http.ServeMux, payload map[string]any) teamTask {
	t.Helper()
	response := teamRequest(t, mux, http.MethodPost, "/api/agents/team-operations/tasks", payload)
	if response.Code != http.StatusCreated {
		t.Fatalf("create task status=%d body=%s", response.Code, response.Body.String())
	}
	var task teamTask
	if err := json.Unmarshal(response.Body.Bytes(), &task); err != nil {
		t.Fatal(err)
	}
	return task
}

func teamRequest(t *testing.T, mux *http.ServeMux, method, path string, payload map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	return response
}
