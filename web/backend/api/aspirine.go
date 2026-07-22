package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/pkg/providers"
)

type aspirineIssue struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Severity       string   `json:"severity"`
	Status         string   `json:"status"`
	Description    string   `json:"description"`
	Suggestion     string   `json:"suggestion"`
	RecoveryPrompt string   `json:"recovery_prompt,omitempty"`
	AutoFixAction  string   `json:"auto_fix_action,omitempty"`
	AutoFixLabel   string   `json:"auto_fix_label,omitempty"`
	Affected       []string `json:"affected,omitempty"`
	LastObservedAt string   `json:"last_observed_at"`
}

type aspirineSummary struct {
	Status        string          `json:"status"`
	IssueCount    int             `json:"issue_count"`
	CriticalCount int             `json:"critical_count"`
	WarningCount  int             `json:"warning_count"`
	CheckedAt     string          `json:"checked_at"`
	Issues        []aspirineIssue `json:"issues"`
}

func (h *Handler) registerAspirineRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/aspirine", h.handleAspirineSummary)
	mux.HandleFunc("POST /api/aspirine/actions/{action}", h.handleAspirineAction)
}

func (h *Handler) handleAspirineSummary(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, h.aspirineSummary())
}

func (h *Handler) aspirineSummary() aspirineSummary {
	checkedAt := time.Now().UTC().Format(time.RFC3339)
	issues := []aspirineIssue{}

	cfg, cfgErr := config.LoadConfig(h.configPath)
	if cfgErr != nil {
		issues = append(issues, aspirineIssue{
			ID:             "config-load-failed",
			Title:          "Configuration cannot be loaded",
			Severity:       "critical",
			Status:         "needs_attention",
			Description:    cfgErr.Error(),
			Suggestion:     "Open Configuration and fix the invalid config before starting services.",
			LastObservedAt: checkedAt,
		})
		return buildAspirineSummary(checkedAt, issues)
	}

	statusData := h.gatewayStatusData()
	gatewayStatus, _ := statusData["gateway_status"].(string)
	startAllowed, _ := statusData["gateway_start_allowed"].(bool)
	startReason, _ := statusData["gateway_start_reason"].(string)
	restartRequired, _ := statusData["gateway_restart_required"].(bool)
	enabledChannels := enabledChannelNames(cfg.Channels)

	if gatewayStatus == "" {
		gatewayStatus = "unknown"
	}

	switch gatewayStatus {
	case "running":
		if restartRequired {
			issues = append(issues, aspirineIssue{
				ID:             "gateway-restart-required",
				Title:          "Gateway needs a restart",
				Severity:       "warning",
				Status:         "can_auto_fix",
				Description:    "The selected default model changed after the gateway started.",
				Suggestion:     "Restart the gateway so new model settings are active.",
				AutoFixAction:  "restart_gateway",
				AutoFixLabel:   "Restart gateway",
				LastObservedAt: checkedAt,
			})
		}
	case "stopped", "error", "unknown":
		issue := aspirineIssue{
			ID:             "gateway-not-running",
			Title:          "Gateway is not running",
			Severity:       "critical",
			Status:         "needs_attention",
			Description:    "Messages from web console and enabled chat channels cannot be handled until the gateway is running.",
			Suggestion:     "Start the gateway after fixing any missing model or credential setup.",
			Affected:       enabledChannels,
			LastObservedAt: checkedAt,
		}
		if startAllowed {
			issue.Status = "can_auto_fix"
			issue.AutoFixAction = "start_gateway"
			issue.AutoFixLabel = "Start gateway"
			issue.Suggestion = "Start the gateway now."
		} else if startReason != "" {
			issue.Description = fmt.Sprintf("%s Start is blocked because %s.", issue.Description, startReason)
		}
		issues = append(issues, issue)
	case "starting", "restarting":
		issues = append(issues, aspirineIssue{
			ID:             "gateway-transitioning",
			Title:          "Gateway is still starting",
			Severity:       "warning",
			Status:         "monitoring",
			Description:    "The gateway is in a transition state.",
			Suggestion:     "Wait a few seconds. Aspirine will refresh automatically.",
			LastObservedAt: checkedAt,
		})
	}

	if len(enabledChannels) > 0 && gatewayStatus != "running" {
		issues = append(issues, aspirineIssue{
			ID:             "channels-waiting-for-gateway",
			Title:          "Enabled channels are waiting for the gateway",
			Severity:       "warning",
			Status:         "blocked",
			Description:    "Enabled channels cannot respond while the gateway is unavailable.",
			Suggestion:     "Bring the gateway online first, then test the affected channels.",
			Affected:       enabledChannels,
			LastObservedAt: checkedAt,
		})
	}

	if issue, ok := recentChannelLogIssue("telegram", "Telegram may not be responding", checkedAt); ok {
		issues = append(issues, issue)
	}

	issues = append(issues, recentConversationRecoveryIssues(h.listAllSessions(), checkedAt)...)

	if len(issues) == 0 {
		issues = append(issues, aspirineIssue{
			ID:             "system-healthy",
			Title:          "No active problems detected",
			Severity:       "info",
			Status:         "healthy",
			Description:    "Gateway and configured services do not show actionable problems right now.",
			Suggestion:     "Aspirine will keep checking automatically while this page is open.",
			LastObservedAt: checkedAt,
		})
	}

	return buildAspirineSummary(checkedAt, issues)
}

