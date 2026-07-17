package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/pkg/cron"
)

type automationResponse struct {
	Items []automationItem `json:"items"`
}

type automationBlueprintResponse struct {
	Blueprints []cron.AutomationBlueprint `json:"blueprints"`
}

type automationBlueprintInstantiateRequest struct {
	Blueprint string            `json:"blueprint"`
	Values    map[string]string `json:"values"`
}

type automationBlueprintInstantiateResponse struct {
	Item automationItem `json:"item"`
}

type automationItem struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Enabled           bool   `json:"enabled"`
	Status            string `json:"status"`
	Schedule          string `json:"schedule"`
	Prompt            string `json:"prompt"`
	Delivery          string `json:"delivery"`
	DeliveryApproved  bool   `json:"delivery_approved"`
	NextRunAtMS       *int64 `json:"next_run_at_ms,omitempty"`
	LastRunAtMS       *int64 `json:"last_run_at_ms,omitempty"`
	LastStatus        string `json:"last_status,omitempty"`
	LastError         string `json:"last_error,omitempty"`
	Running           bool   `json:"running"`
	CreatedAtMS       int64  `json:"created_at_ms"`
	UpdatedAtMS       int64  `json:"updated_at_ms"`
	DeleteAfterRun    bool   `json:"delete_after_run"`
	Timezone          string `json:"timezone,omitempty"`
	RetryAttempts     int    `json:"retry_attempts,omitempty"`
	RetryDelaySeconds int    `json:"retry_delay_seconds,omitempty"`
	QuietHoursStart   string `json:"quiet_hours_start,omitempty"`
	QuietHoursEnd     string `json:"quiet_hours_end,omitempty"`
	MaxRunsPerDay     int    `json:"max_runs_per_day,omitempty"`
	RunsToday         int    `json:"runs_today,omitempty"`
}

func (h *Handler) registerAutomationRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/automation", h.handleAutomationList)
	mux.HandleFunc("GET /api/automation/{id}/output", h.handleAutomationOutput)
	mux.HandleFunc("POST /api/automation/{id}/run", h.handleAutomationRun)
	mux.HandleFunc("GET /api/automation/blueprints", h.handleAutomationBlueprintList)
	mux.HandleFunc("POST /api/automation/blueprints/instantiate", h.handleAutomationBlueprintInstantiate)
}

func (h *Handler) handleAutomationRun(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		http.Error(w, "automation id is required", http.StatusBadRequest)
		return
	}

	gatewayURL := h.gatewayProxyURL()
	gatewayURL.Path = "/automation/run/" + url.PathEscape(id)
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, gatewayURL.String(), nil)
	if err != nil {
		http.Error(w, "failed to create gateway request", http.StatusInternalServerError)
		return
	}
	token := strings.TrimSpace(cfg.Channels.Jame.Token())
	if token == "" {
		http.Error(w, "Jame gateway access token is unavailable", http.StatusServiceUnavailable)
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		http.Error(w, "gateway is unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		http.Error(w, strings.TrimSpace(string(message)), resp.StatusCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "running"})
}

func (h *Handler) handleAutomationOutput(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	service := cron.NewCronService(filepath.Join(cfg.WorkspacePath(), "cron", "jobs.json"), nil)
	var job *cron.CronJob
	for _, candidate := range service.ListJobs(true) {
		if candidate.ID == id {
			jobCopy := candidate
			job = &jobCopy
			break
		}
	}
	if job == nil {
		http.Error(w, "automation not found", http.StatusNotFound)
		return
	}
	if job.State.LastOutputPath == "" {
		http.Error(w, "no output has been recorded yet", http.StatusNotFound)
		return
	}

	outputRoot := filepath.Join(cfg.WorkspacePath(), "cron", "output")
	relPath, err := filepath.Rel(outputRoot, job.State.LastOutputPath)
	if err != nil || relPath == "." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) || filepath.IsAbs(relPath) {
		http.Error(w, "invalid automation output path", http.StatusInternalServerError)
		return
	}
	content, err := os.ReadFile(job.State.LastOutputPath)
	if err != nil {
		http.Error(w, "automation output is unavailable", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"automation_id": id,
		"status":        job.State.LastStatus,
		"ran_at_ms":     job.State.LastRunAtMS,
		"content":       string(content),
	})
}

func (h *Handler) handleAutomationList(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}

	service := cron.NewCronService(filepath.Join(cfg.WorkspacePath(), "cron", "jobs.json"), nil)
	jobs := service.ListJobs(true)
	items := make([]automationItem, 0, len(jobs))
	for _, job := range jobs {
		items = append(items, automationFromCronJob(job))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(automationResponse{Items: items})
}

func (h *Handler) handleAutomationBlueprintList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(automationBlueprintResponse{Blueprints: cron.AutomationBlueprints})
}

func (h *Handler) handleAutomationBlueprintInstantiate(w http.ResponseWriter, r *http.Request) {
	var req automationBlueprintInstantiateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid blueprint request", http.StatusBadRequest)
		return
	}

	spec, err := cron.FillAutomationBlueprint(req.Blueprint, req.Values)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	if spec.Deliver {
		http.Error(w, "origin delivery can only be used from an active chat; choose local in the dashboard", http.StatusUnprocessableEntity)
		return
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}
	service := cron.NewCronService(filepath.Join(cfg.WorkspacePath(), "cron", "jobs.json"), nil)
	job, err := service.AddJob(spec.Name, spec.Schedule, spec.Prompt, false, false, "web", "dashboard")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create automation: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(automationBlueprintInstantiateResponse{Item: automationFromCronJob(*job)})
}

