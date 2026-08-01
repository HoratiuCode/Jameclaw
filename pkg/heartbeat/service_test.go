package heartbeat

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/jameclaw/pkg/tools"
)

func TestExecuteHeartbeat_Async(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "heartbeat-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	hs := NewHeartbeatService(tmpDir, 30, true)
	hs.stopChan = make(chan struct{}) // Enable for testing

	asyncCalled := false
	asyncResult := &tools.ToolResult{
		ForLLM:  "Background task started",
		ForUser: "Task started in background",
		Silent:  false,
		IsError: false,
		Async:   true,
	}

	hs.SetHandler(func(prompt, channel, chatID string) *tools.ToolResult {
		asyncCalled = true
		if prompt == "" {
			t.Error("Expected non-empty prompt")
		}
		return asyncResult
	})

	// Create HEARTBEAT.md
	os.WriteFile(filepath.Join(tmpDir, "HEARTBEAT.md"), []byte("Test task"), 0o644)

	// Execute heartbeat directly (internal method for testing)
	hs.executeHeartbeat()

	if !asyncCalled {
		t.Error("Expected handler to be called")
	}
}

func TestExecuteHeartbeat_ResultLogging(t *testing.T) {
	tests := []struct {
		name    string
		result  *tools.ToolResult
		wantLog string
	}{
		{
			name: "error result",
			result: &tools.ToolResult{
				ForLLM:  "Heartbeat failed: connection error",
				ForUser: "",
				Silent:  false,
				IsError: true,
				Async:   false,
			},
			wantLog: "error message",
		},
		{
			name: "silent result",
			result: &tools.ToolResult{
				ForLLM:  "Heartbeat completed successfully",
				ForUser: "",
				Silent:  true,
				IsError: false,
				Async:   false,
			},
			wantLog: "completion message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "heartbeat-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			hs := NewHeartbeatService(tmpDir, 30, true)
			hs.stopChan = make(chan struct{}) // Enable for testing

			hs.SetHandler(func(prompt, channel, chatID string) *tools.ToolResult {
				return tt.result
			})

			os.WriteFile(filepath.Join(tmpDir, "HEARTBEAT.md"), []byte("Test task"), 0o644)
			hs.executeHeartbeat()

			logFile := filepath.Join(tmpDir, "heartbeat.log")
			data, err := os.ReadFile(logFile)
			if err != nil {
				t.Fatalf("Failed to read log file: %v", err)
			}
			if string(data) == "" {
				t.Errorf("Expected log file to contain %s", tt.wantLog)
			}
		})
	}
}

func TestHeartbeatService_StartStop(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "heartbeat-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	hs := NewHeartbeatService(tmpDir, 1, true)

	err = hs.Start()
	if err != nil {
		t.Fatalf("Failed to start heartbeat service: %v", err)
	}

	hs.Stop()

	time.Sleep(100 * time.Millisecond)
}

func TestHeartbeatService_Disabled(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "heartbeat-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	hs := NewHeartbeatService(tmpDir, 1, false)

	if hs.enabled != false {
		t.Error("Expected service to be disabled")
	}

	err = hs.Start()
	_ = err // Disabled service returns nil
}

func TestExecuteHeartbeat_NilResult(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "heartbeat-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	hs := NewHeartbeatService(tmpDir, 30, true)
	hs.stopChan = make(chan struct{}) // Enable for testing

	hs.SetHandler(func(prompt, channel, chatID string) *tools.ToolResult {
		return nil
	})

	// Create HEARTBEAT.md
	os.WriteFile(filepath.Join(tmpDir, "HEARTBEAT.md"), []byte("Test task"), 0o644)

	// Should not panic with nil result
	hs.executeHeartbeat()
}

// TestLogPath verifies heartbeat log is written to workspace directory
func TestLogPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "heartbeat-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	hs := NewHeartbeatService(tmpDir, 30, true)

	// Write a log entry
	hs.logf("INFO", "Test log entry")

	// Verify log file exists at workspace root
	expectedLogPath := filepath.Join(tmpDir, "heartbeat.log")
	if _, err := os.Stat(expectedLogPath); os.IsNotExist(err) {
		t.Errorf("Expected log file at %s, but it doesn't exist", expectedLogPath)
	}
}

