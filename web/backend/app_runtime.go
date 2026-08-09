package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/pkg/logger"
	"github.com/sipeed/jameclaw/web/backend/utils"
)

const (
	browserDelay    = 500 * time.Millisecond
	shutdownTimeout = 15 * time.Second
)

// shutdownApp gracefully shuts down all server components and resources.
// It performs the following shutdown sequence:
//   - Shuts down the API handler to close all active SSE (Server-Sent Events) connections
//   - Disables HTTP keep-alive to prevent new connections during shutdown
//   - Attempts graceful HTTP server shutdown with timeout
//   - Logs shutdown status at appropriate log levels
//
// The function handles timeout errors gracefully by logging them at info level
// since context.DeadlineExceeded is expected when there are active long-running
// connections (such as SSE streams).
//
// This function should be called during application termination to ensure
// clean resource cleanup and proper connection closure.
func shutdownApp() {
	// First, shutdown API handler to close all SSE connections
	if apiHandler != nil {
		apiHandler.Shutdown()
	}

	if server != nil {
		// Disable keep-alive to allow graceful shutdown
		server.SetKeepAlivesEnabled(false)

		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			// Context deadline exceeded is expected if there are active connections
			// This is not necessarily an error, so log it at info level
			if errors.Is(err, context.DeadlineExceeded) {
				logger.Infof("Server shutdown timeout after %v, forcing close", shutdownTimeout)
			} else {
				logger.Errorf("Server shutdown error: %v", err)
			}
		} else {
			logger.Infof("Server shutdown completed successfully")
		}
	}
}

func openBrowser() error {
	if serverAddr == "" {
		return fmt.Errorf("server address not set")
	}
	return utils.OpenBrowser(launcherOpenURL(serverAddr))
}

func openBrowserBackground() error {
	if serverAddr == "" {
		return fmt.Errorf("server address not set")
	}
	return utils.OpenBrowserBackground(launcherOpenURL(serverAddr))
}

func openNewChat() error {
	if serverAddr == "" {
		return fmt.Errorf("server address not set")
	}

	chatURL, err := url.Parse(launcherOpenURL(serverAddr))
	if err != nil {
		return err
	}
	query := chatURL.Query()
	query.Set("new_chat", "1")
	chatURL.RawQuery = query.Encode()
	return utils.OpenBrowser(chatURL.String())
}

func openTerminalChat() error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("terminal chat launcher is only supported on macOS")
	}

	binary := utils.FindJameclawBinary()
	command := shellQuote(binary)
	if configFile != "" {
		command = config.EnvConfig + "=" + shellQuote(configFile) + " " + command
	}
	command += " agent"

	script := fmt.Sprintf(
		`tell application "Terminal"
	activate
	do script %s
end tell`,
		appleScriptString(command),
	)
	return exec.Command("osascript", "-e", script).Start()
}

func openNativeSettings() error {
	// Settings now lives inside the one visible JameClaw Desktop application.
	// Reopen that app instead of launching a second native settings bundle.
	return openNativeHome()
}

func openNativeHome() error {
	if runtime.GOOS != "darwin" {
		return openBrowserBackground()
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	executableDir := filepath.Dir(executable)
	for _, candidate := range []string{
		filepath.Clean(filepath.Join(executableDir, "..", "..")),
		filepath.Join(executableDir, "Jame.app"),
		filepath.Clean(filepath.Join(executableDir, "..", "Resources", "Jame.app")),
	} {
		if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
			return exec.Command("open", candidate).Start()
		}
	}
	return fmt.Errorf("JameClaw Desktop.app was not found around the launcher")
}

// closeNativeHome keeps the menu-bar launcher and the native Jame window in
// sync. It is intentionally best-effort: shutdown must never be blocked by a
// window that is already closed or was launched from another location.
func closeNativeHome() {
	if runtime.GOOS != "darwin" {
		return
	}
	_ = exec.Command("osascript", "-e", `tell application id "com.jameclaw.launcher" to quit`).Run()
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func appleScriptString(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}
