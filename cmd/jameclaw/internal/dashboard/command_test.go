package dashboard

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/web/backend/launcherconfig"
)

func setupDashboardTest(t *testing.T, port int) *bytes.Buffer {
	t.Helper()

	home := t.TempDir()
	configPath := filepath.Join(home, "config.json")
	t.Setenv(config.EnvHome, home)
	t.Setenv(config.EnvConfig, configPath)
	if err := os.WriteFile(configPath, []byte(`{"version":1}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if port > 0 {
		if err := launcherconfig.Save(
			launcherconfig.PathForAppConfig(configPath),
			launcherconfig.Config{Port: port},
		); err != nil {
			t.Fatalf("save launcher config: %v", err)
		}
	}

	var out bytes.Buffer
	oldOut := dashboardOutput
	oldOpen := dashboardOpenURL
	oldCopy := dashboardCopyText
	oldStart := dashboardStartWebUI
	oldReachable := dashboardReachable
	oldWaitFor := dashboardWaitFor
	oldWait := dashboardWait
	t.Cleanup(func() {
		dashboardOutput = oldOut
		dashboardOpenURL = oldOpen
		dashboardCopyText = oldCopy
		dashboardStartWebUI = oldStart
		dashboardReachable = oldReachable
		dashboardWaitFor = oldWaitFor
		dashboardWait = oldWait
	})
	dashboardOutput = &out
	dashboardOpenURL = func(string) error { return nil }
	dashboardCopyText = func(string) error { return nil }
	dashboardStartWebUI = func() error { return nil }
	dashboardReachable = func(context.Context, string) bool { return true }
	dashboardWaitFor = func(context.Context, string, time.Duration) bool { return true }
	dashboardWait = 10 * time.Millisecond
	return &out
}

func TestResolveTargetUsesLauncherConfigAndAccessToken(t *testing.T) {
	out := setupDashboardTest(t, 19001)
	_ = out
	if err := os.WriteFile(filepath.Join(os.Getenv(config.EnvHome), "launcher_access_token"), []byte("token-123\n"), 0o600); err != nil {
		t.Fatalf("write token: %v", err)
	}

	target, err := resolveTarget()
	if err != nil {
		t.Fatalf("resolveTarget() error = %v", err)
	}
	if target.Port != 19001 {
		t.Fatalf("port = %d, want 19001", target.Port)
	}
	if target.BaseURL != "http://localhost:19001" {
		t.Fatalf("base URL = %q", target.BaseURL)
	}
	if !strings.Contains(target.AuthenticatedURL, "access_token=token-123") {
		t.Fatalf("authenticated URL = %q, want access token", target.AuthenticatedURL)
	}
}

func TestRunDashboardNoOpenCopiesAndDoesNotOpen(t *testing.T) {
	port := 19002
	out := setupDashboardTest(t, port)

	var opened bool
	var copied string
	dashboardOpenURL = func(string) error {
		opened = true
		return nil
	}
	dashboardCopyText = func(text string) error {
		copied = text
		return nil
	}

	if err := runDashboard(context.Background(), dashboardOptions{NoOpen: true}); err != nil {
		t.Fatalf("runDashboard() error = %v", err)
	}
	if opened {
		t.Fatal("browser opener was called with --no-open")
	}
	if copied != "http://localhost:"+strconv.Itoa(port) {
		t.Fatalf("copied = %q", copied)
	}
	if !strings.Contains(out.String(), "SSH tunnel hint") {
		t.Fatalf("output missing SSH hint:\n%s", out.String())
	}
}

func TestRunDashboardStartsWebUIWhenUnreachable(t *testing.T) {
	out := setupDashboardTest(t, 9)
	var started bool
	dashboardStartWebUI = func() error {
		started = true
		return nil
	}
	dashboardReachable = func(context.Context, string) bool { return false }
	dashboardWaitFor = func(context.Context, string, time.Duration) bool { return false }

	if err := runDashboard(context.Background(), dashboardOptions{NoOpen: true}); err != nil {
		t.Fatalf("runDashboard() error = %v", err)
	}
	if !started {
		t.Fatal("expected WebUI start attempt")
	}
	if !strings.Contains(out.String(), "Web Console is not running") {
		t.Fatalf("output missing start message:\n%s", out.String())
	}
}
