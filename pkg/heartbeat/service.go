// JameClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 JameClaw contributors

package heartbeat

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/jameclaw/pkg/bus"
	"github.com/sipeed/jameclaw/pkg/constants"
	"github.com/sipeed/jameclaw/pkg/fileutil"
	"github.com/sipeed/jameclaw/pkg/logger"
	"github.com/sipeed/jameclaw/pkg/state"
	"github.com/sipeed/jameclaw/pkg/tools"
)

const (
	minIntervalMinutes     = 5
	defaultIntervalMinutes = 30
	userTasksMarker        = "Add your heartbeat tasks below this line:"
	initiativeDirectory    = "initiative"
	initiativeStateFile    = "state.json"
	initiativeHistoryFile  = "history.jsonl"
	maxInitiativeHistory   = 100
)

// InitiativeRecord is the durable result of an autonomous heartbeat check.
// It is intentionally local to the configured workspace and powers the
// desktop Team Grid's initiative status.
type InitiativeRecord struct {
	CheckedAt time.Time `json:"checked_at"`
	Status    string    `json:"status"`
	Summary   string    `json:"summary"`
	Channel   string    `json:"channel,omitempty"`
	ChatID    string    `json:"chat_id,omitempty"`
}

// HeartbeatHandler is the function type for handling heartbeat.
// It returns a ToolResult that can indicate async operations.
// channel and chatID are derived from the last active user channel.
type HeartbeatHandler func(prompt, channel, chatID string) *tools.ToolResult

// HeartbeatService manages periodic heartbeat checks
type HeartbeatService struct {
	workspace  string
	bus        *bus.MessageBus
	state      *state.Manager
	handler    HeartbeatHandler
	interval   time.Duration
	enabled    bool
	initiative bool
	running    bool
	mu         sync.RWMutex
	stopChan   chan struct{}
}

// NewHeartbeatService creates a new heartbeat service
func NewHeartbeatService(workspace string, intervalMinutes int, enabled bool) *HeartbeatService {
	return NewHeartbeatServiceWithInitiative(workspace, intervalMinutes, enabled, true)
}

// NewHeartbeatServiceWithInitiative creates a heartbeat service with explicit
// control over autonomous problem discovery. When initiative is false, the
// service retains the legacy task-list-only behavior.
func NewHeartbeatServiceWithInitiative(
	workspace string,
	intervalMinutes int,
	enabled bool,
	initiative bool,
) *HeartbeatService {
	// Apply minimum interval
	if intervalMinutes < minIntervalMinutes && intervalMinutes != 0 {
		intervalMinutes = minIntervalMinutes
	}

	if intervalMinutes == 0 {
		intervalMinutes = defaultIntervalMinutes
	}

	return &HeartbeatService{
		workspace:  workspace,
		interval:   time.Duration(intervalMinutes) * time.Minute,
		enabled:    enabled,
		initiative: initiative,
		state:      state.NewManager(workspace),
	}
}

// SetBus sets the message bus for delivering heartbeat results.
func (hs *HeartbeatService) SetBus(msgBus *bus.MessageBus) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	hs.bus = msgBus
}

// SetHandler sets the heartbeat handler.
func (hs *HeartbeatService) SetHandler(handler HeartbeatHandler) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	hs.handler = handler
}

// Start begins the heartbeat service
func (hs *HeartbeatService) Start() error {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	if hs.stopChan != nil {
		logger.InfoC("heartbeat", "Heartbeat service already running")
		return nil
	}

	if !hs.enabled {
		logger.InfoC("heartbeat", "Heartbeat service disabled")
		return nil
	}

	hs.stopChan = make(chan struct{})
	go hs.runLoop(hs.stopChan)

	logger.InfoCF("heartbeat", "Heartbeat service started", map[string]any{
		"interval_minutes": hs.interval.Minutes(),
	})

	return nil
}

// Stop gracefully stops the heartbeat service
func (hs *HeartbeatService) Stop() {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	if hs.stopChan == nil {
		return
	}

	logger.InfoC("heartbeat", "Stopping heartbeat service")
	close(hs.stopChan)
	hs.stopChan = nil
}

// IsRunning returns whether the service is running
func (hs *HeartbeatService) IsRunning() bool {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	return hs.stopChan != nil
}

// runLoop runs the heartbeat ticker
func (hs *HeartbeatService) runLoop(stopChan chan struct{}) {
	ticker := time.NewTicker(hs.interval)
	defer ticker.Stop()

	// Run first heartbeat after initial delay
	time.AfterFunc(time.Second, func() {
		hs.executeHeartbeat()
	})

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			hs.executeHeartbeat()
		}
	}
}

