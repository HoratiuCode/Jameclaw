package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/pkg/cron"
)

func TestHandleAutomationListReturnsCronJobs(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	storePath := filepath.Join(cfg.WorkspacePath(), "cron", "jobs.json")
	service := cron.NewCronService(storePath, nil)
	job, err := service.AddJob(
		"Daily news",
		cron.CronSchedule{Kind: "cron", Expr: "0 20 * * *"},
		"Send me the top technology news.",
		true,
		true,
		"telegram",
		"12345",
	)
	if err != nil {
		t.Fatalf("AddJob() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/automation", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp automationResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(resp.Items))
	}
	item := resp.Items[0]
	if item.ID != job.ID {
		t.Fatalf("id = %q, want %q", item.ID, job.ID)
	}
	if item.Name != "Daily news" {
		t.Fatalf("name = %q, want Daily news", item.Name)
	}
	if item.Schedule != "Every day at 20:00" {
		t.Fatalf("schedule = %q, want Every day at 20:00", item.Schedule)
	}
	if item.Prompt != "Send me the top technology news." {
		t.Fatalf("prompt = %q", item.Prompt)
	}
	if item.Delivery != "Sends result on telegram to 12345" {
		t.Fatalf("delivery = %q", item.Delivery)
	}
	if !item.DeliveryApproved {
		t.Fatal("delivery_approved = false, want true")
	}
	if item.Status != "scheduled" {
		t.Fatalf("status = %q, want scheduled", item.Status)
	}
}

func TestFormatAutomationScheduleEveryInterval(t *testing.T) {
	every := int64(24 * 60 * 60 * 1000)
	got := formatAutomationSchedule(cron.CronSchedule{Kind: "every", EveryMS: &every})
	if got != "Every day" {
		t.Fatalf("formatAutomationSchedule() = %q, want Every day", got)
	}
}

func TestHandleAutomationBlueprintList(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/automation/blueprints", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp automationBlueprintResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(resp.Blueprints) == 0 {
		t.Fatal("expected blueprints")
	}
	if resp.Blueprints[0].Key == "" || len(resp.Blueprints[0].Fields) == 0 {
		t.Fatalf("invalid blueprint response: %+v", resp.Blueprints[0])
	}
}

func TestHandleAutomationBlueprintInstantiateCreatesCronJob(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body := []byte(`{"blueprint":"news-digest","values":{"topic":"robotics","time":"09:15","recurrence":"weekdays","count":"3","deliver":"local"}}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/automation/blueprints/instantiate", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp automationBlueprintInstantiateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if resp.Item.Name != "Topic news digest" {
		t.Fatalf("name = %q, want Topic news digest", resp.Item.Name)
	}
	if resp.Item.Schedule != "Every weekdays at 09:15" {
		t.Fatalf("schedule = %q, want Every weekdays at 09:15", resp.Item.Schedule)
	}
	if resp.Item.Delivery != "Runs in JameClaw" {
		t.Fatalf("delivery = %q, want Runs in JameClaw", resp.Item.Delivery)
	}
	if !strings.Contains(resp.Item.Prompt, "robotics") || !strings.Contains(resp.Item.Prompt, "3 bullets") {
		t.Fatalf("prompt did not fill values: %q", resp.Item.Prompt)
	}
}

func TestHandleAutomationBlueprintInstantiateRejectsInvalidSlot(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body := []byte(`{"blueprint":"news-digest","values":{"time":"25:99"}}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/automation/blueprints/instantiate", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
}

func TestHandleAutomationBlueprintInstantiateRejectsOriginDeliveryFromDashboard(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body := []byte(`{"blueprint":"morning-brief","values":{"deliver":"origin"}}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/automation/blueprints/instantiate", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
}
