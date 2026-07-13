package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
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
