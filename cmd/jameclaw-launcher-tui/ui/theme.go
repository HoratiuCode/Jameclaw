package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"gopkg.in/yaml.v3"
)

type skinPalette struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Colors      map[string]string `yaml:"colors"`
	Branding    map[string]string `yaml:"branding"`
	ToolPrefix  string            `yaml:"tool_prefix"`
}

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

	uiTagRed         = "#e05a47"
	uiTagGreen       = "#78c458"
	uiTagGreenBold   = "#a4e86d"
	uiTagText        = "#f2efe8"
	uiTagMuted       = "#8c927d"
	uiTagDanger      = "#f06253"
	uiTagDangerLabel = "red"
	uiTagMutedLabel  = "gray"

	uiSelectedStyle = tcell.StyleDefault.
			Background(uiColorAccentGreen).
			Foreground(uiColorInverseText)

	currentSkinName = "default"
	currentSkinDesc = "Classic JameClaw"
	currentAgentName = "JameClaw Launcher"
	currentPrompt    = "❯ "
	currentToolPrefix = "┊"
)

func applySkin(name string) {
	palette := loadSkin(name)
	currentSkinName = palette.Name
	currentSkinDesc = palette.Description
	currentAgentName = palette.String("branding.agent_name", "JameClaw Launcher")
	currentPrompt = palette.String("branding.prompt_symbol", "❯ ")
	currentToolPrefix = palette.String("tool_prefix", "┊")

	uiColorBackground = parseColor(palette.String("colors.background", "#0f120d"), uiColorBackground)
	uiColorPanel = parseColor(palette.String("colors.panel", "#171d14"), uiColorPanel)
	uiColorPanelAlt = parseColor(palette.String("colors.panel_alt", "#232b1f"), uiColorPanelAlt)
	uiColorBorder = parseColor(palette.String("colors.border", "#e05a47"), uiColorBorder)
	uiColorAccentRed = parseColor(palette.String("colors.accent_red", "#e05a47"), uiColorAccentRed)
	uiColorAccentGreen = parseColor(palette.String("colors.accent_green", "#78c458"), uiColorAccentGreen)
	uiColorAccentGreenBold = parseColor(palette.String("colors.accent_green_bold", "#a4e86d"), uiColorAccentGreenBold)
	uiColorText = parseColor(palette.String("colors.text", "#f2efe8"), uiColorText)
	uiColorMuted = parseColor(palette.String("colors.muted", "#8c927d"), uiColorMuted)
	uiColorDanger = parseColor(palette.String("colors.danger", "#f06253"), uiColorDanger)
	uiColorInverseText = parseColor(palette.String("colors.inverse_text", "#0f120d"), uiColorInverseText)

	uiTagRed = palette.String("colors.border", "#e05a47")
	uiTagGreen = palette.String("colors.accent_green", "#78c458")
	uiTagGreenBold = palette.String("colors.accent_green_bold", "#a4e86d")
	uiTagText = palette.String("colors.text", "#f2efe8")
	uiTagMuted = palette.String("colors.muted", "#8c927d")
	uiTagDanger = palette.String("colors.danger", "#f06253")
}

func parseColor(value string, fallback tcell.Color) tcell.Color {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	if !strings.HasPrefix(value, "#") {
		value = "#" + value
	}
	var n uint64
	if _, err := fmt.Sscanf(value, "#%06x", &n); err == nil {
		return tcell.NewHexColor(int32(n))
	}
	return fallback
}

func loadSkin(name string) skinPalette {
	if strings.TrimSpace(name) == "" {
		name = "default"
	}

	if user, ok := loadUserSkin(name); ok {
		return mergeSkin(defaultSkinPalette(), user)
	}
	if builtin, ok := builtinSkin(name); ok {
		return mergeSkin(defaultSkinPalette(), builtin)
	}
	return defaultSkinPalette()
}

func availableSkins() []skinPalette {
	skins := make([]skinPalette, 0, 8)
	seen := map[string]struct{}{}
	appendSkin := func(s skinPalette) {
		name := strings.TrimSpace(s.Name)
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		skins = append(skins, s)
	}

	appendSkin(defaultSkinPalette())
	for _, name := range []string{"ares", "mono", "slate"} {
		if skin, ok := builtinSkin(name); ok {
			appendSkin(skin)
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return skins
	}
	matches, err := filepath.Glob(filepath.Join(home, ".jameclaw", "skins", "*.yaml"))
	if err != nil {
		return skins
	}
	for _, match := range matches {
		base := strings.TrimSuffix(filepath.Base(match), filepath.Ext(match))
		if base == "" {
			continue
		}
		if skin, ok := loadUserSkin(base); ok {
			appendSkin(skin)
		}
	}
	return skins
}

func loadUserSkin(name string) (skinPalette, bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return skinPalette{}, false
	}
	path := filepath.Join(home, ".jameclaw", "skins", name+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return skinPalette{}, false
	}
	var skin skinPalette
	if err := yaml.Unmarshal(data, &skin); err != nil {
		return skinPalette{}, false
	}
	if strings.TrimSpace(skin.Name) == "" {
		skin.Name = name
	}
	return skin, true
}