func automationFromCronJob(job cron.CronJob) automationItem {
	status := "disabled"
	if job.Enabled {
		status = "scheduled"
		if job.State.NextRunAtMS == nil {
			status = "waiting"
		}
	}
	if job.State.LastStatus == "error" {
		status = "error"
	}

	return automationItem{
		ID:                job.ID,
		Name:              firstNonEmpty(job.Name, "Untitled automation"),
		Enabled:           job.Enabled,
		Status:            status,
		Schedule:          formatAutomationSchedule(job.Schedule),
		Prompt:            job.Payload.Message,
		Delivery:          formatAutomationDelivery(job.Payload),
		DeliveryApproved:  job.Payload.DeliveryApproved,
		NextRunAtMS:       job.State.NextRunAtMS,
		LastRunAtMS:       job.State.LastRunAtMS,
		LastStatus:        job.State.LastStatus,
		LastError:         job.State.LastError,
		Running:           job.State.RunningAtMS != nil,
		CreatedAtMS:       job.CreatedAtMS,
		UpdatedAtMS:       job.UpdatedAtMS,
		DeleteAfterRun:    job.DeleteAfterRun,
		Timezone:          job.Schedule.TZ,
		RetryAttempts:     job.Policy.RetryAttempts,
		RetryDelaySeconds: job.Policy.RetryDelaySeconds,
		QuietHoursStart:   job.Policy.QuietHoursStart,
		QuietHoursEnd:     job.Policy.QuietHoursEnd,
		MaxRunsPerDay:     job.Policy.MaxRunsPerDay,
		RunsToday:         job.State.RunsToday,
	}
}

func formatAutomationSchedule(schedule cron.CronSchedule) string {
	switch schedule.Kind {
	case "at":
		if schedule.AtMS == nil {
			return "One-time"
		}
		return "One-time at " + time.UnixMilli(*schedule.AtMS).Format("Jan 2, 2006 15:04")
	case "every":
		if schedule.EveryMS == nil {
			return "Recurring interval"
		}
		return "Every " + formatDuration(time.Duration(*schedule.EveryMS)*time.Millisecond)
	case "cron":
		formatted := formatCronExpression(schedule.Expr)
		if schedule.TZ != "" {
			formatted += " (" + schedule.TZ + ")"
		}
		return formatted
	case "event":
		return "On event: " + firstNonEmpty(schedule.Expr, "unnamed")
	default:
		return firstNonEmpty(schedule.Kind, "Scheduled")
	}
}

func formatAutomationDelivery(payload cron.CronPayload) string {
	if !payload.Deliver {
		return "Runs in JameClaw"
	}
	if !payload.DeliveryApproved {
		return "Delivery needs approval"
	}
	channel := strings.TrimSpace(payload.Channel)
	to := strings.TrimSpace(payload.To)
	if channel == "" && to == "" {
		return "Sends result to chat"
	}
	if channel == "" {
		return "Sends result to " + to
	}
	if to == "" {
		return "Sends result on " + channel
	}
	return fmt.Sprintf("Sends result on %s to %s", channel, to)
}

func formatCronExpression(expr string) string {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return firstNonEmpty(expr, "Cron schedule")
	}
	minute, hour, dom, month, dow := fields[0], fields[1], fields[2], fields[3], fields[4]
	if dom == "*" && month == "*" && dow == "*" && isClockField(hour) && isClockField(minute) {
		return fmt.Sprintf("Every day at %02s:%02s", hour, minute)
	}
	if dom == "*" && month == "*" && dow != "*" && isClockField(hour) && isClockField(minute) {
		return fmt.Sprintf("Every %s at %02s:%02s", weekdayLabel(dow), hour, minute)
	}
	if strings.HasPrefix(minute, "*/") && hour == "*" && dom == "*" && month == "*" && dow == "*" {
		return "Every " + strings.TrimPrefix(minute, "*/") + " minutes"
	}
	return expr
}

func formatDuration(d time.Duration) string {
	if d%(24*time.Hour) == 0 {
		days := int(d / (24 * time.Hour))
		if days == 1 {
			return "day"
		}
		return fmt.Sprintf("%d days", days)
	}
	if d%time.Hour == 0 {
		hours := int(d / time.Hour)
		if hours == 1 {
			return "hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}
	if d%time.Minute == 0 {
		minutes := int(d / time.Minute)
		if minutes == 1 {
			return "minute"
		}
		return fmt.Sprintf("%d minutes", minutes)
	}
	return d.String()
}

func isClockField(value string) bool {
	if value == "" || strings.ContainsAny(value, "*/,-") {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func weekdayLabel(value string) string {
	switch strings.ToUpper(value) {
	case "1-5":
		return "weekdays"
	case "0,6", "6,0":
		return "weekends"
	case "0", "7", "SUN":
		return "Sunday"
	case "1", "MON":
		return "Monday"
	case "2", "TUE":
		return "Tuesday"
	case "3", "WED":
		return "Wednesday"
	case "4", "THU":
		return "Thursday"
	case "5", "FRI":
		return "Friday"
	case "6", "SAT":
		return "Saturday"
	default:
		return value
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
