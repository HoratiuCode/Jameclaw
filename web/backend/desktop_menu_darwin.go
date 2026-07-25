//go:build darwin && cgo

package main

/*
#cgo LDFLAGS: -framework Cocoa
#include "desktop_menu_darwin.h"
*/
import "C"

import (
	"github.com/sipeed/jameclaw/pkg/logger"
	"github.com/sipeed/jameclaw/web/backend/utils"
)

func installDesktopMenu() {
	C.jameclawInstallDesktopMenu()
}

//export jameclawMenuNewChat
func jameclawMenuNewChat() {
	go func() {
		if err := openNewChat(); err != nil {
			logger.Errorf("Failed to start a new chat from the app menu: %v", err)
		}
	}()
}

//export jameclawMenuAutomations
func jameclawMenuAutomations() {
	go func() {
		if err := utils.OpenBrowser(launcherOpenURL(serverAddr + "/automation")); err != nil {
			logger.Errorf("Failed to open automations from the app menu: %v", err)
		}
	}()
}

//export jameclawMenuShowDesktop
func jameclawMenuShowDesktop() {
	go func() {
		if err := openNativeHome(); err != nil {
			logger.Errorf("Failed to show JameClaw Desktop from the app menu: %v", err)
		}
	}()
}

//export jameclawMenuShowConsole
func jameclawMenuShowConsole() {
	go func() {
		if err := openBrowser(); err != nil {
			logger.Errorf("Failed to open the web console from the app menu: %v", err)
		}
	}()
}
