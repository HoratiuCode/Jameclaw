// JameClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 JameClaw contributors

package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const pidFileName = "gateway.pid"

type gatewayStatus struct {
	running bool
	pid     int
}

func getPidPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".jameclaw", pidFileName)
}

func isProcessRunning(pid int) bool {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid))
		output, err := cmd.Output()
		if err != nil {
			return false
		}
		return strings.Contains(string(output), strconv.Itoa(pid))
	} else if runtime.GOOS == "darwin" {
		cmd := exec.Command("ps", "aux")
		output, err := cmd.Output()
		if err != nil {
			return false
		}
		return strings.Contains(string(output), fmt.Sprintf(" %d ", pid))
	}
	// Linux
	_, err := os.Stat(fmt.Sprintf("/proc/%d", pid))
	return err == nil
}

func getGatewayStatus() gatewayStatus {
	pidPath := getPidPath()
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return gatewayStatus{running: false}
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return gatewayStatus{running: false}
	}
	if !isProcessRunning(pid) {
		os.Remove(pidPath)
		return gatewayStatus{running: false}
	}
	return gatewayStatus{
		running: true,
		pid:     pid,
	}
}

func startGateway() error {
	status := getGatewayStatus()
	if status.running {
		return fmt.Errorf("gateway is already running (PID: %d)", status.pid)
	}

	pidPath := getPidPath()
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", "start /B jameclaw gateway > NUL 2>&1")
	} else {
		cmd = exec.Command("sh", "-c", "nohup jameclaw gateway > /dev/null 2>&1 & echo $! > "+pidPath)
	}

	err := cmd.Start()
	if err != nil {
		return err
	}

	time.Sleep(1 * time.Second)

	if runtime.GOOS == "windows" {
		cmd := exec.Command(
			"wmic",
			"process",
			"where",
			"name='jameclaw.exe' and commandline like '%gateway%'",
			"get",
			"processid",
		)
		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to get gateway PID: %w", err)
		}
		lines := strings.Split(string(output), "\n")
		for _, line := range lines[1:] {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			pid, err := strconv.Atoi(line)
			if err == nil {
				os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), 0o600)
				break
			}
		}
	}

	status = getGatewayStatus()
	if !status.running {
		return fmt.Errorf("failed to start gateway")
	}
	return nil
}