// TestHeartbeatFilePath verifies HEARTBEAT.md is at workspace root
func TestHeartbeatFilePath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "heartbeat-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	hs := NewHeartbeatService(tmpDir, 30, true)

	// Trigger default template creation
	hs.buildPrompt()

	// Verify HEARTBEAT.md exists at workspace root
	expectedPath := filepath.Join(tmpDir, "HEARTBEAT.md")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected HEARTBEAT.md at %s, but it doesn't exist", expectedPath)
	}
}

func TestBuildPrompt_TaskOnlyModeStaysIdleWithoutTasks(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "heartbeat-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	hs := NewHeartbeatServiceWithInitiative(tmpDir, 30, true, false)
	hs.createDefaultHeartbeatTemplate()

	if prompt := hs.buildPrompt(); prompt != "" {
		t.Fatalf("buildPrompt() = %q, want empty prompt for untouched default template", prompt)
	}
}

func TestBuildPrompt_InitiativeModeDiscoversProblemsWithoutUserTasks(t *testing.T) {
	tmpDir := t.TempDir()
	hs := NewHeartbeatServiceWithInitiative(tmpDir, 30, true, true)
	hs.createDefaultHeartbeatTemplate()

	prompt := hs.buildPrompt()
	if prompt == "" {
		t.Fatal("buildPrompt() = empty, want proactive initiative prompt")
	}
	for _, expected := range []string{
		"Take initiative",
		"Choose at most ONE",
		"NEEDS_APPROVAL",
		"HEARTBEAT_OK",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("prompt missing %q: %s", expected, prompt)
		}
	}
}

func TestSaveInitiativeRecord_PersistsLatestAndBoundedHistory(t *testing.T) {
	tmpDir := t.TempDir()
	first := InitiativeRecord{CheckedAt: time.Now().Add(-time.Minute), Status: "completed", Summary: "Fixed a failing check."}
	second := InitiativeRecord{CheckedAt: time.Now(), Status: "needs_approval", Summary: "Deployment needs approval."}
	if err := SaveInitiativeRecord(tmpDir, first); err != nil {
		t.Fatal(err)
	}
	if err := SaveInitiativeRecord(tmpDir, second); err != nil {
		t.Fatal(err)
	}

	latest, err := LoadInitiativeState(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if latest.Status != "needs_approval" || latest.Summary != second.Summary {
		t.Fatalf("latest = %#v", latest)
	}
	history, err := LoadInitiativeHistory(tmpDir, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 || history[0].Summary != second.Summary || history[1].Summary != first.Summary {
		t.Fatalf("history = %#v", history)
	}

	idle := InitiativeRecord{CheckedAt: time.Now().Add(time.Minute), Status: "idle", Summary: "Heartbeat OK"}
	if err := SaveInitiativeRecord(tmpDir, idle); err != nil {
		t.Fatal(err)
	}
	history, err = LoadInitiativeHistory(tmpDir, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 {
		t.Fatalf("idle check should not expand history, got %d entries", len(history))
	}
}

func TestBuildPrompt_UserTasksAfterMarkerProducePrompt(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "heartbeat-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	hs := NewHeartbeatService(tmpDir, 30, true)
	hs.createDefaultHeartbeatTemplate()

	path := filepath.Join(tmpDir, "HEARTBEAT.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read HEARTBEAT.md: %v", err)
	}
	updated := string(data) + "\n- Check unread Feishu messages\n"
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		t.Fatalf("Failed to update HEARTBEAT.md: %v", err)
	}

	prompt := hs.buildPrompt()
	if prompt == "" {
		t.Fatal("buildPrompt() = empty, want non-empty prompt when user tasks are present")
	}
	if !strings.Contains(prompt, "Check unread Feishu messages") {
		t.Fatalf("prompt = %q, want user task content", prompt)
	}
}
