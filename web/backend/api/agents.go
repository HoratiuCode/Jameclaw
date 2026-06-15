package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/sipeed/jameclaw/pkg/config"
)

type agentSummary struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Default        bool     `json:"default"`
	Workspace      string   `json:"workspace"`
	Model          string   `json:"model"`
	ModelFallbacks []string `json:"model_fallbacks"`
	Skills         []string `json:"skills"`
	Subagents      []string `json:"subagents"`
	SessionCount   int      `json:"session_count"`
	MessageCount   int      `json:"message_count"`
	ToolCalls      int      `json:"tool_calls"`
}

func (h *Handler) registerAgentRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/agents", h.handleListAgents)
	mux.HandleFunc("PATCH /api/agents/{id}", h.handlePatchAgent)
}

func (h *Handler) handleListAgents(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}

	sessions := h.listAllSessions()
	sessionCount := len(sessions)
	messageCount := 0
	toolCalls := 0
	for _, sess := range sessions {
		for _, msg := range sess.Messages {
			if msg.Content != "" || len(msg.ToolCalls) > 0 {
				messageCount++
			}
			toolCalls += len(msg.ToolCalls)
		}
	}

	listDefaultID := ""
	var mainOverride *config.AgentConfig
	for _, agent := range cfg.Agents.List {
		if agent.Default {
			listDefaultID = agent.ID
		}
		if agent.ID == "main" {
			agentCopy := agent
			mainOverride = &agentCopy
		}
	}

	mainModel := cfg.Agents.Defaults.GetModelName()
	mainFallbacks := stringListOrEmpty(cfg.Agents.Defaults.ModelFallbacks)
	mainSkills := []string{}
	mainSubagents := []string{}
	if mainOverride != nil {
		if mainOverride.Model != nil {
			if mainOverride.Model.Primary != "" {
				mainModel = mainOverride.Model.Primary
			}
			if mainOverride.Model.Fallbacks != nil {
				mainFallbacks = stringListOrEmpty(mainOverride.Model.Fallbacks)
			}
		}
		mainSkills = stringListOrEmpty(mainOverride.Skills)
		if mainOverride.Subagents != nil {
			mainSubagents = stringListOrEmpty(mainOverride.Subagents.AllowAgents)
		}
	}

	agents := []agentSummary{
		{
			ID:             "main",
			Name:           "Main",
			Default:        listDefaultID == "" || listDefaultID == "main",
			Workspace:      cfg.Agents.Defaults.Workspace,
			Model:          mainModel,
			ModelFallbacks: mainFallbacks,
			Skills:         mainSkills,
			Subagents:      mainSubagents,
			SessionCount:   sessionCount,
			MessageCount:   messageCount,
			ToolCalls:      toolCalls,
		},
	}

	for _, agent := range cfg.Agents.List {
		if agent.ID == "main" {
			continue
		}
		model := cfg.Agents.Defaults.GetModelName()
		fallbacks := stringListOrEmpty(cfg.Agents.Defaults.ModelFallbacks)
		if agent.Model != nil {
			if agent.Model.Primary != "" {
				model = agent.Model.Primary
			}
			if agent.Model.Fallbacks != nil {
				fallbacks = stringListOrEmpty(agent.Model.Fallbacks)
			}
		}
		workspace := agent.Workspace
		if workspace == "" {
			workspace = cfg.Agents.Defaults.Workspace
		}
		subagents := []string{}
		if agent.Subagents != nil {
			subagents = append(subagents, agent.Subagents.AllowAgents...)
		}
		name := agent.Name
		if name == "" {
			name = agent.ID
		}
		agents = append(agents, agentSummary{
			ID:             agent.ID,
			Name:           name,
			Default:        agent.ID == listDefaultID,
			Workspace:      workspace,
			Model:          model,
			ModelFallbacks: fallbacks,
			Skills:         stringListOrEmpty(agent.Skills),
			Subagents:      subagents,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"agents":            agents,
		"default_model":     cfg.Agents.Defaults.GetModelName(),
		"enabled_channels":  countEnabledChannels(cfg.Channels),
		"configured_models": len(cfg.ModelList),
	})
}

func stringListOrEmpty(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return append([]string(nil), values...)
}

func (h *Handler) handlePatchAgent(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("id")
	if agentID == "" {
		http.Error(w, "missing agent id", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	type patchAgentRequest struct {
		Default        *bool    `json:"default"`
		Model          *string  `json:"model"`
		ModelFallbacks []string `json:"model_fallbacks"`
	}
	var req patchAgentRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}

	if agentID == "main" {
		if req.Default != nil && *req.Default {
			for i := range cfg.Agents.List {
				cfg.Agents.List[i].Default = false
			}
		}
		if req.Model != nil {
			cfg.Agents.Defaults.ModelName = *req.Model
		}
		if req.ModelFallbacks != nil {
			cfg.Agents.Defaults.ModelFallbacks = cleanStringList(req.ModelFallbacks)
		}
	} else {
		index := -1
		for i := range cfg.Agents.List {
			if cfg.Agents.List[i].ID == agentID {
				index = i
				break
			}
		}
		if index < 0 {
			http.Error(w, "agent not found", http.StatusNotFound)
			return
		}
		if req.Default != nil && *req.Default {
			for i := range cfg.Agents.List {
				cfg.Agents.List[i].Default = i == index
			}
		}
		if req.Model != nil || req.ModelFallbacks != nil {
			model := cfg.Agents.List[index].Model
			if model == nil {
				model = &config.AgentModelConfig{}
			}
			if req.Model != nil {
				model.Primary = *req.Model
			}
			if req.ModelFallbacks != nil {
				model.Fallbacks = cleanStringList(req.ModelFallbacks)
			}
			cfg.Agents.List[index].Model = model
		}
	}

	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, fmt.Sprintf("failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func cleanStringList(values []string) []string {
	out := []string{}
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func countEnabledChannels(channels config.ChannelsConfig) int {
	count := 0
	value := reflect.ValueOf(channels)
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		if field.Kind() != reflect.Struct {
			continue
		}
		enabled := field.FieldByName("Enabled")
		if enabled.IsValid() && enabled.Kind() == reflect.Bool && enabled.Bool() {
			count++
		}
	}
	return count
}