// executeHeartbeat performs a single heartbeat check
func (hs *HeartbeatService) executeHeartbeat() {
	hs.mu.Lock()
	enabled := hs.enabled
	handler := hs.handler
	if !hs.enabled || hs.stopChan == nil || hs.running {
		hs.mu.Unlock()
		return
	}
	hs.running = true
	hs.mu.Unlock()
	defer func() {
		hs.mu.Lock()
		hs.running = false
		hs.mu.Unlock()
	}()

	if !enabled {
		return
	}

	logger.DebugC("heartbeat", "Executing heartbeat")

	prompt := hs.buildPrompt()
	if prompt == "" {
		logger.InfoC("heartbeat", "No heartbeat prompt (HEARTBEAT.md empty or missing)")
		return
	}

	if handler == nil {
		hs.logErrorf("Heartbeat handler not configured")
		return
	}

	// Get last channel info for context
	lastChannel := hs.state.GetLastChannel()
	channel, chatID := hs.parseLastChannel(lastChannel)

	// Debug log for channel resolution
	hs.logInfof("Resolved channel: %s, chatID: %s (from lastChannel: %s)", channel, chatID, lastChannel)

	result := handler(prompt, channel, chatID)

	if result == nil {
		hs.logInfof("Heartbeat handler returned nil result")
		return
	}
	hs.recordInitiative(result, channel, chatID)

	// Handle different result types
	if result.IsError {
		hs.logErrorf("Heartbeat error: %s", result.ForLLM)
		return
	}

	if result.Async {
		hs.logInfof("Async task started: %s", result.ForLLM)
		logger.InfoCF("heartbeat", "Async heartbeat task started",
			map[string]any{
				"message": result.ForLLM,
			})
		return
	}

	// Check if silent
	if result.Silent {
		hs.logInfof("Heartbeat OK - silent")
		return
	}

	// Send result to user
	if result.ForUser != "" {
		hs.sendResponse(result.ForUser)
	} else if result.ForLLM != "" {
		hs.sendResponse(result.ForLLM)
	}

	hs.logInfof("Heartbeat completed: %s", result.ForLLM)
}

