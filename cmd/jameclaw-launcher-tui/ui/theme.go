package ui

import "github.com/gdamore/tcell/v2"

var (
	uiColorBackground      = tcell.NewHexColor(0x0f120d)
	uiColorPanel           = tcell.NewHexColor(0x171d14)
	uiColorPanelAlt        = tcell.NewHexColor(0x232b1f)
	uiColorBorder          = tcell.NewHexColor(0xe05a47)
	uiColorAccentRed       = tcell.NewHexColor(0xe05a47)
	uiColorAccentGreen     = tcell.NewHexColor(0x78c458)
	uiColorAccentGreenBold = tcell.NewHexColor(0xa4e86d)
	uiColorText            = tcell.NewHexColor(0xf2efe8)
	uiColorMuted           = tcell.NewHexColor(0x8c927d)
	uiColorDanger          = tcell.NewHexColor(0xf06253)
	uiColorInverseText     = tcell.NewHexColor(0x0f120d)

	uiSelectedStyle = tcell.StyleDefault.
			Background(uiColorAccentGreen).
			Foreground(uiColorInverseText)
)

const (
	uiTagRed         = "#e05a47"
	uiTagGreen       = "#78c458"
	uiTagGreenBold   = "#a4e86d"
	uiTagText        = "#f2efe8"
	uiTagMuted       = "#8c927d"
	uiTagDanger      = "#f06253"
	uiTagDangerLabel = "red"
	uiTagMutedLabel  = "gray"
)
