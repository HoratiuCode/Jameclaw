package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (a *App) newSkinsPage() tview.Primitive {
	list := tview.NewList()
	list.SetBorder(true).
		SetTitle(" [" + uiTagRed + "::b] LAUNCHER SKINS ").
		SetTitleColor(uiColorAccentRed).
		SetBorderColor(uiColorBorder)
	list.SetMainTextColor(uiColorText)
	list.SetSecondaryTextColor(uiColorMuted)
	list.SetSelectedStyle(uiSelectedStyle)
	list.SetHighlightFullLine(true)
	list.SetBackgroundColor(uiColorBackground)

	preview := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true)
	preview.SetBorder(true).
		SetTitle(" [" + uiTagRed + "::b] PREVIEW ").
		SetTitleColor(uiColorAccentRed).
		SetBorderColor(uiColorBorder)
	preview.SetBackgroundColor(uiColorPanel)

	skins := availableSkins()
	selectedSkin := a.cfg.Skin

	rebuild := func() {
		sel := list.GetCurrentItem()
		list.Clear()

		for _, skin := range skins {
			name := skin.Name
			desc := skin.Description
			if name == selectedSkin {
				desc = "Current skin: " + desc
			}
			skinName := name
			list.AddItem(strings.ToUpper(name), desc, 0, func() {
				a.cfg.Skin = skinName
				a.refreshSkin()
				selectedSkin = skinName
				a.save()
				preview.SetText(fmt.Sprintf(
					"[%s::b]%s[-]\n\n%s\n\n[%s]Launcher structure[-]\n- web console\n- TUI launcher\n- gateway\n- skills and config\n",
					uiTagGreenBold,
					skinName,
					skin.Description,
					uiTagMutedLabel,
				))
				a.goBack()
			})
		}

		if len(skins) > 0 {
			idx := 0
			for i, skin := range skins {
				if skin.Name == selectedSkin {
					idx = i
					break
				}
			}
			if sel >= 0 && sel < list.GetItemCount() {
				list.SetCurrentItem(sel)
			} else {
				list.SetCurrentItem(idx)
			}
		}
	}
	rebuild()

	a.pageRefreshFns["skins"] = rebuild

	if len(skins) > 0 {
		preview.SetText(fmt.Sprintf(
			"[%s::b]%s[-]\n\n%s\n\n[%s]This matches the Hermes-style idea of a configurable terminal skin.\nChoose a preset, save it, and the launcher updates immediately.[-]",
			uiTagGreenBold,
			skins[0].Name,
			skins[0].Description,
			uiTagMutedLabel,
		))
	}

	flex := tview.NewFlex().
		AddItem(list, 0, 1, true).
		AddItem(preview, 0, 2, false)

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			return a.goBack()
		}
		return event
	})

	return a.buildShell("skins", flex, " ["+uiTagGreenBold+"]Enter:[-] apply  ["+uiTagMuted+"]ESC:[-] back ")
}