func builtinSkin(name string) (skinPalette, bool) {
	skins := map[string]skinPalette{
		"default": {
			Name:        "default",
			Description: "Classic JameClaw gold and ember",
			Colors: map[string]string{
				"background":       "#0f120d",
				"panel":            "#171d14",
				"panel_alt":        "#232b1f",
				"border":           "#e05a47",
				"accent_red":       "#e05a47",
				"accent_green":     "#78c458",
				"accent_green_bold":"#a4e86d",
				"text":             "#f2efe8",
				"muted":            "#8c927d",
				"danger":           "#f06253",
				"inverse_text":     "#0f120d",
			},
			Branding: map[string]string{
				"agent_name":   "JameClaw Launcher",
				"prompt_symbol":"❯ ",
			},
			ToolPrefix: "┊",
		},
		"ares": {
			Name:        "ares",
			Description: "Crimson and bronze war-god theme",
			Colors: map[string]string{
				"background":       "#120e0d",
				"panel":            "#1e1412",
				"panel_alt":        "#2b1b18",
				"border":           "#9f1c1c",
				"accent_red":       "#dd4a3a",
				"accent_green":     "#63d0a6",
				"accent_green_bold":"#a9dfff",
				"text":             "#f1e6cf",
				"muted":            "#a88a77",
				"danger":           "#ef5350",
				"inverse_text":     "#120e0d",
			},
			Branding: map[string]string{
				"agent_name":   "Ares Claw",
				"prompt_symbol":"⚔ ❯ ",
			},
			ToolPrefix: "╎",
		},
		"mono": {
			Name:        "mono",
			Description: "Clean grayscale terminal theme",
			Colors: map[string]string{
				"background":       "#101010",
				"panel":            "#181818",
				"panel_alt":        "#202020",
				"border":           "#888888",
				"accent_red":       "#cccccc",
				"accent_green":     "#d5d5d5",
				"accent_green_bold":"#ffffff",
				"text":             "#f2f2f2",
				"muted":            "#9a9a9a",
				"danger":           "#d0d0d0",
				"inverse_text":     "#101010",
			},
			Branding: map[string]string{
				"agent_name":   "JameClaw Mono",
				"prompt_symbol":"❯ ",
			},
			ToolPrefix: "│",
		},
		"slate": {
			Name:        "slate",
			Description: "Cool blue developer theme",
			Colors: map[string]string{
				"background":       "#0f1418",
				"panel":            "#171f26",
				"panel_alt":        "#1e2931",
				"border":           "#5db8f5",
				"accent_red":       "#5db8f5",
				"accent_green":     "#63d0a6",
				"accent_green_bold":"#a9dfff",
				"text":             "#e7eef5",
				"muted":            "#93a1ad",
				"danger":           "#f7a072",
				"inverse_text":     "#0f1418",
			},
			Branding: map[string]string{
				"agent_name":   "JameClaw Slate",
				"prompt_symbol":"❯ ",
			},
			ToolPrefix: "┊",
		},
	}
	skin, ok := skins[name]
	return skin, ok
}

func defaultSkinPalette() skinPalette {
	skin, _ := builtinSkin("default")
	return skin
}

func mergeSkin(base, override skinPalette) skinPalette {
	if strings.TrimSpace(override.Name) != "" {
		base.Name = override.Name
	}
	if strings.TrimSpace(override.Description) != "" {
		base.Description = override.Description
	}
	if base.Colors == nil {
		base.Colors = map[string]string{}
	}
	for key, value := range override.Colors {
		if strings.TrimSpace(value) != "" {
			base.Colors[key] = value
		}
	}
	if base.Branding == nil {
		base.Branding = map[string]string{}
	}
	for key, value := range override.Branding {
		if strings.TrimSpace(value) != "" {
			base.Branding[key] = value
		}
	}
	if strings.TrimSpace(override.ToolPrefix) != "" {
		base.ToolPrefix = override.ToolPrefix
	}
	return base
}

func (s skinPalette) String(path, fallback string) string {
	switch path {
	case "tool_prefix":
		if strings.TrimSpace(s.ToolPrefix) != "" {
			return s.ToolPrefix
		}
		return fallback
	case "branding.agent_name":
		if v := strings.TrimSpace(s.Branding["agent_name"]); v != "" {
			return v
		}
		return fallback
	case "branding.prompt_symbol":
		if v := strings.TrimSpace(s.Branding["prompt_symbol"]); v != "" {
			return v
		}
		return fallback
	case "colors.background":
		return lookupOrFallback(s.Colors, "background", fallback)
	case "colors.panel":
		return lookupOrFallback(s.Colors, "panel", fallback)
	case "colors.panel_alt":
		return lookupOrFallback(s.Colors, "panel_alt", fallback)
	case "colors.border":
		return lookupOrFallback(s.Colors, "border", fallback)
	case "colors.accent_red":
		return lookupOrFallback(s.Colors, "accent_red", fallback)
	case "colors.accent_green":
		return lookupOrFallback(s.Colors, "accent_green", fallback)
	case "colors.accent_green_bold":
		return lookupOrFallback(s.Colors, "accent_green_bold", fallback)
	case "colors.text":
		return lookupOrFallback(s.Colors, "text", fallback)
	case "colors.muted":
		return lookupOrFallback(s.Colors, "muted", fallback)
	case "colors.danger":
		return lookupOrFallback(s.Colors, "danger", fallback)
	case "colors.inverse_text":
		return lookupOrFallback(s.Colors, "inverse_text", fallback)
	default:
		return fallback
	}
}

func lookupOrFallback(values map[string]string, key, fallback string) string {
	if values == nil {
		return fallback
	}
	if v := strings.TrimSpace(values[key]); v != "" {
		return v
	}
	return fallback
}
