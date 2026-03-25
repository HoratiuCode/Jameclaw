// JameClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 JameClaw contributors

package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	tuicfg "github.com/sipeed/jameclaw/cmd/jameclaw-launcher-tui/config"
)

func (a *App) newSchemesPage() tview.Primitive {
	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false)
	table.SetBorder(true).
		SetTitle(" [" + uiTagRed + "::b] PROVIDER SCHEMES ").
		SetTitleColor(uiColorAccentRed).
		SetBorderColor(uiColorBorder)
	table.SetSelectedStyle(uiSelectedStyle)
	table.SetBackgroundColor(uiColorBackground)

	rowToIdx := func(row int) int { return row / 2 }

	selectedSchemeName := func() string {
		row, _ := table.GetSelection()
		idx := rowToIdx(row)
		schemes := a.cfg.Provider.Schemes
		if idx >= 0 && idx < len(schemes) {
			return schemes[idx].Name
		}
		return ""
	}

	rebuild := func() {
		selName := selectedSchemeName()
		table.Clear()
		schemes := a.cfg.Provider.Schemes
		for i, s := range schemes {
			nameRow := i * 2
			detailRow := nameRow + 1

			table.SetCell(nameRow, 0,
				tview.NewTableCell(" "+s.Name).
					SetTextColor(uiColorText).
					SetExpansion(1).
					SetSelectable(true),
			)

			users := a.cfg.Provider.UsersForScheme(s.Name)
			n := len(users)
			m := 0
			for _, u := range users {
				if models := a.cachedModels(s.Name, u.Name); len(models) > 0 {
					m++
				}
			}
			table.SetCell(detailRow, 0,
				tview.NewTableCell(fmt.Sprintf("  [%s](%d/%d) %s", uiTagMuted, m, n, s.BaseURL)).
					SetTextColor(uiColorMuted).
					SetExpansion(1).
					SetSelectable(false),
			)
			table.SetCell(detailRow, 1,
				tview.NewTableCell("["+uiTagGreen+"]"+s.Type+"  ").
					SetAlign(tview.AlignRight).
					SetSelectable(false),
			)
		}
		if selName != "" {
			for i, s := range schemes {
				if s.Name == selName {
					table.Select(i*2, 0)
					return
				}
			}
		}
		if table.GetRowCount() > 0 {
			table.Select(0, 0)
		}
	}
	rebuild()

	a.refreshModelCache(rebuild)
	a.pageRefreshFns["schemes"] = func() { a.refreshModelCache(rebuild) }

	table.SetSelectedFunc(func(row, _ int) {
		idx := rowToIdx(row)
		schemes := a.cfg.Provider.Schemes
		if idx < 0 || idx >= len(schemes) {
			return
		}
		name := schemes[idx].Name
		a.navigateTo("users", a.newUsersPage(name))
	})

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		row, _ := table.GetSelection()
		idx := rowToIdx(row)
		schemes := a.cfg.Provider.Schemes
		switch event.Rune() {
		case 'a':
			a.showSchemeForm(nil, func(s tuicfg.Scheme) {
				a.cfg.Provider.Schemes = append(a.cfg.Provider.Schemes, s)
				a.save()
				a.refreshModelCache(rebuild)
			})
			return nil
		case 'e':
			if idx < 0 || idx >= len(schemes) {
				return nil
			}
			origName := schemes[idx].Name
			orig := schemes[idx]
			a.showSchemeForm(&orig, func(s tuicfg.Scheme) {
				current := a.cfg.Provider.Schemes
				for i, sc := range current {
					if sc.Name == origName {
						a.cfg.Provider.Schemes[i] = s
						break
					}
				}
				a.save()
				a.refreshModelCache(func() {
					rebuild()
					for i, sc := range a.cfg.Provider.Schemes {
						if sc.Name == s.Name {
							table.Select(i*2, 0)
							break
						}
					}
				})
			})
			return nil
		case 'd':
			if idx < 0 || idx >= len(schemes) {
				return nil
			}
			name := schemes[idx].Name
			a.confirmDelete(fmt.Sprintf("scheme %q", name), func() {
				current := a.cfg.Provider.Schemes
				newSchemes := make([]tuicfg.Scheme, 0, len(current))
				for _, sc := range current {
					if sc.Name != name {
						newSchemes = append(newSchemes, sc)
					}
				}
				a.cfg.Provider.Schemes = newSchemes

				existing := a.cfg.Provider.Users
				filtered := make([]tuicfg.User, 0, len(existing))
				for _, u := range existing {
					if u.Scheme != name {
						filtered = append(filtered, u)
					}
				}
				a.cfg.Provider.Users = filtered

				a.save()
				a.refreshModelCache(rebuild)
			})
			return nil
		}
		return event
	})

	return a.buildShell(
		"schemes",
		table,
		" ["+uiTagRed+"]a:[-] add  ["+uiTagRed+"]e:[-] edit  ["+uiTagDanger+"]d:[-] delete  ["+uiTagGreenBold+"]Enter:[-] open  ["+uiTagMuted+"]ESC:[-] back ",
	)
}

func (a *App) showSchemeForm(existing *tuicfg.Scheme, onSave func(tuicfg.Scheme)) {
	name := ""
	baseURL := ""
	schemeType := "openai-compatible"
	title := " ADD SCHEME "

	if existing != nil {
		name = existing.Name
		baseURL = existing.BaseURL
		schemeType = existing.Type
		title = " EDIT SCHEME "
	}

	typeOptions := []string{"openai-compatible", "anthropic"}
	typeIdx := 0
	for i, t := range typeOptions {
		if t == schemeType {
			typeIdx = i
			break
		}
	}

	form := tview.NewForm()

	form.
		AddInputField("Name", name, 20, nil, func(text string) { name = text }).
		AddInputField("Base URL", baseURL, 28, nil, func(text string) { baseURL = text }).
		AddDropDown("Type", typeOptions, typeIdx, func(option string, _ int) { schemeType = option }).
		AddButton("SAVE", func() {
			if name == "" {
				a.showError("Name is required")
				return
			}
			if baseURL == "" {
				a.showError("Base URL is required")
				return
			}
			if existing == nil {
				for _, s := range a.cfg.Provider.Schemes {
					if s.Name == name {
						a.showError(fmt.Sprintf("Scheme name %q already exists", name))
						return
					}
				}
			}
			a.hideModal("scheme-form")
			onSave(tuicfg.Scheme{Name: name, BaseURL: baseURL, Type: schemeType})
		}).
		AddButton("CANCEL", func() {
			a.hideModal("scheme-form")
		})

	form.SetBorder(true).
		SetTitle(" [::b]" + title + " ").
		SetTitleColor(uiColorAccentRed).
		SetBorderColor(uiColorBorder)
	form.SetBackgroundColor(uiColorPanel)
	form.SetFieldBackgroundColor(uiColorBackground)
	form.SetFieldTextColor(uiColorAccentGreenBold)
	form.SetLabelColor(uiColorText)
	form.SetButtonBackgroundColor(uiColorAccentGreen)
	form.SetButtonTextColor(uiColorInverseText)
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			a.hideModal("scheme-form")
			return nil
		}
		return event
	})

	a.showModal("scheme-form", centeredForm(form, 4, 12))
}
