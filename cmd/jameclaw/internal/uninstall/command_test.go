package uninstall

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/jameclaw/pkg/config"
)

func TestRunRemovesLocalState(t *testing.T) {
	originalInput := uninstallInput
	originalOutput := uninstallOutput
	t.Cleanup(func() {
		uninstallInput = originalInput
		uninstallOutput = originalOutput
	})

	tmpDir := t.TempDir()
	userHome := filepath.Join(tmpDir, "user")
	jameclawHome := filepath.Join(tmpDir, ".jameclaw")
	sshKeyPath := filepath.Join(userHome, ".ssh", "jameclaw_ed25519.key")

	t.Setenv("HOME", userHome)
	t.Setenv(config.EnvHome, jameclawHome)

	writeTestFile(t, filepath.Join(jameclawHome, "config.json"))
	writeTestFile(t, filepath.Join(jameclawHome, "launcher-config.json"))
	writeTestFile(t, filepath.Join(jameclawHome, "workspace", "AGENT.md"))
	writeTestFile(t, sshKeyPath)
	writeTestFile(t, sshKeyPath+".pub")

	var output bytes.Buffer
	uninstallOutput = &output

	if err := run(options{yes: true}); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	assertMissing(t, jameclawHome)
	assertMissing(t, sshKeyPath)
	assertMissing(t, sshKeyPath+".pub")

	if !strings.Contains(output.String(), "jameclaw install") {
		t.Fatalf("run() output = %q, want install guidance", output.String())
	}
}

func TestRunCanceledKeepsState(t *testing.T) {
	originalInput := uninstallInput
	originalOutput := uninstallOutput
	t.Cleanup(func() {
		uninstallInput = originalInput
		uninstallOutput = originalOutput
	})

	tmpDir := t.TempDir()
	userHome := filepath.Join(tmpDir, "user")
	jameclawHome := filepath.Join(tmpDir, ".jameclaw")
	configPath := filepath.Join(jameclawHome, "config.json")

	t.Setenv("HOME", userHome)
	t.Setenv(config.EnvHome, jameclawHome)

	writeTestFile(t, configPath)

	uninstallInput = strings.NewReader("n\n")
	var output bytes.Buffer
	uninstallOutput = &output

	if err := run(options{}); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("expected config to remain after cancel, stat error = %v", err)
	}
	if !strings.Contains(output.String(), "uninstall canceled") {
		t.Fatalf("run() output = %q, want cancel message", output.String())
	}
}

func TestRunRemovesExternalConfigPath(t *testing.T) {
	originalInput := uninstallInput
	originalOutput := uninstallOutput
	t.Cleanup(func() {
		uninstallInput = originalInput
		uninstallOutput = originalOutput
	})

	tmpDir := t.TempDir()
	userHome := filepath.Join(tmpDir, "user")
	jameclawHome := filepath.Join(tmpDir, ".jameclaw")
	externalConfigDir := filepath.Join(tmpDir, "custom-config")
	configPath := filepath.Join(externalConfigDir, "config.json")
	launcherConfigPath := filepath.Join(externalConfigDir, "launcher-config.json")

	t.Setenv("HOME", userHome)
	t.Setenv(config.EnvHome, jameclawHome)
	t.Setenv(config.EnvConfig, configPath)

	writeTestFile(t, configPath)
	writeTestFile(t, launcherConfigPath)
	writeTestFile(t, filepath.Join(jameclawHome, "workspace", "AGENT.md"))

	uninstallOutput = &bytes.Buffer{}

	if err := run(options{yes: true}); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	assertMissing(t, jameclawHome)
	assertMissing(t, configPath)
	assertMissing(t, launcherConfigPath)
}

func writeTestFile(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte("test"), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func assertMissing(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %q to be removed, stat err = %v", path, err)
	}
}
