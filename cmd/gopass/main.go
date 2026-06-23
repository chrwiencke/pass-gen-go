package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"gopass/internal/clipboard"
	"gopass/internal/password"
	appsettings "gopass/internal/settings"
	"gopass/internal/settingsui"
	"gopass/internal/updater"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/systray"
)

const (
	appName     = "GoPass"
	githubOwner = "chrwiencke"
	githubRepo  = "pass-gen-go"
)

var version = "dev"
var guiApp fyne.App
var passwordSettings *appsettings.Manager
var passwordSettingsUI *settingsui.UI

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	guiApp = fyneapp.NewWithID("local.gopass.tray")

	settingsManager, err := appsettings.NewManager()
	if err != nil {
		log.Printf("settings load failed: %v", err)
	}
	passwordSettings = settingsManager
	passwordSettingsUI = settingsui.New(guiApp, settingsManager)

	startTray, stopTray := systray.RunWithExternalLoop(onReady, onExit)
	guiApp.Lifecycle().SetOnStarted(startTray)
	guiApp.Lifecycle().SetOnStopped(stopTray)
	guiApp.Run()
}

func onReady() {
	iconPNG := makeKeyIconPNG()

	systray.SetIcon(iconPNG)
	systray.SetTemplateIcon(iconPNG, iconPNG)
	systray.SetTitle(appName)
	systray.SetTooltip(appName + ": left-click to copy a password")

	copyItem := systray.AddMenuItem("Copy password", "Copy a password")
	settingsItem := systray.AddMenuItem("Settings...", "Change password generator settings")
	updateItem := systray.AddMenuItem("Update", "Update "+appName)
	updateItem.Hide()
	systray.AddSeparator()
	quitItem := systray.AddMenuItem("Quit", "Quit "+appName)

	updateMenu := newUpdateMenu(updateItem)
	updateMenu.start()

	// Left-clicking the macOS menu-bar icon or Windows taskbar tray icon copies a new password.
	systray.SetOnTapped(func() {
		copyPassword()
	})

	go func() {
		for range copyItem.ClickedCh {
			copyPassword()
		}
	}()

	go func() {
		for range settingsItem.ClickedCh {
			openSettings()
		}
	}()

	go func() {
		<-quitItem.ClickedCh
		if guiApp != nil {
			fyne.Do(guiApp.Quit)
			return
		}
		systray.Quit()
	}()
}

func onExit() {
	// Nothing to clean up.
}

func copyPassword() {
	settings := password.DefaultSettings()
	if passwordSettings != nil {
		settings = passwordSettings.Current()
	}

	pw, err := password.GenerateWithSettings(settings)
	if err != nil {
		systray.SetTooltip(appName + ": could not generate password")
		log.Printf("password generation failed: %v", err)
		return
	}

	if err := clipboard.CopyText(pw); err != nil {
		systray.SetTooltip(appName + ": clipboard is unavailable")
		log.Printf("clipboard copy failed: %v", err)
		return
	}

	systray.SetTooltip(fmt.Sprintf(appName+": password copied at %s", time.Now().Format("15:04:05")))

	// Do not log, display, or notify the actual password. It is only written to the clipboard.
}

func openSettings() {
	if passwordSettingsUI == nil {
		systray.SetTooltip(appName + ": settings are unavailable")
		return
	}
	if err := passwordSettingsUI.Open(); err != nil {
		systray.SetTooltip(appName + ": could not open settings")
		log.Printf("settings open failed: %v", err)
		return
	}
	systray.SetTooltip(appName + ": settings opened")
}

type updateMenu struct {
	item       *systray.MenuItem
	mu         sync.RWMutex
	available  *updater.AvailableUpdate
	lastCheck  time.Time
	checking   atomic.Bool
	installing atomic.Bool
}

func newUpdateMenu(item *systray.MenuItem) *updateMenu {
	return &updateMenu{item: item}
}

