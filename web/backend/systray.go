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

// onReady is called when the system tray is ready
func onReady() {
	// Set icon and tooltip
	systray.SetIcon(getIcon())
	systray.SetTooltip(fmt.Sprintf(T(AppTooltip), appName))
	startTrayActivityIndicator()

	// Create menu items
	mOpen := systray.AddMenuItem(T(MenuOpen), T(MenuOpenTooltip))
	mConsole := mOpen.AddSubMenuItem(T(MenuConsole), T(MenuConsoleTooltip))
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

	// Add restart option
	mRestart := systray.AddMenuItem(T(MenuRestart), T(MenuRestartTooltip))
	var keepAwakeClicked <-chan struct{}
	var mKeepAwake *systray.MenuItem
	if runtime.GOOS == "darwin" {
		mKeepAwake = systray.AddMenuItemCheckbox(T(MenuKeepAwake), T(MenuKeepAwakeTooltip), false)
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
				systray.Quit()
			}
		}
	}()

	if !*noBrowser {
		// Auto-open browser after systray is ready (if not disabled)
		// Check no-browser flag via environment or pass as parameter if needed
		if err := openBrowserBackground(); err != nil {
			logger.Errorf("Warning: Failed to auto-open browser: %v", err)
		}
	}
}

// onExit is called when the system tray is exiting
func onExit() {
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
			next := apiHandler != nil && apiHandler.WebActivityActive()
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
		systray.SetIcon(activeIcon)
		systray.SetTooltip(fmt.Sprintf("%s - Web active", fmt.Sprintf(T(AppTooltip), appName)))
		return
	}
	systray.SetIcon(normal)
	systray.SetTooltip(fmt.Sprintf(T(AppTooltip), appName))
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
