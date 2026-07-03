package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/jameclaw/pkg/config"
)

const onboardingStateFile = "onboarding_state.json"

type onboardingStepResponse struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Detail      string `json:"detail,omitempty"`
	Action      string `json:"action,omitempty"`
	ActionHref  string `json:"action_href,omitempty"`
}

type onboardingStatusResponse struct {
	Complete     bool                     `json:"complete"`
	ShouldShow   bool                     `json:"should_show"`
	CompletedAt  string                   `json:"completed_at,omitempty"`
	ConfigPath   string                   `json:"config_path"`
	Workspace    string                   `json:"workspace,omitempty"`
	Version      string                   `json:"version"`
	Steps        []onboardingStepResponse `json:"steps"`
	NextStepID   string                   `json:"next_step_id,omitempty"`
	ReadyCount   int                      `json:"ready_count"`
	TotalCount   int                      `json:"total_count"`
	ReadyForChat bool                     `json:"ready_for_chat"`
}

type onboardingState struct {
	Complete    bool   `json:"complete"`
	CompletedAt string `json:"completed_at,omitempty"`
}

func (h *Handler) registerOnboardingRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/onboarding/status", h.handleOnboardingStatus)
	mux.HandleFunc("POST /api/onboarding/complete", h.handleOnboardingComplete)
	mux.HandleFunc("POST /api/onboarding/reset", h.handleOnboardingReset)
}

func (h *Handler) handleOnboardingStatus(w http.ResponseWriter, r *http.Request) {
	status := h.onboardingStatus()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

func (h *Handler) handleOnboardingComplete(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC().Format(time.RFC3339)
	if err := saveOnboardingState(onboardingState{Complete: true, CompletedAt: now}); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save onboarding state: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "completed_at": now})
}

