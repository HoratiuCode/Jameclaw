package agent

import (
	"fmt"
	"strings"

	"github.com/sipeed/jameclaw/pkg/tools"
	"github.com/sipeed/jameclaw/pkg/utils"
)

// MaxRecoveryAttempts is the total number of tool strategies allowed for one
// turn after the first strategy fails.
const MaxRecoveryAttempts = 2

func recoverableToolFailure(result *tools.ToolResult) bool {
	if result == nil || !result.IsError {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(result.ForLLM))
	if result.Err != nil {
		message += " " + strings.ToLower(result.Err.Error())
	}
	for _, marker := range []string{"permission denied", "access denied", "approval", "unauthorized", "forbidden", "authentication", "invalid token", "confirm=true", "not approved"} {
		if strings.Contains(message, marker) {
			return false
		}
	}
	for _, marker := range []string{"not found", "no such file", "no such directory", "timeout", "timed out", "connection refused", "connection reset", "network", "selector", "element", "invalid argument", "invalid json", "missing or invalid", "required", "parse", "exit status", "command failed", "failed to execute", "temporary", "could not connect"} {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}

func (ts *turnState) recordRecoveryFailure(result *tools.ToolResult) (attempt int, canRecover bool) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if !recoverableToolFailure(result) {
		ts.recoveryAttempts = MaxRecoveryAttempts
		return MaxRecoveryAttempts, false
	}
	ts.recoveryAttempts++
	return ts.recoveryAttempts, ts.recoveryAttempts < MaxRecoveryAttempts
}

func (ts *turnState) recoveryAttemptCount() int {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.recoveryAttempts
}

func (ts *turnState) resetRecoveryAfterSuccess() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.recoveryAttempts = 0
}

func recoveryInstruction(toolName string, result *tools.ToolResult, attempt int, canRecover bool) string {
	failure := "the tool returned an error"
	if result != nil && strings.TrimSpace(result.ForLLM) != "" {
		failure = utils.Truncate(strings.TrimSpace(result.ForLLM), 700)
	}
	if !canRecover {
		return fmt.Sprintf("[Recovery stopped after %d/%d attempts. Tool: %s. Error: %s] Do not call another tool for this objective. Explain what was attempted, why it failed, and what the user must provide or approve next.", attempt, MaxRecoveryAttempts, toolName, failure)
	}
	return fmt.Sprintf("[Recovery attempt %d/%d. Tool: %s failed: %s] Do not repeat the same tool and arguments. Choose a different safe strategy, verify the relevant state first, or explain why no safe alternative exists.", attempt, MaxRecoveryAttempts, toolName, failure)
}