// buildPrompt builds the heartbeat prompt from HEARTBEAT.md
func (hs *HeartbeatService) buildPrompt() string {
	heartbeatPath := filepath.Join(hs.workspace, "HEARTBEAT.md")

	data, err := os.ReadFile(heartbeatPath)
	if err != nil {
		if os.IsNotExist(err) {
			hs.createDefaultHeartbeatTemplate()
			data = nil
		} else {
			hs.logErrorf("Error reading HEARTBEAT.md: %v", err)
			return ""
		}
	}

	content := string(data)
	hasUserTasks := heartbeatHasUserTasks(content)
	if !hasUserTasks && !hs.initiative {
		return ""
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	if !hs.initiative {
		return fmt.Sprintf(`# Heartbeat Check

Current time: %s

Review the user-authored recurring tasks below and execute necessary actions.
If nothing requires attention, respond ONLY with: HEARTBEAT_OK

%s
`, now, content)
	}
	previous := hs.previousInitiativeContext()
	if previous == "" {
		previous = "No previous initiative check has been recorded."
	}
	userTasks := content
	if !hasUserTasks {
		userTasks = "No user-authored recurring tasks are configured. Use the initiative policy below."
	}

	return fmt.Sprintf(`# Proactive Agent Check

Current time: %s

You are running without a new user prompt. Take initiative: inspect the workspace,
memory, recent work, automation state, and available health signals to find one
concrete problem or high-leverage improvement. Do not invent work merely to appear
busy. Use tools to gather evidence before acting.

## Initiative policy

1. First handle any explicit recurring tasks below.
2. Then independently look for unfinished, failing, stale, inconsistent, or blocked
   work that matters to the user's stated goals.
3. Choose at most ONE additional high-value issue per check. Prefer completing a
   small durable fix over producing a broad list of suggestions.
4. You may act without asking only when the action is local, reversible, scoped to
   the configured workspace, and supported by direct evidence. Examples: fix a
   clear code or documentation defect, run focused verification, repair a broken
   local workflow, organize an existing backlog, or create a missing test.
5. Do NOT autonomously send messages, publish/deploy, spend money, change accounts
   or credentials, alter security/privacy/network settings, install software,
   delete user data, rewrite broad areas, or make an external commitment. For any
   such action, investigate safely and return NEEDS_APPROVAL with the exact proposed
   action and evidence.
6. Respect existing work and dirty files. Never overwrite unrelated changes. Do
   not repeatedly solve the same issue; review the previous initiative result.
7. Review memory/self-improvement.json for repeated failures, stale candidates,
   and workflows worth proposing. Never approve a candidate, create a skill, or
   change approval/security behavior without the user's decision in Memory.
8. Timebox the check. If there is no evidence-backed useful action, respond ONLY:
   HEARTBEAT_OK
9. When you act or find a blocker, finish with this compact format:
   INITIATIVE_REPORT
   Problem: ...
   Evidence: ...
   Action: ...
   Verification: ...
   Next: ...

## Previous initiative result

%s

## User-authored recurring tasks

%s
`, now, previous, userTasks)
}

// createDefaultHeartbeatTemplate creates the default HEARTBEAT.md file
func (hs *HeartbeatService) createDefaultHeartbeatTemplate() {
	heartbeatPath := filepath.Join(hs.workspace, "HEARTBEAT.md")

	defaultContent := `# Heartbeat Check List

This file contains optional recurring tasks for the heartbeat service. When
initiative mode is enabled, Jame also inspects the workspace for one useful,
evidence-backed problem even when this list is empty.

## Examples

- Check for unread messages
- Review upcoming calendar events
- Check device status (e.g., MaixCam)

## Instructions

- Execute ALL tasks listed below. Do NOT skip any task.
- For simple tasks (e.g., report current time), respond directly.
- For complex tasks that may take time, use the spawn tool to create a subagent.
- The spawn tool is async - subagent results will be sent to the user automatically.
- After spawning a subagent, CONTINUE to process remaining tasks.
- Only respond with HEARTBEAT_OK when ALL tasks are done AND nothing needs attention.
- Initiative work stays local and reversible. External, destructive, financial,
  credential, security, account, publishing, and deployment actions require approval.

---

Add your heartbeat tasks below this line:
`

	if err := fileutil.WriteFileAtomic(heartbeatPath, []byte(defaultContent), 0o644); err != nil {
		hs.logErrorf("Failed to create default HEARTBEAT.md: %v", err)
	} else {
		hs.logInfof("Created default HEARTBEAT.md template")
	}
}

func (hs *HeartbeatService) previousInitiativeContext() string {
	record, err := LoadInitiativeState(hs.workspace)
	if err != nil || record.CheckedAt.IsZero() {
		return ""
	}
	return fmt.Sprintf(
		"Checked: %s\nStatus: %s\nSummary:\n%s",
		record.CheckedAt.Format(time.RFC3339),
		record.Status,
		truncateInitiativeSummary(record.Summary, 3000),
	)
}

func (hs *HeartbeatService) recordInitiative(result *tools.ToolResult, channel, chatID string) {
	if result == nil {
		return
	}
	summary := strings.TrimSpace(result.ForUser)
	if summary == "" {
		summary = strings.TrimSpace(result.ForLLM)
	}
	status := "completed"
	switch {
	case result.IsError:
		status = "error"
	case result.Async:
		status = "working"
	case result.Silent && strings.EqualFold(summary, "Heartbeat OK"):
		status = "idle"
	case strings.Contains(strings.ToUpper(summary), "NEEDS_APPROVAL"):
		status = "needs_approval"
	}
	record := InitiativeRecord{
		CheckedAt: time.Now(),
		Status:    status,
		Summary:   truncateInitiativeSummary(summary, 12000),
		Channel:   channel,
		ChatID:    chatID,
	}
	if err := SaveInitiativeRecord(hs.workspace, record); err != nil {
		hs.logErrorf("Failed to save initiative result: %v", err)
	}
}

func truncateInitiativeSummary(value string, max int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max]) + "..."
}

// SaveInitiativeRecord updates the latest state and keeps a bounded history of
// actionable checks. Idle checks update freshness without spamming history.
func SaveInitiativeRecord(workspace string, record InitiativeRecord) error {
	dir := filepath.Join(workspace, initiativeDirectory)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	if err := fileutil.WriteFileAtomic(filepath.Join(dir, initiativeStateFile), data, 0o600); err != nil {
		return err
	}
	if record.Status == "idle" {
		return nil
	}
	history, err := LoadInitiativeHistory(workspace, maxInitiativeHistory-1)
	if err != nil {
		return err
	}
	for left, right := 0, len(history)-1; left < right; left, right = left+1, right-1 {
		history[left], history[right] = history[right], history[left]
	}
	history = append(history, record)
	if len(history) > maxInitiativeHistory {
		history = history[len(history)-maxInitiativeHistory:]
	}
	var builder strings.Builder
	for _, item := range history {
		line, marshalErr := json.Marshal(item)
		if marshalErr != nil {
			return marshalErr
		}
		builder.Write(line)
		builder.WriteByte('\n')
	}
	return fileutil.WriteFileAtomic(
		filepath.Join(dir, initiativeHistoryFile),
		[]byte(builder.String()),
		0o600,
	)
}