// recentConversationRecoveryIssues identifies clear corrective feedback that
// follows an assistant response. It intentionally uses a small, conservative
// phrase set: Aspirine should suggest a recovery, not pretend to know a user's
// sentiment from ordinary questions.
func recentConversationRecoveryIssues(sessions []sessionFile, checkedAt string) []aspirineIssue {
	issues := make([]aspirineIssue, 0, 3)
	for _, sess := range sessions {
		if len(issues) == 3 {
			break
		}
		feedback := latestCorrectiveFeedback(sess.Messages)
		if feedback == "" {
			continue
		}
		issues = append(issues, aspirineIssue{
			ID:             "conversation-recovery-" + sessionIDForKey(sess.Key),
			Title:          "The user may be unhappy with a recent answer",
			Severity:       "warning",
			Status:         "needs_follow_up",
			Description:    "Recent user feedback: “" + truncateRunes(feedback, 280) + "”",
			Suggestion:     "Follow up in the same conversation: acknowledge the gap, state what was missed, then give a corrected answer or take the requested action.",
			RecoveryPrompt: "I’m sorry that missed the mark. I understand the issue is: " + truncateRunes(feedback, 180) + ". I’ll correct it now by checking the earlier request and giving you a concrete improved result.",
			Affected:       []string{sessionIDForKey(sess.Key)},
			LastObservedAt: checkedAt,
		})
	}
	return issues
}

func latestCorrectiveFeedback(messages []providers.Message) string {
	for i := len(messages) - 1; i > 0; i-- {
		message := messages[i]
		if message.Role != "user" {
			continue
		}
		feedback := strings.TrimSpace(message.Content)
		if !looksLikeCorrectiveFeedback(feedback) {
			continue
		}
		for previous := i - 1; previous >= 0; previous-- {
			if strings.TrimSpace(messages[previous].Content) == "" {
				continue
			}
			if messages[previous].Role == "assistant" {
				return feedback
			}
			break
		}
	}
	return ""
}

