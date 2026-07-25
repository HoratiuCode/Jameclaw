//go:build (!darwin && !freebsd) || cgo

package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"runtime"
	"sync"
	"time"

	"fyne.io/systray"

	"github.com/sipeed/jameclaw/pkg/logger"
	"github.com/sipeed/jameclaw/web/backend/utils"
)

const githubRepoURL = "https://github.com/HoratiuCode/Jameclaw"

var (
	trayIconOnce   sync.Once
	trayIconNormal []byte
	trayIconActive []byte
)

func runTray() {
	systray.Run(onReady, onExit)
}

// requestLauncherQuit is shared by the menu bar and the native Jame app so
// either way of stopping Jame follows the same full shutdown path.
func requestLauncherQuit() {
	systray.Quit()
}

// onReady is called when the system tray is ready
func onReady() {
	installDesktopMenu()

	// A template icon lets macOS choose the correct foreground automatically:
	// dark on light menu bars and white on dark menu bars. The source artwork is
	// used as an alpha mask by AppKit, so it stays legible over wallpaper too.
	setTrayIcon(getIcon())
	systray.SetTooltip(fmt.Sprintf(T(AppTooltip), appName))
	startTrayActivityIndicator()

	// Create menu items
	// Top-level: Open, New Chat, About, Settings, Quit — keep the bar clean on macOS.
	mOpen := systray.AddMenuItem(T(MenuOpen), T(MenuOpenTooltip))
	mConsole := mOpen.AddSubMenuItem(T(MenuConsole), T(MenuConsoleTooltip))
	mNewChat := systray.AddMenuItem(T(MenuNewChat), T(MenuNewChatTooltip))
	var terminalClicked <-chan struct{}
	if runtime.GOOS == "darwin" {
		mTerminalChat := mOpen.AddSubMenuItem(T(MenuTerminalChat), T(MenuTerminalTooltip))
		terminalClicked = mTerminalChat.ClickedCh
	}
	mAbout := systray.AddMenuItem(T(MenuAbout), T(MenuAboutTooltip))

	// Add version info under About menu
	mVersion := mAbout.AddSubMenuItem(fmt.Sprintf(T(MenuVersion), appVersion), T(MenuVersionTooltip))
	mVersion.Disable()
	mRepo := mAbout.AddSubMenuItem(T(MenuGitHub), "")
	mDocs := mAbout.AddSubMenuItem(T(MenuDocs), "")

	systray.AddSeparator()

	// Settings submenu: service + power options (Restart, Keep Awake on macOS)
	mSettings := systray.AddMenuItem(T(MenuSettings), T(MenuSettingsTooltip))
	var nativeSettingsClicked <-chan struct{}
	if runtime.GOOS == "darwin" {
		mNativeSettings := mSettings.AddSubMenuItem("Open native settings", "Open simple JameClaw settings")
		nativeSettingsClicked = mNativeSettings.ClickedCh
	}
	mRestart := mSettings.AddSubMenuItem(T(MenuRestart), T(MenuRestartTooltip))
	var keepAwakeClicked <-chan struct{}
	var mKeepAwake *systray.MenuItem
	if runtime.GOOS == "darwin" {
		mKeepAwake = mSettings.AddSubMenuItemCheckbox(T(MenuKeepAwake), T(MenuKeepAwakeTooltip), false)
		keepAwakeClicked = mKeepAwake.ClickedCh
	}

	systray.AddSeparator()

	// Quit option
	mQuit := systray.AddMenuItem(T(MenuQuit), T(MenuQuitTooltip))

	// Handle menu clicks
	go func() {
		keepAwakeChecked := false
		for {
			select {
			case <-mConsole.ClickedCh:
				if err := openBrowser(); err != nil {
					logger.Errorf("Failed to open browser: %v", err)
				}

			case <-mNewChat.ClickedCh:
				if err := openNewChat(); err != nil {
					logger.Errorf("Failed to start a new chat: %v", err)
				}

			case <-terminalClicked:
				if err := openTerminalChat(); err != nil {
					logger.Errorf("Failed to open terminal chat: %v", err)
				}

			case <-mVersion.ClickedCh:
				// Version info - do nothing, just shows current version

			case <-mRepo.ClickedCh:
				if err := utils.OpenBrowser(githubRepoURL); err != nil {
					logger.Errorf("Failed to open GitHub: %v", err)
				}

			case <-mDocs.ClickedCh:
				if err := utils.OpenBrowser(T(DocUrl)); err != nil {
					logger.Errorf("Failed to open docs: %v", err)
				}

			case <-mRestart.ClickedCh:
				fmt.Println("Restart request received...")
				if apiHandler != nil {
					if pid, err := apiHandler.RestartGateway(); err != nil {
						logger.Errorf("Failed to restart gateway: %v", err)
					} else {
						logger.Infof("Gateway restarted (PID: %d)", pid)
					}
				}

			case <-nativeSettingsClicked:
				if err := openNativeSettings(); err != nil {
					logger.Errorf("Failed to open native settings: %v", err)
				}

			case <-keepAwakeClicked:
				next := !keepAwakeChecked
				if err := setKeepAwake(next); err != nil {
					logger.Errorf("Failed to update keep-awake mode: %v", err)
					if mKeepAwake != nil {
						mKeepAwake.Uncheck()
					}
					keepAwakeChecked = false
					continue
				}
				keepAwakeChecked = next
				if mKeepAwake != nil {
					if keepAwakeChecked {
						mKeepAwake.Check()
					} else {
						mKeepAwake.Uncheck()
					}
				}

			case <-mQuit.ClickedCh:
				requestLauncherQuit()
			}
		}
	}()

	if !*noBrowser {
		if err := openNativeHome(); err != nil {
			logger.Errorf("Warning: Failed to open native home: %v", err)
		}
	}
}

