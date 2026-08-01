package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/pkg/fileutil"
)

const teamOperationsVersion = 1

var (
	errTeamOperationsValidation = errors.New("invalid team operation")
	errTeamOperationsConflict   = errors.New("team operation conflict")
)

type teamGoal struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Outcome     string `json:"outcome"`
	LeadAgentID string `json:"lead_agent_id"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type teamTask struct {
	ID                 string   `json:"id"`
	GoalID             string   `json:"goal_id"`
	Title              string   `json:"title"`
	Description        string   `json:"description,omitempty"`
	OwnerAgentID       string   `json:"owner_agent_id,omitempty"`
	Status             string   `json:"status"`
	DependsOn          []string `json:"depends_on"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	FileScopes         []string `json:"file_scopes"`
	TimeBudgetMinutes  int      `json:"time_budget_minutes"`
	TokenBudget        int64    `json:"token_budget"`
	Result             string   `json:"result,omitempty"`
	Verification       string   `json:"verification,omitempty"`
	BlockedReason      string   `json:"blocked_reason,omitempty"`
	CreatedAt          string   `json:"created_at"`
	UpdatedAt          string   `json:"updated_at"`
}

type teamOperationsSnapshot struct {
	Version   int        `json:"version"`
	Goal      *teamGoal  `json:"goal,omitempty"`
	Tasks     []teamTask `json:"tasks"`
	UpdatedAt string     `json:"updated_at"`
}

func (h *Handler) registerTeamOperationsRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/agents/team-operations", h.handleGetTeamOperations)
	mux.HandleFunc("PUT /api/agents/team-operations/goal", h.handlePutTeamGoal)
	mux.HandleFunc("POST /api/agents/team-operations/tasks", h.handleCreateTeamTask)
	mux.HandleFunc("PATCH /api/agents/team-operations/tasks/{taskID}", h.handlePatchTeamTask)
	mux.HandleFunc("POST /api/agents/team-operations/tasks/{taskID}/action", h.handleTeamTaskAction)
}

func (h *Handler) handleGetTeamOperations(w http.ResponseWriter, _ *http.Request) {
	h.teamOperationsMu.Lock()
	defer h.teamOperationsMu.Unlock()
	_, path, err := h.teamOperationsWorkspace()
	if err != nil {
		http.Error(w, "failed to load team workspace", http.StatusInternalServerError)
		return
	}
	snapshot, err := loadTeamOperations(path)
	if err != nil {
		http.Error(w, "failed to load team operations", http.StatusInternalServerError)
		return
	}
	writeTeamOperationsJSON(w, snapshot)
}

