// JameClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 JameClaw contributors

package ui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rivo/tview"

	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/web/backend/launcherconfig"
)

func (a *App) newHomePage() tview.Primitive {
	list := tview.NewList()
	list.SetBorder(true).
		SetTitle(" [" + uiTagRed + "::b] ACTIVE CONFIGURATION ").
		SetTitleColor(uiColorAccentRed).
		SetBorderColor(uiColorBorder)
	list.SetMainTextColor(uiColorText)
	list.SetSecondaryTextColor(uiColorMuted)
	list.SetSelectedStyle(uiSelectedStyle)
	list.SetHighlightFullLine(true)
	list.SetBackgroundColor(uiColorBackground)

	overview := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true)
	overview.SetBorder(true).
		SetTitle(" [" + uiTagRed + "::b] CONTROL ROOM ").
		SetTitleColor(uiColorAccentRed).
		SetBorderColor(uiColorBorder)
	overview.SetBackgroundColor(uiColorPanel)
	refreshOverview := func() {
		overview.SetText(formatHomeDashboard(a.cfg.CurrentModelLabel(), currentSkinName))
	}
	refreshOverview()

	rebuildList := func() {
		sel := list.GetCurrentItem()
		list.Clear()
		list.AddItem("MODEL: "+a.cfg.CurrentModelLabel(), "Select to configure AI model", 'm', func() {
			a.navigateTo("schemes", a.newSchemesPage())
		})
		list.AddItem(
			"CHANNELS: Configure communication channels",
			"Manage Telegram/Discord/WeChat channels",
			'n',
			func() {
				a.navigateTo("channels", a.newChannelsPage())
			},
		)
		list.AddItem("SKINS: Choose launcher theme", "Pick a preset or custom terminal skin", 's', func() {
			a.navigateTo("skins", a.newSkinsPage())
		})
		list.AddItem("GATEWAY MANAGEMENT", "Manage JameClaw gateway daemon", 'g', func() {
			a.navigateTo("gateway", a.newGatewayPage())
		})
		list.AddItem("DASHBOARD: Open Web Console", "Open/copy browser dashboard URL", 'd', func() {
			a.tapp.Suspend(func() {
				cmd := exec.Command("jameclaw", "dashboard")
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				_ = cmd.Run()
			})
			refreshOverview()
		})
		list.AddItem("REFRESH STATUS", "Update dashboard health badges", 'r', refreshOverview)
		list.AddItem("CHAT: Start AI agent chat", "Launch interactive chat session", 'c', func() {
			a.tapp.Suspend(func() {
				cmd := exec.Command("jameclaw", "agent")
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				_ = cmd.Run()
			})
		})
		list.AddItem("QUIT SYSTEM", "Exit JameClaw Launcher", 'q', func() { a.tapp.Stop() })
		if sel >= 0 && sel < list.GetItemCount() {
			list.SetCurrentItem(sel)
		}
	}
	rebuildList()

	a.pageRefreshFns["home"] = rebuildList

	return a.buildShell(
		"home",
		tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(overview, 0, 1, false).
			AddItem(list, 0, 2, true),
		" ["+uiTagRed+"]m:[-] model  ["+uiTagRed+"]n:[-] channels  ["+uiTagRed+"]s:[-] skins  ["+uiTagRed+"]g:[-] gateway  ["+uiTagRed+"]d:[-] dashboard  ["+uiTagRed+"]r:[-] refresh  ["+uiTagRed+"]c:[-] chat  ["+uiTagDanger+"]q:[-] quit ",
	)
}

func formatHomeDashboard(modelLabel, skinName string) string {
	webURL := launcherDashboardURL()
	gateway := getGatewayStatus()
	gatewayBadge := "[" + uiTagDanger + "::b]stopped[-]"
	if gateway.running {
		gatewayBadge = fmt.Sprintf("[%s::b]running[-] [gray]pid %d[-]", uiTagGreenBold, gateway.pid)
	}
	webBadge := "[" + uiTagDanger + "::b]stopped[-]"
	if webConsoleReachable(webURL) {
		webBadge = "[" + uiTagGreenBold + "::b]running[-]"
	}
	channelCount, enabledCount, recentError := mainConfigSummary()
	health := "[" + uiTagGreenBold + "::b]healthy[-]"
	if recentError != "" {
		health = "[" + uiTagDanger + "::b]needs attention[-]"
	}

	return fmt.Sprintf(
		"[%s::b]%s[-]\n\n[%s]Health:[-] %s\n[%s]Web Console:[-] %s  %s\n[%s]Gateway:[-] %s\n[%s]Model:[-] %s\n[%s]Channels:[-] %d configured / %d enabled\n[%s]Skin:[-] %s\n[%s]Shortcuts:[-] m model | n channels | s skins | g gateway | d dashboard | r refresh | c chat | q quit\n",
		uiTagGreenBold,
		currentAgentName,
		uiTagMuted,
		health,
		uiTagMuted,
		webBadge,
		webURL,
		uiTagMuted,
		gatewayBadge,
		uiTagMuted,
		modelLabel,
		uiTagMuted,
		channelCount,
		enabledCount,
		uiTagMuted,
		skinName,
		uiTagMuted,
	)
}

func mainConfigPath() string {
	if path := strings.TrimSpace(os.Getenv(config.EnvConfig)); path != "" {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".jameclaw", "config.json")
}

func launcherDashboardURL() string {
	launcherCfg, err := launcherconfig.Load(
		launcherconfig.PathForAppConfig(mainConfigPath()),
		launcherconfig.Default(),
	)
	if err != nil || launcherCfg.Port <= 0 {
		return "http://localhost:18800"
	}
	return fmt.Sprintf("http://localhost:%d", launcherCfg.Port)
}

func webConsoleReachable(rawURL string) bool {
	client := http.Client{Timeout: 400 * time.Millisecond}
	resp, err := client.Get(rawURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < http.StatusInternalServerError
}

func mainConfigSummary() (channels int, enabled int, recentError string) {
	data, err := os.ReadFile(mainConfigPath())
	if err != nil {
		return 0, 0, err.Error()
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		return 0, 0, err.Error()
	}
	channelMap, ok := cfg["channels"].(map[string]any)
	if !ok {
		return 0, 0, ""
	}
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
	return channels, enabled, ""
}
