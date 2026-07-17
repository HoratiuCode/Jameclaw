package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/pkg/fileutil"
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
	Human          human    `json:"human"`
	SessionCount   int      `json:"session_count"`
	MessageCount   int      `json:"message_count"`
	ToolCalls      int      `json:"tool_calls"`
}

type human struct {
	AgentName      string `json:"agent_name"`
	Persona        string `json:"persona"`
	Tone           string `json:"tone"`
	DiscussionMode string `json:"discussion_mode"`
	MemoryNotes    string `json:"memory_notes"`
	StatusStyle    string `json:"status_style"`
}

type agentMemoryResponse struct {
	AgentID      string            `json:"agent_id"`
	Workspace    string            `json:"workspace"`
	MemoryPath   string            `json:"memory_path"`
	LongTerm     string            `json:"long_term"`
	DailyNotes   []agentDailyNote  `json:"daily_notes"`
	HumanNotes   string            `json:"human_notes,omitempty"`
	FilesChecked map[string]string `json:"files_checked"`
}

type agentDailyNote struct {
	Date    string `json:"date"`
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (h *Handler) registerAgentRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/agents", h.handleListAgents)
	mux.HandleFunc("GET /api/agents/{id}/memory", h.handleGetAgentMemory)
	mux.HandleFunc("PUT /api/agents/{id}/memory", h.handlePutAgentMemory)
	mux.HandleFunc("POST /api/agents", h.handleCreateAgent)
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
	mainHuman := human{}
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
		mainHuman = summarizeHuman(mainOverride.Human)
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
			Human:          mainHuman,
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
			Human:          summarizeHuman(agent.Human),
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

func (h *Handler) handleGetAgentMemory(w http.ResponseWriter, r *http.Request) {
	agentID := strings.TrimSpace(r.PathValue("id"))
	if agentID == "" {
		http.Error(w, "missing agent id", http.StatusBadRequest)
		return
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}

	workspace, humanNotes, ok := resolveAgentMemoryWorkspace(cfg, agentID)
	if !ok {
		http.Error(w, "agent not found", http.StatusNotFound)
		return
	}

	memoryDir := filepath.Join(workspace, "memory")
	memoryPath := filepath.Join(memoryDir, "MEMORY.md")
	longTerm := readOptionalTextFile(memoryPath)

	dailyNotes := make([]agentDailyNote, 0, 7)
	filesChecked := map[string]string{"long_term": memoryPath}
	for i := range 7 {
		date := time.Now().AddDate(0, 0, -i)
		compactDate := date.Format("20060102")
		notePath := filepath.Join(memoryDir, compactDate[:6], compactDate+".md")
		filesChecked[date.Format("2006-01-02")] = notePath
		if content := readOptionalTextFile(notePath); strings.TrimSpace(content) != "" {
			dailyNotes = append(dailyNotes, agentDailyNote{
				Date:    date.Format("2006-01-02"),
				Path:    notePath,
				Content: content,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agentMemoryResponse{
		AgentID:      agentID,
		Workspace:    workspace,
		MemoryPath:   memoryPath,
		LongTerm:     longTerm,
		DailyNotes:   dailyNotes,
		HumanNotes:   humanNotes,
		FilesChecked: filesChecked,
	})
}

func (h *Handler) handlePutAgentMemory(w http.ResponseWriter, r *http.Request) {
	agentID := strings.TrimSpace(r.PathValue("id"))
	if agentID == "" {
		http.Error(w, "missing agent id", http.StatusBadRequest)
		return
	}
	var request struct {
		LongTerm string `json:"long_term"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "invalid memory payload", http.StatusBadRequest)
		return
	}
	if decoder.More() {
		http.Error(w, "invalid memory payload", http.StatusBadRequest)
		return
	}
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}
	workspace, _, ok := resolveAgentMemoryWorkspace(cfg, agentID)
	if !ok {
		http.Error(w, "agent not found", http.StatusNotFound)
		return
	}
	memoryPath := filepath.Join(workspace, "memory", "MEMORY.md")
	if err := fileutil.WriteFileAtomic(memoryPath, []byte(request.LongTerm), 0o600); err != nil {
		http.Error(w, "failed to save memory", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "saved", "memory_path": memoryPath})
}

func resolveAgentMemoryWorkspace(cfg *config.Config, agentID string) (string, string, bool) {
	defaultWorkspace := cfg.WorkspacePath()
	if agentID == "main" {
		for _, agent := range cfg.Agents.List {
			if agent.ID == "main" {
				workspace := strings.TrimSpace(agent.Workspace)
				if workspace == "" {
					workspace = defaultWorkspace
				}
				humanNotes := ""
				if agent.Human != nil {
					humanNotes = agent.Human.MemoryNotes
				}
				return expandAgentWorkspace(workspace), humanNotes, true
			}
		}
		return expandAgentWorkspace(defaultWorkspace), "", true
	}

	for _, agent := range cfg.Agents.List {
		if agent.ID != agentID {
			continue
		}
		workspace := strings.TrimSpace(agent.Workspace)
		if workspace == "" {
			workspace = defaultWorkspace
		}
		humanNotes := ""
		if agent.Human != nil {
			humanNotes = agent.Human.MemoryNotes
		}
		return expandAgentWorkspace(workspace), humanNotes, true
	}
	return "", "", false
}

func expandAgentWorkspace(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	if path[0] != '~' {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if len(path) > 1 && path[1] == '/' {
		return home + path[1:]
	}
	return home
}

func readOptionalTextFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func summarizeHuman(value *config.HumanConfig) human {
	if value == nil {
		return human{}
	}
	return human{
		AgentName:      value.AgentName,
		Persona:        value.Persona,
		Tone:           value.Tone,
		DiscussionMode: value.DiscussionMode,
		MemoryNotes:    value.MemoryNotes,
		StatusStyle:    value.StatusStyle,
	}
}

func stringListOrEmpty(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return append([]string(nil), values...)
}

var agentIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func (h *Handler) handleCreateAgent(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	type createAgentRequest struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Model     string `json:"model"`
		Workspace string `json:"workspace"`
		ParentID  string `json:"parent_id"`
		Human     *human `json:"human"`
	}
	var req createAgentRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	req.ID = strings.TrimSpace(req.ID)
	req.Name = strings.TrimSpace(req.Name)
	req.Model = strings.TrimSpace(req.Model)
	req.Workspace = strings.TrimSpace(req.Workspace)
	req.ParentID = strings.TrimSpace(req.ParentID)
	if req.ParentID == "" {
		req.ParentID = "main"
	}
	if req.ID == "" {
		http.Error(w, "missing agent id", http.StatusBadRequest)
		return
	}
	if req.ID == "main" {
		http.Error(w, "agent id main is reserved", http.StatusBadRequest)
		return
	}
	if !agentIDPattern.MatchString(req.ID) {
		http.Error(w, "agent id can only contain letters, numbers, underscores, and hyphens", http.StatusBadRequest)
		return
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}

	for _, agent := range cfg.Agents.List {
		if agent.ID == req.ID {
			http.Error(w, "agent id already exists", http.StatusConflict)
			return
		}
	}

	agent := config.AgentConfig{
		ID:        req.ID,
		Name:      req.Name,
		Workspace: req.Workspace,
	}
	if req.Model != "" {
		agent.Model = &config.AgentModelConfig{Primary: req.Model}
	}
	agent.Human = cleanHuman(req.Human)
	if agent.Human == nil && req.Name != "" {
		agent.Human = &config.HumanConfig{AgentName: req.Name}
	} else if agent.Human != nil && agent.Human.AgentName == "" {
		if req.Name != "" {
			agent.Human.AgentName = req.Name
		} else {
			agent.Human.AgentName = req.ID
		}
	}
	cfg.Agents.List = append(cfg.Agents.List, agent)

	if err := allowSubagent(cfg, req.ParentID, req.ID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, fmt.Sprintf("failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "id": req.ID})
}

func allowSubagent(cfg *config.Config, parentID string, subagentID string) error {
	if parentID == "main" {
		for i := range cfg.Agents.List {
			if cfg.Agents.List[i].ID == "main" {
				appendAllowedSubagent(&cfg.Agents.List[i], subagentID)
				return nil
			}
		}
		main := config.AgentConfig{ID: "main", Name: "Main"}
		appendAllowedSubagent(&main, subagentID)
		cfg.Agents.List = append(cfg.Agents.List, main)
		return nil
	}

	for i := range cfg.Agents.List {
		if cfg.Agents.List[i].ID == parentID {
			appendAllowedSubagent(&cfg.Agents.List[i], subagentID)
			return nil
		}
	}
	return fmt.Errorf("parent agent not found")
}

func appendAllowedSubagent(agent *config.AgentConfig, subagentID string) {
	if agent.Subagents == nil {
		agent.Subagents = &config.SubagentsConfig{}
	}
	agent.Subagents.AllowAgents = cleanStringList(append(agent.Subagents.AllowAgents, subagentID))
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
		Human          *human   `json:"human"`
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
		if req.Human != nil {
			main := findOrCreateAgent(&cfg.Agents.List, "main")
			main.Name = "Main"
			main.Human = cleanHuman(req.Human)
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
		if req.Human != nil {
			cfg.Agents.List[index].Human = cleanHuman(req.Human)
		}
	}

	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, fmt.Sprintf("failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func findOrCreateAgent(agents *[]config.AgentConfig, id string) *config.AgentConfig {
	for i := range *agents {
		if (*agents)[i].ID == id {
			return &(*agents)[i]
		}
	}
	*agents = append(*agents, config.AgentConfig{ID: id})
	return &(*agents)[len(*agents)-1]
}

func cleanHuman(value *human) *config.HumanConfig {
	if value == nil {
		return nil
	}
	cleaned := &config.HumanConfig{
		AgentName:      strings.TrimSpace(value.AgentName),
		Persona:        strings.TrimSpace(value.Persona),
		Tone:           strings.TrimSpace(value.Tone),
		DiscussionMode: strings.TrimSpace(value.DiscussionMode),
		MemoryNotes:    strings.TrimSpace(value.MemoryNotes),
		StatusStyle:    strings.TrimSpace(value.StatusStyle),
	}
	if cleaned.AgentName == "" && cleaned.Persona == "" && cleaned.Tone == "" && cleaned.DiscussionMode == "" &&
		cleaned.MemoryNotes == "" && cleaned.StatusStyle == "" {
		return nil
	}
	return cleaned
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