func looksLikeCorrectiveFeedback(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	for _, phrase := range []string{
		"that is wrong", "this is wrong", "not correct", "not what i asked", "not what i want",
		"you didn't", "you did not", "try again", "redo", "fix this", "doesn't work", "does not work",
		"i'm unhappy", "i am unhappy", "bad answer", "missed the point",
	} {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

func buildAspirineSummary(checkedAt string, issues []aspirineIssue) aspirineSummary {
	summary := aspirineSummary{
		Status:    "healthy",
		CheckedAt: checkedAt,
		Issues:    issues,
	}
	for _, issue := range issues {
		if issue.Severity == "info" && issue.Status == "healthy" {
			continue
		}
		summary.IssueCount++
		switch issue.Severity {
		case "critical":
			summary.CriticalCount++
			summary.Status = "critical"
		case "warning":
			summary.WarningCount++
			if summary.Status != "critical" {
				summary.Status = "warning"
			}
		}
	}
	return summary
}

func enabledChannelNames(channels config.ChannelsConfig) []string {
	cards := enabledChannelCards(channels)
	names := make([]string, 0, len(cards))
	for _, card := range cards {
		names = append(names, card.ID)
	}
	return names
}

func recentChannelLogIssue(channel string, title string, checkedAt string) (aspirineIssue, bool) {
	lines := recentAspirineLogLines(320)
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		lower := strings.ToLower(line)
		if !strings.Contains(lower, channel) {
			continue
		}
		if !strings.Contains(lower, "error") &&
			!strings.Contains(lower, "failed") &&
			!strings.Contains(lower, "timeout") &&
			!strings.Contains(lower, "unauthorized") &&
			!strings.Contains(lower, "forbidden") &&
			!strings.Contains(lower, "not responding") &&
			!strings.Contains(lower, "panic") {
			continue
		}
		if len(line) > 360 {
			line = line[:360] + "..."
		}
		return aspirineIssue{
			ID:             channel + "-recent-error",
			Title:          title,
			Severity:       "warning",
			Status:         "needs_attention",
			Description:    line,
			Suggestion:     "Check the channel token, allowed users or groups, network access, and recent gateway logs. Restart the gateway after changing channel settings.",
			AutoFixAction:  "restart_gateway",
			AutoFixLabel:   "Restart gateway",
			Affected:       []string{channel},
			LastObservedAt: checkedAt,
		}, true
	}
	return aspirineIssue{}, false
}

func recentAspirineLogLines(limit int) []string {
	lines := recentJameClawFileLogs(limit)
	if gateway.logs.RunID() == 0 {
		return lines
	}
	bufferLines, _, _ := gateway.logs.LinesSince(0)
	lines = append(lines, bufferLines...)
	if len(lines) > limit {
		return lines[len(lines)-limit:]
	}
	return lines
}

func (h *Handler) handleAspirineAction(w http.ResponseWriter, r *http.Request) {
	action := r.PathValue("action")
	var pid int
	var err error

	switch action {
	case "start_gateway":
		pid, err = h.StartGateway()
	case "restart_gateway":
		pid, err = h.RestartGateway()
	default:
		http.Error(w, "unknown aspirine action", http.StatusNotFound)
		return
	}

	if err != nil {
		var precondErr *preconditionFailedError
		if errors.As(err, &precondErr) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"status":  "precondition_failed",
				"message": precondErr.reason,
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "failed",
			"message": err.Error(),
		})
		return
	}

	writeJSON(w, map[string]any{
		"status": "ok",
		"action": action,
		"pid":    pid,
	})
}

func (h *Handler) StartGateway() (int, error) {
	cfg, cfgErr := config.LoadConfig(h.configPath)
	if cfgErr == nil && cfg != nil {
		healthResp, statusCode, err := h.getGatewayHealth(cfg, 2*time.Second)
		if err == nil && statusCode == http.StatusOK {
			gateway.mu.Lock()
			defer gateway.mu.Unlock()
			ready, reason, err := h.gatewayStartReady()
			if err != nil {
				return 0, fmt.Errorf("failed to validate gateway start conditions: %w", err)
			}
			if !ready {
				return 0, &preconditionFailedError{reason: reason}
			}
			return h.startGatewayLocked("starting", healthResp.Pid)
		}
	}

	gateway.mu.Lock()
	defer gateway.mu.Unlock()
	// The caller may have reached this point while another request was starting
	// the gateway. Reuse that process instead of launching a second one that
	// would contend for the shared gateway port.
	if gateway.cmd != nil && isCmdProcessAliveLocked(gateway.cmd) {
		return gateway.cmd.Process.Pid, nil
	}
	if gateway.cmd != nil {
		gateway.cmd = nil
		setGatewayRuntimeStatusLocked("stopped")
	}
	ready, reason, err := h.gatewayStartReady()
	if err != nil {
		return 0, fmt.Errorf("failed to validate gateway start conditions: %w", err)
	}
	if !ready {
		return 0, &preconditionFailedError{reason: reason}
	}
	return h.startGatewayLocked("starting", 0)
}

func writeJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(body)
}