// LoadInitiativeState returns the latest autonomous check result.
func LoadInitiativeState(workspace string) (InitiativeRecord, error) {
	data, err := os.ReadFile(filepath.Join(workspace, initiativeDirectory, initiativeStateFile))
	if err != nil {
		return InitiativeRecord{}, err
	}
	var record InitiativeRecord
	err = json.Unmarshal(data, &record)
	return record, err
}

// LoadInitiativeHistory returns newest-first initiative results.
func LoadInitiativeHistory(workspace string, limit int) ([]InitiativeRecord, error) {
	data, err := os.ReadFile(filepath.Join(workspace, initiativeDirectory, initiativeHistoryFile))
	if err != nil {
		if os.IsNotExist(err) {
			return []InitiativeRecord{}, nil
		}
		return nil, err
	}
	items := make([]InitiativeRecord, 0)
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var record InitiativeRecord
		if json.Unmarshal([]byte(line), &record) == nil {
			items = append(items, record)
		}
	}
	if limit > 0 && len(items) > limit {
		items = items[len(items)-limit:]
	}
	for left, right := 0, len(items)-1; left < right; left, right = left+1, right-1 {
		items[left], items[right] = items[right], items[left]
	}
	return items, nil
}

func heartbeatHasUserTasks(content string) bool {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return false
	}

	markerIdx := strings.Index(content, userTasksMarker)
	if markerIdx < 0 {
		return true
	}

	tasksSection := content[markerIdx+len(userTasksMarker):]
	for _, line := range strings.Split(tasksSection, "\n") {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}
		if strings.HasPrefix(trimmedLine, "#") {
			continue
		}
		return true
	}

	return false
}

// sendResponse sends the heartbeat response to the last channel
func (hs *HeartbeatService) sendResponse(response string) {
	hs.mu.RLock()
	msgBus := hs.bus
	hs.mu.RUnlock()

	if msgBus == nil {
		hs.logInfof("No message bus configured, heartbeat result not sent")
		return
	}

	// Get last channel from state
	lastChannel := hs.state.GetLastChannel()
	if lastChannel == "" {
		hs.logInfof("No last channel recorded, heartbeat result not sent")
		return
	}

	platform, userID := hs.parseLastChannel(lastChannel)

	// Skip internal channels that can't receive messages
	if platform == "" || userID == "" {
		return
	}

	pubCtx, pubCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pubCancel()
	msgBus.PublishOutbound(pubCtx, bus.OutboundMessage{
		Channel: platform,
		ChatID:  userID,
		Content: response,
	})

	hs.logInfof("Heartbeat result sent to %s", platform)
}

// parseLastChannel parses the last channel string into platform and userID.
// Returns empty strings for invalid or internal channels.
func (hs *HeartbeatService) parseLastChannel(lastChannel string) (platform, userID string) {
	if lastChannel == "" {
		return "", ""
	}

	// Parse channel format: "platform:user_id" (e.g., "telegram:123456")
	parts := strings.SplitN(lastChannel, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		hs.logErrorf("Invalid last channel format: %s", lastChannel)
		return "", ""
	}

	platform, userID = parts[0], parts[1]

	// Skip internal channels
	if constants.IsInternalChannel(platform) {
		hs.logInfof("Skipping internal channel: %s", platform)
		return "", ""
	}

	return platform, userID
}

// logInfof logs an informational message to the heartbeat log
func (hs *HeartbeatService) logInfof(format string, args ...any) {
	hs.logf("INFO", format, args...)
}

// logErrorf logs an error message to the heartbeat log
func (hs *HeartbeatService) logErrorf(format string, args ...any) {
	hs.logf("ERROR", format, args...)
}

// logf writes a message to the heartbeat log file
func (hs *HeartbeatService) logf(level, format string, args ...any) {
	logFile := filepath.Join(hs.workspace, "heartbeat.log")
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(f, "[%s] [%s] %s\n", timestamp, level, fmt.Sprintf(format, args...))
}
