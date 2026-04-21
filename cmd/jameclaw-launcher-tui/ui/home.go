// JameClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 JameClaw contributors

package ui

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/rivo/tview"
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
	overview.SetText(fmt.Sprintf(
		"[%s::b]%s[-]\n\n[%s]Skin:[-] %s\n[%s]Model:[-] %s\n[%s]Structure:[-] web console, TUI launcher, gateway, chat, skins\n",
		uiTagGreenBold,
		currentAgentName,
		uiTagMuted,
		currentSkinName,
		uiTagMuted,
		a.cfg.CurrentModelLabel(),
		uiTagMuted,
	))

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
		" ["+uiTagRed+"]m:[-] model  ["+uiTagRed+"]n:[-] channels  ["+uiTagRed+"]s:[-] skins  ["+uiTagRed+"]g:[-] gateway  ["+uiTagRed+"]c:[-] chat  ["+uiTagDanger+"]q:[-] quit ",
	)
}
