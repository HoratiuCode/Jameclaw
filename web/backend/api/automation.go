package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/pkg/cron"
)

type automationResponse struct {
	Items []automationItem `json:"items"`
}

type automationItem struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Enabled          bool   `json:"enabled"`
	Status           string `json:"status"`
	Schedule         string `json:"schedule"`
	Prompt           string `json:"prompt"`
	Delivery         string `json:"delivery"`
	DeliveryApproved bool   `json:"delivery_approved"`
	NextRunAtMS      *int64 `json:"next_run_at_ms,omitempty"`
	LastRunAtMS      *int64 `json:"last_run_at_ms,omitempty"`
	LastStatus       string `json:"last_status,omitempty"`
	LastError        string `json:"last_error,omitempty"`
	CreatedAtMS      int64  `json:"created_at_ms"`
	UpdatedAtMS      int64  `json:"updated_at_ms"`
	DeleteAfterRun   bool   `json:"delete_after_run"`
}

func (h *Handler) registerAutomationRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/automation", h.handleAutomationList)
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
		ID:               job.ID,
		Name:             firstNonEmpty(job.Name, "Untitled automation"),
		Enabled:          job.Enabled,
		Status:           status,
		Schedule:         formatAutomationSchedule(job.Schedule),
		Prompt:           job.Payload.Message,
		Delivery:         formatAutomationDelivery(job.Payload),
		DeliveryApproved: job.Payload.DeliveryApproved,
		NextRunAtMS:      job.State.NextRunAtMS,
		LastRunAtMS:      job.State.LastRunAtMS,
		LastStatus:       job.State.LastStatus,
		LastError:        job.State.LastError,
		CreatedAtMS:      job.CreatedAtMS,
		UpdatedAtMS:      job.UpdatedAtMS,
		DeleteAfterRun:   job.DeleteAfterRun,
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
		return formatCronExpression(schedule.Expr)
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