func stopGateway() error {
	status := getGatewayStatus()
	if !status.running {
		return fmt.Errorf("gateway is not running")
	}

	var err error
	if runtime.GOOS == "windows" {
		err = exec.Command("taskkill", "/F", "/PID", strconv.Itoa(status.pid)).Run()
	} else {
		err = exec.Command("kill", "-9", strconv.Itoa(status.pid)).Run()
	}
	if err != nil {
		return err
	}

	// Retry a few times to confirm the process has stopped.
	for i := 0; i < 5; i++ {
		if !isProcessRunning(status.pid) {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	os.Remove(getPidPath())
	return nil
}

func (a *App) newGatewayPage() tview.Primitive {
	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.SetBorder(true).
		SetTitle(" [" + uiTagRed + "::b] GATEWAY MANAGEMENT ").
		SetTitleColor(uiColorAccentRed).
		SetBorderColor(uiColorBorder)
	flex.SetBackgroundColor(uiColorBackground)

	statusTV := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft).
		SetWrap(true).
		SetText("Checking status...")
	statusTV.SetBackgroundColor(uiColorBackground)

	var updateStatus func()

	// Use a List as the button surface to keep rendering and input reliable.
	buttons := tview.NewList()
	buttons.SetBackgroundColor(uiColorBackground)
	buttons.SetMainTextColor(uiColorText)
	buttons.SetSelectedBackgroundColor(uiColorAccentGreen)
	buttons.SetSelectedTextColor(uiColorInverseText)

	buttons.AddItem("START GATEWAY", "Launch jameclaw gateway in the background", 's', func() {
		if !getGatewayStatus().running {
			err := startGateway()
			if err != nil {
				a.showError(err.Error())
			}
			updateStatus()
		}
	})
	buttons.AddItem("STOP GATEWAY", "Stop the managed gateway process", 'x', func() {
		if getGatewayStatus().running {
			err := stopGateway()
			if err != nil {
				a.showError(err.Error())
			}
			updateStatus()
		}
	})
	buttons.AddItem("OPEN WEB CONSOLE", "Run jameclaw dashboard", 'd', func() {
		a.tapp.Suspend(func() {
			cmd := exec.Command("jameclaw", "dashboard")
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			_ = cmd.Run()
		})
		updateStatus()
	})
	buttons.AddItem("REFRESH STATUS", "Recheck gateway, Web Console, config and health", 'r', updateStatus)
	var stopPolling func()
	buttons.AddItem("BACK", "Return to control room", 'b', func() {
		stopPolling()
		a.goBack()
	})

	buttonFlex := tview.NewFlex().SetDirection(tview.FlexColumn)
	buttonFlex.
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(buttons, 34, 1, true).
		AddItem(tview.NewBox(), 0, 1, false)

	flex.
		AddItem(statusTV, 0, 3, false).
		AddItem(buttonFlex, 8, 1, true).
		AddItem(tview.NewBox(), 0, 1, false)

	updateStatus = func() {
		status := getGatewayStatus()
		webURL := launcherDashboardURL()
		webStatus := "[" + uiTagDanger + "::b]stopped[-]"
		if webConsoleReachable(webURL) {
			webStatus = "[" + uiTagGreenBold + "::b]running[-]"
		}
		model, channels, enabled, configErr := gatewayManagementConfigSummary()
		healthBadge := "[" + uiTagGreenBold + "::b]ready[-]"
		if configErr != "" {
			healthBadge = "[" + uiTagDanger + "::b]config warning[-]"
		}

		var text string
		if status.running {
			text = fmt.Sprintf(
				"[%s::b]GATEWAY RUNNING[-]\n\n[%s]Health:[-] %s\n[%s]Gateway PID:[-] %d\n[%s]Web Console:[-] %s  %s\n[%s]Model:[-] %s\n[%s]Channels:[-] %d configured / %d enabled\n[%s]Actions:[-] start, stop, open dashboard, refresh\n",
				uiTagGreenBold,
				uiTagMuted,
				healthBadge,
				uiTagMuted,
				status.pid,
				uiTagMuted,
				webStatus,
				webURL,
				uiTagMuted,
				model,
				uiTagMuted,
				channels,
				enabled,
				uiTagMuted,
			)
			buttons.SetItemText(0, "["+uiTagMutedLabel+"]START GATEWAY[-]", "Already running")
			buttons.SetItemText(1, "["+uiTagDanger+"]STOP GATEWAY[-]", "Stop PID "+strconv.Itoa(status.pid))
		} else {
			text = fmt.Sprintf(
				"[%s::b]GATEWAY STOPPED[-]\n\n[%s]Health:[-] %s\n[%s]Gateway PID:[-] N/A\n[%s]Web Console:[-] %s  %s\n[%s]Model:[-] %s\n[%s]Channels:[-] %d configured / %d enabled\n[%s]Actions:[-] start gateway, open dashboard, refresh\n",
				uiTagDanger,
				uiTagMuted,
				healthBadge,
				uiTagMuted,
				uiTagMuted,
				webStatus,
				webURL,
				uiTagMuted,
				model,
				uiTagMuted,
				channels,
				enabled,
				uiTagMuted,
			)
			buttons.SetItemText(0, "["+uiTagGreenBold+"]START GATEWAY[-]", "Launch jameclaw gateway in the background")
			buttons.SetItemText(1, "["+uiTagMutedLabel+"]STOP GATEWAY[-]", "Gateway is not running")
		}
		if configErr != "" {
			text += "\n[" + uiTagDanger + "]Config warning:[-] " + configErr
		}
		statusTV.SetText(text)
	}

	updateStatus()

	done := make(chan struct{})
	var closeDone sync.Once
	stopPolling = func() {
		closeDone.Do(func() {
			close(done)
		})
	}
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				a.tapp.QueueUpdateDraw(updateStatus)
			case <-done:
				return
			}
		}
	}()

	originalInputCapture := flex.GetInputCapture()
	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			stopPolling()
			return a.goBack()
		}
		if originalInputCapture != nil {
			return originalInputCapture(event)
		}
		return event
	})

	a.pageRefreshFns["gateway"] = updateStatus

	return a.buildShell("gateway", flex, " ["+uiTagGreenBold+"]Enter:[-] select  ["+uiTagRed+"]s:[-] start  ["+uiTagRed+"]x:[-] stop  ["+uiTagRed+"]d:[-] dashboard  ["+uiTagRed+"]r:[-] refresh  ["+uiTagMuted+"]ESC:[-] back ")
}

func gatewayManagementConfigSummary() (model string, channels int, enabled int, errText string) {
	model = "not configured"
	data, err := os.ReadFile(mainConfigPath())
	if err != nil {
		return model, 0, 0, err.Error()
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		return model, 0, 0, err.Error()
	}
	if agents, ok := cfg["agents"].(map[string]any); ok {
		if defaults, ok := agents["defaults"].(map[string]any); ok {
			if value, ok := defaults["model_name"].(string); ok && strings.TrimSpace(value) != "" {
				model = strings.TrimSpace(value)
			}
		}
	}
	if channelMap, ok := cfg["channels"].(map[string]any); ok {
		for _, raw := range channelMap {
			channels++
			entry, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			if value, ok := entry["enabled"].(bool); ok && value {
				enabled++
			}
		}
	}
	return model, channels, enabled, ""
}
