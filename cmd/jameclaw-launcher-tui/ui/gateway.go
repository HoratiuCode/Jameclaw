// JameClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 JameClaw contributors

package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
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
		SetTextAlign(tview.AlignCenter).
		SetText("Checking status...")
	statusTV.SetBackgroundColor(uiColorBackground)

	var updateStatus func()

	// Use a List as the button surface to keep rendering and input reliable.
	buttons := tview.NewList()
	buttons.SetBackgroundColor(uiColorBackground)
	buttons.SetMainTextColor(uiColorText)
	buttons.SetSelectedBackgroundColor(uiColorAccentGreen)
	buttons.SetSelectedTextColor(uiColorInverseText)

	buttons.AddItem(" ["+uiTagGreenBold+"]START["+uiTagText+"]   ", "", 0, func() {
		if !getGatewayStatus().running {
			err := startGateway()
			if err != nil {
				a.showError(err.Error())
			}
			updateStatus()
		}
	})
	buttons.AddItem(" ["+uiTagDanger+"]STOP["+uiTagText+"]    ", "", 0, func() {
		if getGatewayStatus().running {
			err := stopGateway()
			if err != nil {
				a.showError(err.Error())
			}
			updateStatus()
		}
	})

	buttonFlex := tview.NewFlex().SetDirection(tview.FlexColumn)
	buttonFlex.
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(buttons, 20, 1, true).
		AddItem(tview.NewBox(), 0, 1, false)

	flex.
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(statusTV, 3, 1, false).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(buttonFlex, 4, 1, true).
		AddItem(tview.NewBox(), 0, 1, false)

	updateStatus = func() {
		status := getGatewayStatus()
		if status.running {
			statusTV.SetText(fmt.Sprintf("[%s::b]GATEWAY RUNNING[-]\n\nPID: %d", uiTagGreenBold, status.pid))
			buttons.SetItemText(0, " ["+uiTagMutedLabel+"]START["+uiTagText+"]   ", "")
			buttons.SetItemText(1, " ["+uiTagDanger+"]STOP["+uiTagText+"]    ", "")
		} else {
			statusTV.SetText("[" + uiTagDanger + "::b]GATEWAY STOPPED[-]\n\nPID: N/A")
			buttons.SetItemText(0, " ["+uiTagGreenBold+"]START["+uiTagText+"]   ", "")
			buttons.SetItemText(1, " ["+uiTagMutedLabel+"]STOP["+uiTagText+"]    ", "")
		}
	}

	updateStatus()

	done := make(chan struct{})
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
			close(done)
			return a.goBack()
		}
		if originalInputCapture != nil {
			return originalInputCapture(event)
		}
		return event
	})

	a.pageRefreshFns["gateway"] = updateStatus

	return a.buildShell("gateway", flex, " ["+uiTagGreenBold+"]Enter:[-] select  ["+uiTagMuted+"]ESC:[-] back ")
}