func (h *Handler) handlePutTeamGoal(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Title       string `json:"title"`
		Outcome     string `json:"outcome"`
		LeadAgentID string `json:"lead_agent_id"`
		Status      string `json:"status"`
	}
	if err := decodeTeamOperationsRequest(w, r, &request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.teamOperationsMu.Lock()
	defer h.teamOperationsMu.Unlock()
	cfg, path, err := h.teamOperationsWorkspace()
	if err != nil {
		http.Error(w, "failed to load team workspace", http.StatusInternalServerError)
		return
	}
	if strings.TrimSpace(request.Title) == "" || strings.TrimSpace(request.Outcome) == "" {
		http.Error(w, "goal title and measurable outcome are required", http.StatusUnprocessableEntity)
		return
	}
	leadID := strings.TrimSpace(request.LeadAgentID)
	if leadID == "" {
		leadID = "main"
	}
	if !teamAgentExists(cfg, leadID) {
		http.Error(w, "lead agent not found", http.StatusUnprocessableEntity)
		return
	}
	status := strings.TrimSpace(request.Status)
	if status == "" {
		status = "active"
	}
	if status != "active" && status != "paused" && status != "done" {
		http.Error(w, "goal status must be active, paused, or done", http.StatusUnprocessableEntity)
		return
	}
	snapshot, err := loadTeamOperations(path)
	if err != nil {
		http.Error(w, "failed to load team operations", http.StatusInternalServerError)
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	createdAt := now
	goalID := newTeamOperationsID("goal")
	if snapshot.Goal != nil {
		createdAt = snapshot.Goal.CreatedAt
		goalID = snapshot.Goal.ID
	}
	snapshot.Goal = &teamGoal{
		ID:          goalID,
		Title:       strings.TrimSpace(request.Title),
		Outcome:     strings.TrimSpace(request.Outcome),
		LeadAgentID: leadID,
		Status:      status,
		CreatedAt:   createdAt,
		UpdatedAt:   now,
	}
	snapshot.UpdatedAt = now
	if err := saveTeamOperations(path, snapshot); err != nil {
		http.Error(w, "failed to save team goal", http.StatusInternalServerError)
		return
	}
	writeTeamOperationsJSON(w, snapshot)
}

func (h *Handler) handleCreateTeamTask(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Title              string   `json:"title"`
		Description        string   `json:"description"`
		OwnerAgentID       string   `json:"owner_agent_id"`
		DependsOn          []string `json:"depends_on"`
		AcceptanceCriteria []string `json:"acceptance_criteria"`
		FileScopes         []string `json:"file_scopes"`
		TimeBudgetMinutes  int      `json:"time_budget_minutes"`
		TokenBudget        int64    `json:"token_budget"`
	}
	if err := decodeTeamOperationsRequest(w, r, &request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.teamOperationsMu.Lock()
	defer h.teamOperationsMu.Unlock()
	cfg, path, err := h.teamOperationsWorkspace()
	if err != nil {
		http.Error(w, "failed to load team workspace", http.StatusInternalServerError)
		return
	}
	snapshot, err := loadTeamOperations(path)
	if err != nil {
		http.Error(w, "failed to load team operations", http.StatusInternalServerError)
		return
	}
	if snapshot.Goal == nil || snapshot.Goal.Status == "done" {
		http.Error(w, "create an active team goal before adding tasks", http.StatusUnprocessableEntity)
		return
	}
	if strings.TrimSpace(request.Title) == "" {
		http.Error(w, "task title is required", http.StatusUnprocessableEntity)
		return
	}
	ownerID := strings.TrimSpace(request.OwnerAgentID)
	if ownerID != "" && !teamAgentExists(cfg, ownerID) {
		http.Error(w, "task owner agent not found", http.StatusUnprocessableEntity)
		return
	}
	if err := validateTeamDependencies(snapshot.Tasks, "", request.DependsOn); err != nil {
		writeTeamOperationsError(w, err)
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	status := "unassigned"
	if ownerID != "" {
		status = "planned"
	}
	task := teamTask{
		ID:                 newTeamOperationsID("task"),
		GoalID:             snapshot.Goal.ID,
		Title:              strings.TrimSpace(request.Title),
		Description:        strings.TrimSpace(request.Description),
		OwnerAgentID:       ownerID,
		Status:             status,
		DependsOn:          cleanTeamStrings(request.DependsOn),
		AcceptanceCriteria: cleanTeamStrings(request.AcceptanceCriteria),
		FileScopes:         cleanTeamStrings(request.FileScopes),
		TimeBudgetMinutes:  max(0, request.TimeBudgetMinutes),
		TokenBudget:        max(0, request.TokenBudget),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	snapshot.Tasks = append(snapshot.Tasks, task)
	snapshot.UpdatedAt = now
	if err := saveTeamOperations(path, snapshot); err != nil {
		http.Error(w, "failed to save team task", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeTeamOperationsJSON(w, task)
}

func (h *Handler) handlePatchTeamTask(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Title              *string   `json:"title"`
		Description        *string   `json:"description"`
		OwnerAgentID       *string   `json:"owner_agent_id"`
		DependsOn          *[]string `json:"depends_on"`
		AcceptanceCriteria *[]string `json:"acceptance_criteria"`
		FileScopes         *[]string `json:"file_scopes"`
		TimeBudgetMinutes  *int      `json:"time_budget_minutes"`
		TokenBudget        *int64    `json:"token_budget"`
	}
	if err := decodeTeamOperationsRequest(w, r, &request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.teamOperationsMu.Lock()
	defer h.teamOperationsMu.Unlock()
	cfg, path, err := h.teamOperationsWorkspace()
	if err != nil {
		http.Error(w, "failed to load team workspace", http.StatusInternalServerError)
		return
	}
	snapshot, err := loadTeamOperations(path)
	if err != nil {
		http.Error(w, "failed to load team operations", http.StatusInternalServerError)
		return
	}
	index := teamTaskIndex(snapshot.Tasks, r.PathValue("taskID"))
	if index < 0 {
		http.Error(w, "team task not found", http.StatusNotFound)
		return
	}
	task := &snapshot.Tasks[index]
	if task.Status == "working" || task.Status == "review" || task.Status == "done" {
		http.Error(w, "pause or reopen the task before editing its contract", http.StatusConflict)
		return
	}
	if request.Title != nil {
		if strings.TrimSpace(*request.Title) == "" {
			http.Error(w, "task title cannot be empty", http.StatusUnprocessableEntity)
			return
		}
		task.Title = strings.TrimSpace(*request.Title)
	}
	if request.Description != nil {
		task.Description = strings.TrimSpace(*request.Description)
	}
	if request.OwnerAgentID != nil {
		ownerID := strings.TrimSpace(*request.OwnerAgentID)
		if ownerID != "" && !teamAgentExists(cfg, ownerID) {
			http.Error(w, "task owner agent not found", http.StatusUnprocessableEntity)
			return
		}
		task.OwnerAgentID = ownerID
		if ownerID == "" {
			task.Status = "unassigned"
		} else {
			task.Status = "planned"
		}
	}
	if request.DependsOn != nil {
		if err := validateTeamDependencies(snapshot.Tasks, task.ID, *request.DependsOn); err != nil {
			writeTeamOperationsError(w, err)
			return
		}
		task.DependsOn = cleanTeamStrings(*request.DependsOn)
	}
	if request.AcceptanceCriteria != nil {
		task.AcceptanceCriteria = cleanTeamStrings(*request.AcceptanceCriteria)
	}
	if request.FileScopes != nil {
		task.FileScopes = cleanTeamStrings(*request.FileScopes)
	}
	if request.TimeBudgetMinutes != nil {
		task.TimeBudgetMinutes = max(0, *request.TimeBudgetMinutes)
	}
	if request.TokenBudget != nil {
		task.TokenBudget = max(0, *request.TokenBudget)
	}
	task.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	snapshot.UpdatedAt = task.UpdatedAt
	if err := saveTeamOperations(path, snapshot); err != nil {
		http.Error(w, "failed to save team task", http.StatusInternalServerError)
		return
	}
	writeTeamOperationsJSON(w, task)
}

func (h *Handler) handleTeamTaskAction(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Action        string `json:"action"`
		Result        string `json:"result"`
		Verification  string `json:"verification"`
		BlockedReason string `json:"blocked_reason"`
	}
	if err := decodeTeamOperationsRequest(w, r, &request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.teamOperationsMu.Lock()
	defer h.teamOperationsMu.Unlock()
	_, path, err := h.teamOperationsWorkspace()
	if err != nil {
		http.Error(w, "failed to load team workspace", http.StatusInternalServerError)
		return
	}
	snapshot, err := loadTeamOperations(path)
	if err != nil {
		http.Error(w, "failed to load team operations", http.StatusInternalServerError)
		return
	}
	index := teamTaskIndex(snapshot.Tasks, r.PathValue("taskID"))
	if index < 0 {
		http.Error(w, "team task not found", http.StatusNotFound)
		return
	}
	if err := applyTeamTaskAction(&snapshot, index, strings.TrimSpace(request.Action), request.Result, request.Verification, request.BlockedReason); err != nil {
		writeTeamOperationsError(w, err)
		return
	}
	if err := saveTeamOperations(path, snapshot); err != nil {
		http.Error(w, "failed to save team task", http.StatusInternalServerError)
		return
	}
	writeTeamOperationsJSON(w, snapshot.Tasks[index])
}

func applyTeamTaskAction(snapshot *teamOperationsSnapshot, index int, action, result, verification, blockedReason string) error {
	task := &snapshot.Tasks[index]
	now := time.Now().UTC().Format(time.RFC3339)
	switch action {
	case "start":
		if task.OwnerAgentID == "" {
			return fmt.Errorf("%w: assign an owner before starting", errTeamOperationsValidation)
		}
		if task.Status != "planned" && task.Status != "paused" && task.Status != "blocked" {
			return fmt.Errorf("%w: only planned, paused, or blocked tasks can start", errTeamOperationsConflict)
		}
		for _, dependencyID := range task.DependsOn {
			dependencyIndex := teamTaskIndex(snapshot.Tasks, dependencyID)
			if dependencyIndex < 0 || snapshot.Tasks[dependencyIndex].Status != "done" {
				return fmt.Errorf("%w: dependency %s is not done", errTeamOperationsConflict, dependencyID)
			}
		}
		for otherIndex := range snapshot.Tasks {
			if otherIndex == index || snapshot.Tasks[otherIndex].Status != "working" {
				continue
			}
			if scope := conflictingTeamFileScope(task.FileScopes, snapshot.Tasks[otherIndex].FileScopes); scope != "" {
				return fmt.Errorf("%w: file scope %s is already owned by %s", errTeamOperationsConflict, scope, snapshot.Tasks[otherIndex].Title)
			}
		}
		task.Status = "working"
		task.BlockedReason = ""
	case "submit_review":
		if task.Status != "working" {
			return fmt.Errorf("%w: only working tasks can be submitted for review", errTeamOperationsConflict)
		}
		result = strings.TrimSpace(result)
		if result == "" {
			return fmt.Errorf("%w: a concrete result is required for review", errTeamOperationsValidation)
		}
		task.Result = result
		task.Status = "review"
	case "complete":
		if task.Status != "review" {
			return fmt.Errorf("%w: only tasks in review can be completed", errTeamOperationsConflict)
		}
		verification = strings.TrimSpace(verification)
		if verification == "" {
			return fmt.Errorf("%w: verification evidence is required", errTeamOperationsValidation)
		}
		task.Verification = verification
		task.Status = "done"
	case "block":
		blockedReason = strings.TrimSpace(blockedReason)
		if blockedReason == "" {
			return fmt.Errorf("%w: blocked reason is required", errTeamOperationsValidation)
		}
		task.BlockedReason = blockedReason
		task.Status = "blocked"
	case "pause":
		if task.Status == "done" {
			return fmt.Errorf("%w: completed tasks cannot be paused", errTeamOperationsConflict)
		}
		task.Status = "paused"
	case "reopen":
		task.Status = "planned"
		task.Result = ""
		task.Verification = ""
		task.BlockedReason = ""
	default:
		return fmt.Errorf("%w: unsupported task action %q", errTeamOperationsValidation, action)
	}
	task.UpdatedAt = now
	snapshot.UpdatedAt = now
	if snapshot.Goal != nil {
		if action == "start" || action == "reopen" {
			snapshot.Goal.Status = "active"
		}
		if action == "complete" && len(snapshot.Tasks) > 0 {
			allDone := true
			for _, candidate := range snapshot.Tasks {
				if candidate.GoalID == snapshot.Goal.ID && candidate.Status != "done" {
					allDone = false
					break
				}
			}
			if allDone {
				snapshot.Goal.Status = "done"
			}
		}
		snapshot.Goal.UpdatedAt = now
	}
	return nil
}

func (h *Handler) teamOperationsWorkspace() (*config.Config, string, error) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		return nil, "", err
	}
	workspace, _, ok := resolveAgentMemoryWorkspace(cfg, "main")
	if !ok || strings.TrimSpace(workspace) == "" {
		return nil, "", fmt.Errorf("main agent workspace not found")
	}
	return cfg, filepath.Join(workspace, "state", "team-operations.json"), nil
}

func loadTeamOperations(path string) (teamOperationsSnapshot, error) {
	snapshot := teamOperationsSnapshot{Version: teamOperationsVersion, Tasks: []teamTask{}}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return snapshot, nil
	}
	if err != nil {
		return teamOperationsSnapshot{}, err
	}
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return teamOperationsSnapshot{}, err
	}
	if snapshot.Version == 0 {
		snapshot.Version = teamOperationsVersion
	}
	if snapshot.Tasks == nil {
		snapshot.Tasks = []teamTask{}
	}
	return snapshot, nil
}

func saveTeamOperations(path string, snapshot teamOperationsSnapshot) error {
	raw, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	return fileutil.WriteFileAtomic(path, raw, 0o600)
}

func validateTeamDependencies(tasks []teamTask, taskID string, dependencyIDs []string) error {
	dependencies := cleanTeamStrings(dependencyIDs)
	for _, dependencyID := range dependencies {
		if dependencyID == taskID && taskID != "" {
			return fmt.Errorf("%w: a task cannot depend on itself", errTeamOperationsValidation)
		}
		if teamTaskIndex(tasks, dependencyID) < 0 {
			return fmt.Errorf("%w: dependency %s does not exist", errTeamOperationsValidation, dependencyID)
		}
		if taskID != "" && teamDependencyReaches(tasks, dependencyID, taskID, map[string]bool{}) {
			return fmt.Errorf("%w: dependency would create a cycle", errTeamOperationsValidation)
		}
	}
	return nil
}

func teamDependencyReaches(tasks []teamTask, currentID, targetID string, seen map[string]bool) bool {
	if currentID == targetID {
		return true
	}
	if seen[currentID] {
		return false
	}
	seen[currentID] = true
	index := teamTaskIndex(tasks, currentID)
	if index < 0 {
		return false
	}
	for _, dependencyID := range tasks[index].DependsOn {
		if teamDependencyReaches(tasks, dependencyID, targetID, seen) {
			return true
		}
	}
	return false
}

func conflictingTeamFileScope(left, right []string) string {
	for _, leftScope := range cleanTeamStrings(left) {
		leftScope = filepath.Clean(leftScope)
		for _, rightScope := range cleanTeamStrings(right) {
			rightScope = filepath.Clean(rightScope)
			if leftScope == rightScope || strings.HasPrefix(leftScope, rightScope+string(filepath.Separator)) || strings.HasPrefix(rightScope, leftScope+string(filepath.Separator)) {
				return leftScope
			}
		}
	}
	return ""
}

func teamAgentExists(cfg *config.Config, agentID string) bool {
	if agentID == "main" {
		return true
	}
	for _, agent := range cfg.Agents.List {
		if agent.ID == agentID {
			return true
		}
	}
	return false
}

func teamTaskIndex(tasks []teamTask, id string) int {
	for i := range tasks {
		if tasks[i].ID == id {
			return i
		}
	}
	return -1
}

func cleanTeamStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func newTeamOperationsID(prefix string) string {
	buffer := make([]byte, 8)
	if _, err := rand.Read(buffer); err == nil {
		return prefix + "-" + hex.EncodeToString(buffer)
	}
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func decodeTeamOperationsRequest(w http.ResponseWriter, r *http.Request, target any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid team operations payload: %w", err)
	}
	return nil
}

func writeTeamOperationsError(w http.ResponseWriter, err error) {
	status := http.StatusUnprocessableEntity
	if errors.Is(err, errTeamOperationsConflict) {
		status = http.StatusConflict
	}
	http.Error(w, err.Error(), status)
}

func writeTeamOperationsJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
