package main

import (
	"fmt"
	"os"
	"strings"
)

// Language represents the supported languages
type Language string

const (
	LanguageEnglish Language = "en"
	LanguageChinese Language = "zh"
)

// current language (default: English)
var currentLang Language = LanguageEnglish

// TranslationKey represents a translation key used for i18n
type TranslationKey string

const (
	AppTooltip         TranslationKey = "AppTooltip"
	MenuOpen           TranslationKey = "MenuOpen"
	MenuOpenTooltip    TranslationKey = "MenuOpenTooltip"
	MenuAbout          TranslationKey = "MenuAbout"
	MenuAboutTooltip   TranslationKey = "MenuAboutTooltip"
	MenuVersion        TranslationKey = "MenuVersion"
	MenuVersionTooltip TranslationKey = "MenuVersionTooltip"
	MenuGitHub         TranslationKey = "MenuGitHub"
	MenuDocs           TranslationKey = "MenuDocs"
	MenuRestart        TranslationKey = "MenuRestart"
	MenuRestartTooltip TranslationKey = "MenuRestartTooltip"
	MenuQuit           TranslationKey = "MenuQuit"
	MenuQuitTooltip    TranslationKey = "MenuQuitTooltip"
	Exiting            TranslationKey = "Exiting"
	DocUrl             TranslationKey = "DocUrl"
)

// Translation tables
var translations = map[Language]map[TranslationKey]string{
	LanguageEnglish: {
		AppTooltip:         "%s - Web Console",
		MenuOpen:           "Open Console",
		MenuOpenTooltip:    "Open JameClaw console in browser",
		MenuAbout:          "About",
		MenuAboutTooltip:   "About JameClaw",
		MenuVersion:        "Version: %s",
		MenuVersionTooltip: "Current version number",
		MenuGitHub:         "GitHub",
		MenuDocs:           "Documentation",
		MenuRestart:        "Restart Service",
		MenuRestartTooltip: "Restart Gateway service",
		MenuQuit:           "Quit",
		MenuQuitTooltip:    "Exit JameClaw",
		Exiting:            "Exiting JameClaw...",
		DocUrl:             "https://docs.jameclaw.io/docs/",
	},
	LanguageChinese: {
		AppTooltip:         "%s - Web Console",
		MenuOpen:           "Open Console",
		MenuOpenTooltip:    "Open JameClaw console in browser",
		MenuAbout:          "About",
		MenuAboutTooltip:   "About JameClaw",
		MenuVersion:        "Version: %s",
		MenuVersionTooltip: "Current version number",
		MenuGitHub:         "GitHub",
		MenuDocs:           "Documentation",
		MenuRestart:        "Restart Service",
		MenuRestartTooltip: "Restart Gateway service",
		MenuQuit:           "Quit",
		MenuQuitTooltip:    "Exit JameClaw",
		Exiting:            "Exiting JameClaw...",
		DocUrl:             "https://docs.jameclaw.io/docs/",
	},
}

// SetLanguage sets the current language
func SetLanguage(lang string) {
	lang = strings.ToLower(strings.TrimSpace(lang))

	// Extract language code before first underscore or dot
	// e.g., "en_US.UTF-8" -> "en", "zh_CN" -> "zh"
	if idx := strings.IndexAny(lang, "_."); idx > 0 {
		lang = lang[:idx]
	}

	if lang == "zh" || lang == "zh-cn" || lang == "chinese" {
		currentLang = LanguageChinese
	} else {
		currentLang = LanguageEnglish
	}
}

// GetLanguage returns the current language
func GetLanguage() Language {
	return currentLang
}

// T translates a key to the current language
func T(key TranslationKey, args ...any) string {
	if trans, ok := translations[currentLang][key]; ok {
		if len(args) > 0 {
			return fmt.Sprintf(trans, args...)
		}
		return trans
	}
	return string(key)
}

// Initialize i18n from environment variable
func init() {
	if lang := os.Getenv("LANG"); lang != "" {
		SetLanguage(lang)
	}
}
