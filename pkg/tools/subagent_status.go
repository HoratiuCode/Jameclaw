package tools

const (
	SubagentStatusQueued    = "queued"
	SubagentStatusRunning   = "running"
	SubagentStatusSucceeded = "succeeded"
	SubagentStatusFailed    = "failed"
	SubagentStatusTimedOut  = "timed_out"
	SubagentStatusCancelled = "cancelled"
	SubagentStatusLost      = "lost"
)

func normalizeSubagentStatus(status string) string {
	switch status {
	case "completed":
		return SubagentStatusSucceeded
	case "canceled":
		return SubagentStatusCancelled
	case "":
		return SubagentStatusQueued
	default:
		return status
	}
}

func isTerminalSubagentStatus(status string) bool {
	switch normalizeSubagentStatus(status) {
	case SubagentStatusSucceeded, SubagentStatusFailed, SubagentStatusTimedOut, SubagentStatusCancelled, SubagentStatusLost:
		return true
	default:
		return false
	}
}
