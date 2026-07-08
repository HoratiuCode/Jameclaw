package api

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/pkg/cron"
)

type dashboardCard struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	Description string `json:"description,omitempty"`
	Count       int    `json:"count,omitempty"`
}

func (h *Handler) registerDashboardRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/profiles", h.handleProfilesSummary)
	mux.HandleFunc("GET /api/cron", h.handleCronSummary)
	mux.HandleFunc("GET /api/pairing", h.handlePairingSummary)
	mux.HandleFunc("GET /api/plugins", h.handlePluginsSummary)
	mux.HandleFunc("GET /api/analytics", h.handleAnalyticsSummary)
}

func (h *Handler) handleProfilesSummary(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}

	cards := []dashboardCard{{
		ID:          "main",
		Title:       "Main",
		Status:      "default",
		Description: cfg.WorkspacePath(),
	}}
	for _, agent := range cfg.Agents.List {
		if agent.ID == "main" {
			continue
		}
		status := "available"
		if agent.Default {
			status = "default"
		}
		name := strings.TrimSpace(agent.Name)
		if name == "" {
			name = agent.ID
		}
		workspace := strings.TrimSpace(agent.Workspace)
		if workspace == "" {
			workspace = cfg.WorkspacePath()
		}
		cards = append(cards, dashboardCard{
			ID:          agent.ID,
			Title:       name,
			Status:      status,
			Description: workspace,
			Count:       len(agent.Skills),
		})
	}

	writeDashboardCards(w, cards)
}

func (h *Handler) handleCronSummary(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}
	service := cron.NewCronService(filepath.Join(cfg.WorkspacePath(), "cron", "jobs.json"), nil)
	jobs := service.ListJobs(true)
	cards := make([]dashboardCard, 0, len(jobs))
	for _, job := range jobs {
		status := "disabled"
		if job.Enabled {
			status = "enabled"
		}
		cards = append(cards, dashboardCard{
			ID:          job.ID,
			Title:       job.Name,
			Status:      status,
			Description: job.Payload.Message,
		})
	}
	writeDashboardCards(w, cards)
}

func (h *Handler) handlePairingSummary(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}
	cards := enabledChannelCards(cfg.Channels)
	writeDashboardCards(w, cards)
}

func (h *Handler) handlePluginsSummary(w http.ResponseWriter, r *http.Request) {
	cards := []dashboardCard{
		{ID: "skills", Title: "Skills", Status: "built-in", Description: "Workspace, user, and bundled skill directories"},
		{ID: "mcp", Title: "MCP", Status: "configured", Description: "External tool servers configured through tools/MCP settings"},
		{ID: "extensions", Title: "Extensions", Status: "available", Description: "Dashboard extension catalog"},
	}
	writeDashboardCards(w, cards)
}

func (h *Handler) handleAnalyticsSummary(w http.ResponseWriter, r *http.Request) {
	sessions := h.listAllSessions()
	messageCount := 0
	toolCalls := 0
	for _, sess := range sessions {
		for _, msg := range sess.Messages {
			if strings.TrimSpace(msg.Content) != "" || len(msg.ToolCalls) > 0 {
				messageCount++
			}
			toolCalls += len(msg.ToolCalls)
		}
	}
	writeDashboardCards(w, []dashboardCard{
		{ID: "sessions", Title: "Sessions", Status: "tracked", Count: len(sessions)},
		{ID: "messages", Title: "Messages", Status: "tracked", Count: messageCount},
		{ID: "tool-calls", Title: "Tool calls", Status: "tracked", Count: toolCalls},
	})
}

func enabledChannelCards(channels config.ChannelsConfig) []dashboardCard {
	value := reflect.ValueOf(channels)
	valueType := value.Type()
	cards := []dashboardCard{}
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		if field.Kind() != reflect.Struct {
			continue
		}
		enabled := field.FieldByName("Enabled")
		if !enabled.IsValid() || enabled.Kind() != reflect.Bool || !enabled.Bool() {
			continue
		}
		cards = append(cards, dashboardCard{
			ID:     strings.ToLower(valueType.Field(i).Name),
			Title:  valueType.Field(i).Name,
			Status: "enabled",
		})
	}
	return cards
}

func writeDashboardCards(w http.ResponseWriter, cards []dashboardCard) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"items": cards})
}