func (m *updateMenu) start() {
	go m.handleClicks()
	go m.check()

	go func() {
		for range systray.TrayOpenedCh {
			if m.shouldRefresh() {
				go m.check()
			}
		}
	}()
}

func (m *updateMenu) shouldRefresh() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return time.Since(m.lastCheck) >= 30*time.Minute
}

func (m *updateMenu) check() {
	if !m.checking.CompareAndSwap(false, true) {
		return
	}
	defer m.checking.Store(false)

	m.mu.Lock()
	m.lastCheck = time.Now()
	m.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	available, err := updater.CheckGitHubRelease(ctx, updater.Config{
		Owner:          githubOwner,
		Repo:           githubRepo,
		CurrentVersion: version,
	})
	if err != nil {
		log.Printf("update check failed: %v", err)
		return
	}

	m.mu.Lock()
	m.available = available
	m.mu.Unlock()

	if available == nil {
		m.item.Hide()
		return
	}

	m.item.SetTitle("Update to " + available.Version)
	m.item.SetTooltip("Install " + appName + " " + available.Version)
	m.item.Enable()
	m.item.Show()
}

func (m *updateMenu) handleClicks() {
	for range m.item.ClickedCh {
		m.mu.RLock()
		available := m.available
		m.mu.RUnlock()

		if available == nil {
			continue
		}
		if !m.installing.CompareAndSwap(false, true) {
			continue
		}

		update := *available
		m.item.SetTitle("Updating...")
		m.item.SetTooltip("Installing " + appName + " " + update.Version)
		m.item.Disable()

		go func() {
			defer m.installing.Store(false)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			if err := updater.ApplyGitHubRelease(ctx, update, nil); err != nil {
				log.Printf("update failed: %v", err)
				systray.SetTooltip(appName + ": update failed")
				m.item.SetTitle("Update to " + update.Version)
				m.item.SetTooltip("Install " + appName + " " + update.Version)
				m.item.Enable()
				return
			}

			m.mu.Lock()
			m.available = nil
			m.mu.Unlock()

			m.item.SetTitle("Updated to " + update.Version + " - restart")
			m.item.SetTooltip("Restart " + appName + " to use " + update.Version)
			m.item.Disable()
			systray.SetTooltip(appName + ": updated to " + update.Version + "; restart to use it")
		}()
	}
}

func makeKeyIconPNG() []byte {
	const size = 32

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	transparent := color.RGBA{0, 0, 0, 0}
	draw.Draw(img, img.Bounds(), &image.Uniform{transparent}, image.Point{}, draw.Src)

	// Pixel-art key tray icon. PNG is accepted by fyne.io/systray on macOS and Windows.
	key := color.RGBA{245, 190, 60, 255}
	shadow := color.RGBA{132, 88, 0, 255}

	// Slight shadow/outline first, then the key shape on top.
	drawFilledCircle(img, 11, 16, 8, shadow)
	drawFilledCircle(img, 11, 16, 4, transparent)
	draw.Draw(img, image.Rect(17, 14, 30, 19), &image.Uniform{shadow}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(23, 18, 27, 24), &image.Uniform{shadow}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(27, 18, 31, 22), &image.Uniform{shadow}, image.Point{}, draw.Src)

	drawFilledCircle(img, 10, 15, 8, key)
	drawFilledCircle(img, 10, 15, 4, transparent)
	draw.Draw(img, image.Rect(16, 13, 29, 18), &image.Uniform{key}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(22, 17, 26, 23), &image.Uniform{key}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(26, 17, 30, 21), &image.Uniform{key}, image.Point{}, draw.Src)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func drawFilledCircle(img *image.RGBA, centerX, centerY, radius int, c color.RGBA) {
	for y := centerY - radius; y <= centerY+radius; y++ {
		for x := centerX - radius; x <= centerX+radius; x++ {
			dx := x - centerX
			dy := y - centerY
			if dx*dx+dy*dy <= radius*radius && image.Pt(x, y).In(img.Bounds()) {
				img.SetRGBA(x, y, c)
			}
		}
	}
}