// onExit is called when the system tray is exiting
func onExit() {
	closeNativeHome()
	stopKeepAwake()
	logger.Info(T(Exiting))
}

// getIcon returns the system tray icon
func getIcon() []byte {
	return iconData
}

func startTrayActivityIndicator() {
	go func() {
		active := false
		applyTrayActivityIcon(active)

		ticker := time.NewTicker(750 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			next := apiHandler != nil && apiHandler.AgentActivityActive()
			if next == active {
				continue
			}
			active = next
			applyTrayActivityIcon(active)
		}
	}()
}

func applyTrayActivityIcon(active bool) {
	normal, activeIcon := trayIcons()
	if active {
		setTrayIcon(activeIcon)
		systray.SetTooltip(fmt.Sprintf("%s - Agent active", fmt.Sprintf(T(AppTooltip), appName)))
		return
	}
	setTrayIcon(normal)
	systray.SetTooltip(fmt.Sprintf(T(AppTooltip), appName))
}

// setTrayIcon uses the native template mode on macOS. Other platforms fall
// back to the regular image through the systray library, so this remains
// cross-platform while making the menu-bar icon appearance-aware on macOS.
func setTrayIcon(icon []byte) {
	systray.SetTemplateIcon(icon, icon)
}

func trayIcons() ([]byte, []byte) {
	trayIconOnce.Do(func() {
		trayIconNormal = iconData
		trayIconActive = buildActiveTrayIcon(iconData)
	})
	return trayIconNormal, trayIconActive
}

func buildActiveTrayIcon(src []byte) []byte {
	img, err := png.Decode(bytes.NewReader(src))
	if err != nil {
		return src
	}

	bounds := img.Bounds()
	dst := image.NewRGBA(bounds)
	draw.Draw(dst, bounds, img, bounds.Min, draw.Src)

	size := min(bounds.Dx(), bounds.Dy())
	if size <= 0 {
		return src
	}

	radius := max(3, size/7)
	cx := bounds.Max.X - radius - max(1, size/12)
	cy := bounds.Max.Y - radius - max(1, size/12)
	border := max(1, radius/3)

	drawCircle(dst, cx, cy, radius+border, color.RGBA{R: 255, G: 255, B: 255, A: 230})
	drawCircle(dst, cx, cy, radius, color.RGBA{R: 42, G: 196, B: 91, A: 255})

	var out bytes.Buffer
	if err := png.Encode(&out, dst); err != nil {
		return src
	}
	return out.Bytes()
}

func drawCircle(dst *image.RGBA, cx, cy, radius int, c color.Color) {
	if radius <= 0 {
		return
	}
	r2 := radius * radius
	for y := cy - radius; y <= cy+radius; y++ {
		for x := cx - radius; x <= cx+radius; x++ {
			dx := x - cx
			dy := y - cy
			if dx*dx+dy*dy <= r2 {
				dst.Set(x, y, c)
			}
		}
	}
}