func (h *Handler) handleOnboardingReset(w http.ResponseWriter, r *http.Request) {
	if err := saveOnboardingState(onboardingState{}); err != nil {
		http.Error(w, fmt.Sprintf("Failed to reset onboarding state: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
}

func (h *Handler) onboardingStatus() onboardingStatusResponse {
	state, _ := loadOnboardingState()
	cfg, err := config.LoadConfig(h.configPath)
	var cfgErr string
	if err != nil {
		cfgErr = err.Error()
		cfg = nil
	}

	steps := []onboardingStepResponse{
		onboardingConfigStep(h.configPath, cfgErr),
		onboardingWorkspaceStep(cfg),
		onboardingModelStep(cfg),
		onboardingCredentialStep(cfg),
		h.onboardingGatewayStep(cfg),
		onboardingChannelStep(cfg),
		onboardingChatStep(cfg, h.gatewayStatusData()),
	}

	ready := 0
	nextStepID := ""
	for _, step := range steps {
		if step.Status == "ready" || step.Status == "optional" {
			ready++
			continue
		}
		if nextStepID == "" {
			nextStepID = step.ID
		}
	}

	workspace := ""
	if cfg != nil {
		workspace = cfg.WorkspacePath()
	}

	return onboardingStatusResponse{
		Complete:     state.Complete,
		ShouldShow:   !state.Complete && ready < len(steps),
		CompletedAt:  state.CompletedAt,
		ConfigPath:   h.configPath,
		Workspace:    workspace,
		Version:      config.GetVersion(),
		Steps:        steps,
		NextStepID:   nextStepID,
		ReadyCount:   ready,
		TotalCount:   len(steps),
		ReadyForChat: nextStepID == "",
	}
}

func onboardingConfigStep(configPath, cfgErr string) onboardingStepResponse {
	step := onboardingStepResponse{
		ID:          "config",
		Title:       "Configuration",
		Description: "Load the JameClaw config that powers the CLI and Web Console.",
		Action:      "Open config",
		ActionHref:  "/config",
	}
	if cfgErr != "" {
		step.Status = "blocked"
		step.Detail = cfgErr
		return step
	}
	step.Status = "ready"
	step.Detail = configPath
	return step
}

func onboardingWorkspaceStep(cfg *config.Config) onboardingStepResponse {
	step := onboardingStepResponse{
		ID:          "workspace",
		Title:       "Workspace",
		Description: "Confirm JameClaw has a workspace path for memory, skills, and agent files.",
		Action:      "Review config",
		ActionHref:  "/config",
	}
	if cfg == nil {
		step.Status = "blocked"
		step.Detail = "Configuration is not available."
		return step
	}
	workspace := strings.TrimSpace(cfg.WorkspacePath())
	if workspace == "" {
		step.Status = "blocked"
		step.Detail = "No workspace path is configured."
		return step
	}
	if info, err := os.Stat(workspace); err == nil && info.IsDir() {
		step.Status = "ready"
		step.Detail = workspace
		return step
	}
	parent := filepath.Dir(workspace)
	if info, err := os.Stat(parent); err == nil && info.IsDir() {
		step.Status = "attention"
		step.Detail = "Workspace will be created at " + workspace
		return step
	}
	step.Status = "blocked"
	step.Detail = "Workspace parent directory is missing: " + parent
	return step
}

func onboardingModelStep(cfg *config.Config) onboardingStepResponse {
	step := onboardingStepResponse{
		ID:          "model",
		Title:       "Model",
		Description: "Choose a default model for the agent.",
		Action:      "Choose model",
		ActionHref:  "/models",
	}
	if cfg == nil {
		step.Status = "blocked"
		step.Detail = "Configuration is not available."
		return step
	}
	defaultModel := strings.TrimSpace(cfg.Agents.Defaults.GetModelName())
	if defaultModel == "" {
		step.Status = "blocked"
		step.Detail = "No default model is selected."
		return step
	}
	for _, model := range cfg.ModelList {
		if model != nil && model.ModelName == defaultModel {
			step.Status = "ready"
			step.Detail = defaultModel
			return step
		}
	}
	step.Status = "attention"
	step.Detail = "Default model is set but no matching model entry was found."
	return step
}

func onboardingCredentialStep(cfg *config.Config) onboardingStepResponse {
	step := onboardingStepResponse{
		ID:          "credentials",
		Title:       "Credentials",
		Description: "Make sure at least one default or configured model can authenticate.",
		Action:      "Connect provider",
		ActionHref:  "/credentials",
	}
	if cfg == nil {
		step.Status = "blocked"
		step.Detail = "Configuration is not available."
		return step
	}
	if len(cfg.ModelList) == 0 {
		step.Status = "blocked"
		step.Detail = "No models are configured."
		return step
	}
	defaultModel := strings.TrimSpace(cfg.Agents.Defaults.GetModelName())
	for _, model := range cfg.ModelList {
		if model == nil || (defaultModel != "" && model.ModelName != defaultModel) {
			continue
		}
		if isModelConfigured(model) {
			step.Status = "ready"
			step.Detail = "Credential available for " + model.ModelName
			return step
		}
		step.Status = "blocked"
		step.Detail = "Default model exists but still needs credentials or a local runtime."
		return step
	}
	for _, model := range cfg.ModelList {
		if model != nil && isModelConfigured(model) {
			step.Status = "attention"
			step.Detail = "A model is configured, but the default model needs review."
			return step
		}
	}
	step.Status = "blocked"
	step.Detail = "No configured model credentials were found."
	return step
}

func (h *Handler) onboardingGatewayStep(cfg *config.Config) onboardingStepResponse {
	step := onboardingStepResponse{
		ID:          "gateway",
		Title:       "Gateway",
		Description: "Start or verify the local gateway before the Web Console sends agent messages.",
		Action:      "Start gateway",
		ActionHref:  "/",
	}
	data := h.gatewayStatusData()
	status, _ := data["gateway_status"].(string)
	if status == "running" {
		step.Status = "ready"
		if pid, ok := data["pid"]; ok {
			step.Detail = fmt.Sprintf("Running, pid %v", pid)
		} else {
			step.Detail = "Running"
		}
		return step
	}
	if reason, ok := data["gateway_start_reason"].(string); ok && strings.TrimSpace(reason) != "" {
		step.Status = "blocked"
		step.Detail = reason
		return step
	}
	if cfg != nil && cfg.Gateway.Port > 0 {
		if pid := processUsingPort(cfg.Gateway.Port); pid != "" {
			step.Status = "attention"
			step.Detail = fmt.Sprintf("Port %d is already in use by %s.", cfg.Gateway.Port, pid)
			return step
		}
	}
	step.Status = "attention"
	step.Detail = "Gateway is " + status + "."
	return step
}

func onboardingChannelStep(cfg *config.Config) onboardingStepResponse {
	step := onboardingStepResponse{
		ID:          "channels",
		Title:       "Channels",
		Description: "Review optional channels after the web chat path is ready.",
		Action:      "Review channels",
		ActionHref:  "/channels",
		Status:      "optional",
		Detail:      "Web Console chat is available without configuring an external channel.",
	}
	if cfg == nil {
		return step
	}
	if cfg.Channels.Telegram.Enabled || cfg.Channels.Discord.Enabled || cfg.Channels.Slack.Enabled {
		step.Status = "ready"
		step.Detail = "At least one external channel is enabled."
	}
	return step
}

func onboardingChatStep(cfg *config.Config, gateway map[string]any) onboardingStepResponse {
	step := onboardingStepResponse{
		ID:          "chat",
		Title:       "First agent test",
		Description: "Send a short test prompt to confirm onboarding produced a working agent loop.",
		Action:      "Open chat",
		ActionHref:  "/",
	}
	if cfg == nil {
		step.Status = "blocked"
		step.Detail = "Configuration is not available."
		return step
	}
	if strings.TrimSpace(cfg.Agents.Defaults.GetModelName()) == "" {
		step.Status = "blocked"
		step.Detail = "Choose a default model first."
		return step
	}
	if gateway["gateway_status"] != "running" {
		step.Status = "attention"
		step.Detail = "Start the gateway, then send a test message."
		return step
	}
	step.Status = "attention"
	step.Detail = "Ready for a smoke test. Send a message in Chat, then mark onboarding complete."
	return step
}

func loadOnboardingState() (onboardingState, error) {
	var state onboardingState
	path, err := onboardingStatePath()
	if err != nil {
		return state, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return state, err
	}
	return state, json.Unmarshal(data, &state)
}

func saveOnboardingState(state onboardingState) error {
	path, err := onboardingStatePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func onboardingStatePath() (string, error) {
	if home := strings.TrimSpace(os.Getenv("JAMECLAW_HOME")); home != "" {
		return filepath.Join(home, onboardingStateFile), nil
	}
	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(userHome, ".jameclaw", onboardingStateFile), nil
}

func processUsingPort(port int) string {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 250*time.Millisecond)
	if err != nil {
		return ""
	}
	_ = conn.Close()
	return "another process"
}
