package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"gopass/internal/clipboard"
	"gopass/internal/hotkey"
	"gopass/internal/password"
	"gopass/internal/paste"
	appsettings "gopass/internal/settings"
	"gopass/internal/settingsui"
	"gopass/internal/updater"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/systray"
)

const (
	appName             = "GoPass"
	githubOwner         = "chrwiencke"
	githubRepo          = "pass-gen-go"
	gracefulQuitTimeout = 5 * time.Second
)

var version = "dev"
var guiApp fyne.App
var passwordSettings *appsettings.Manager
var passwordSettingsUI *settingsui.UI
var pasteHotkey *hotkey.Manager
var appUpdateMenu *updateMenu
var shutdownOnce sync.Once
var templatesMenu *systray.MenuItem
var templateMenuItemsMu sync.Mutex
var templateMenuItems []*systray.MenuItem

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	enforceMenuBarOnly()
	guiApp = fyneapp.NewWithID("local.gopass.tray")

	settingsManager, err := appsettings.NewManager()
	if err != nil {
		log.Printf("settings load failed: %v", err)
	}
	passwordSettings = settingsManager
	pasteHotkey = hotkey.New(pastePassword)
	passwordSettingsUI = settingsui.New(guiApp, settingsManager, settingsSaved, version)

	startTray, _ := systray.RunWithExternalLoop(onReady, onExit)
	guiApp.Lifecycle().SetOnStarted(func() {
		// GLFW promotes unbundled macOS apps to a regular Dock app during init.
		enforceMenuBarOnly()
		startTray()
		enforceMenuBarOnly()
	})
	guiApp.Run()
}

func onReady() {
	icon := makeKeyIcon()

	systray.SetIcon(icon)
	systray.SetTemplateIcon(icon, icon)
	systray.SetTitle(appName)
	systray.SetTooltip(appName + ": left-click to copy a password")

	copyItem := systray.AddMenuItem("Copy password", "Copy a password")
	templatesMenu = systray.AddMenuItem("Templates", "Copy a password using a template")
	refreshTemplateMenu()
	settingsItem := systray.AddMenuItem("Settings...", "Change password generator settings")
	updateItem := systray.AddMenuItem("Update", "Update "+appName)
	updateItem.Hide()
	systray.AddSeparator()
	quitItem := systray.AddMenuItem("Quit", "Quit "+appName)

	appUpdateMenu = newUpdateMenu(updateItem)
	appUpdateMenu.start()
	configurePasteHotkey()

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
		quitApp()
	}()
}

func onExit() {
	// Nothing to clean up.
}

func quitApp() {
	shutdownOnce.Do(func() {
		systray.SetTooltip(appName + ": quitting")
		time.AfterFunc(gracefulQuitTimeout, func() {
			log.Printf("forced exit after waiting %s for graceful quit", gracefulQuitTimeout)
			os.Exit(0)
		})

		go systray.Quit()

		if passwordSettingsUI != nil {
			passwordSettingsUI.Close()
		}
		if pasteHotkey != nil {
			pasteHotkey.Close()
		}
		if guiApp != nil {
			fyne.Do(guiApp.Quit)
			return
		}
		os.Exit(0)
	})
}

func copyPassword() {
	if _, ok := generateAndCopyPassword(); !ok {
		return
	}

	systray.SetTooltip(fmt.Sprintf(appName+": password copied at %s", time.Now().Format("15:04:05")))

	// Do not log, display, or notify the actual password. It is only written to the clipboard.
}

func copyPasswordTemplate(template password.Template) {
	if _, ok := generateAndCopyPasswordWithSettings(template.Settings); !ok {
		return
	}

	systray.SetTooltip(fmt.Sprintf(appName+": %s copied at %s", template.Name, time.Now().Format("15:04:05")))
}

func pastePassword() {
	if _, ok := generateAndCopyPassword(); !ok {
		return
	}

	time.Sleep(80 * time.Millisecond)
	if err := paste.Send(); err != nil {
		systray.SetTooltip(appName + ": could not paste password")
		log.Printf("paste failed: %v", err)
		return
	}

	systray.SetTooltip(fmt.Sprintf(appName+": password pasted at %s", time.Now().Format("15:04:05")))
}

func generateAndCopyPassword() (string, bool) {
	settings := password.DefaultSettings()
	if passwordSettings != nil {
		settings = passwordSettings.Current()
	}

	return generateAndCopyPasswordWithSettings(settings)
}

func generateAndCopyPasswordWithSettings(settings password.Settings) (string, bool) {
	pw, err := password.GenerateWithSettings(settings)
	if err != nil {
		systray.SetTooltip(appName + ": could not generate password")
		log.Printf("password generation failed: %v", err)
		return "", false
	}

	if err := clipboard.CopyText(pw); err != nil {
		systray.SetTooltip(appName + ": clipboard is unavailable")
		log.Printf("clipboard copy failed: %v", err)
		return "", false
	}

	return pw, true
}

func refreshTemplateMenu() {
	if templatesMenu == nil {
		return
	}

	templateMenuItemsMu.Lock()
	defer templateMenuItemsMu.Unlock()

	for _, item := range templateMenuItems {
		item.Remove()
	}
	templateMenuItems = nil

	templates := []password.Template(nil)
	if passwordSettings != nil {
		templates = passwordSettings.Templates()
	}
	if len(templates) == 0 {
		templatesMenu.Disable()
		return
	}

	templatesMenu.Enable()
	for _, template := range templates {
		template := template
		item := templatesMenu.AddSubMenuItem(template.Name, "Copy a password using "+template.Name)
		templateMenuItems = append(templateMenuItems, item)
		go func() {
			for range item.ClickedCh {
				copyPasswordTemplate(template)
			}
		}()
	}
}

func settingsSaved() {
	configurePasteHotkey()
	refreshTemplateMenu()
	refreshUpdateMenu()
}

func configurePasteHotkey() {
	if pasteHotkey == nil || passwordSettings == nil {
		return
	}
	if err := pasteHotkey.SetShortcut(passwordSettings.PasteShortcut()); err != nil {
		systray.SetTooltip(appName + ": paste shortcut unavailable")
		log.Printf("paste shortcut registration failed: %v", err)
		return
	}
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

func refreshUpdateMenu() {
	if appUpdateMenu != nil {
		appUpdateMenu.preferenceChanged()
	}
}

func automaticUpdatesEnabled() bool {
	return passwordSettings != nil && passwordSettings.AutomaticUpdates()
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
	if automaticUpdatesEnabled() {
		go m.check()
	}

	go func() {
		for range systray.TrayOpenedCh {
			if m.shouldRefresh() {
				go m.check()
			}
		}
	}()
}

func (m *updateMenu) shouldRefresh() bool {
	if !automaticUpdatesEnabled() {
		return false
	}
	if m.installing.Load() || m.checking.Load() {
		return false
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	return time.Since(m.lastCheck) >= 30*time.Minute
}

func (m *updateMenu) preferenceChanged() {
	if automaticUpdatesEnabled() {
		go m.check()
		return
	}
	m.clear()
}

func (m *updateMenu) clear() {
	if m.installing.Load() {
		return
	}
	m.mu.Lock()
	m.available = nil
	m.mu.Unlock()
	m.item.SetTitle("Update")
	m.item.SetTooltip("Update " + appName)
	m.item.Hide()
}

func (m *updateMenu) check() {
	if !automaticUpdatesEnabled() {
		m.clear()
		return
	}
	if m.installing.Load() {
		return
	}
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

	if !automaticUpdatesEnabled() {
		m.clear()
		return
	}

	if m.installing.Load() {
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

			m.item.SetTitle("Restarting...")
			m.item.SetTooltip("Restarting " + appName + " " + update.Version)
			m.item.Disable()

			if err := relaunchApp(); err != nil {
				log.Printf("relaunch after update failed: %v", err)
				m.item.SetTitle("Updated to " + update.Version)
				m.item.SetTooltip("Restart " + appName + " to use " + update.Version)
				systray.SetTooltip(appName + ": updated to " + update.Version + "; restart failed")
				return
			}

			systray.SetTooltip(appName + ": updated to " + update.Version + "; restarting")
			quitApp()
		}()
	}
}

func makeKeyIcon() []byte {
	if runtime.GOOS == "windows" {
		return makeKeyIconICO()
	}
	return makeKeyIconPNG(32)
}

func makeKeyIconPNG(size int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	transparent := color.RGBA{0, 0, 0, 0}
	draw.Draw(img, img.Bounds(), &image.Uniform{transparent}, image.Point{}, draw.Src)

	// Pixel-art key tray icon.
	key := color.RGBA{245, 190, 60, 255}
	shadow := color.RGBA{132, 88, 0, 255}
	scale := func(v int) int {
		scaled := v * size / 32
		if scaled < 1 {
			return 1
		}
		return scaled
	}

	// Slight shadow/outline first, then the key shape on top.
	drawFilledCircle(img, scale(11), scale(16), scale(8), shadow)
	drawFilledCircle(img, scale(11), scale(16), scale(4), transparent)
	draw.Draw(img, image.Rect(scale(17), scale(14), scale(30), scale(19)), &image.Uniform{shadow}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(scale(23), scale(18), scale(27), scale(24)), &image.Uniform{shadow}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(scale(27), scale(18), scale(31), scale(22)), &image.Uniform{shadow}, image.Point{}, draw.Src)

	drawFilledCircle(img, scale(10), scale(15), scale(8), key)
	drawFilledCircle(img, scale(10), scale(15), scale(4), transparent)
	draw.Draw(img, image.Rect(scale(16), scale(13), scale(29), scale(18)), &image.Uniform{key}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(scale(22), scale(17), scale(26), scale(23)), &image.Uniform{key}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(scale(26), scale(17), scale(30), scale(21)), &image.Uniform{key}, image.Point{}, draw.Src)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func makeKeyIconICO() []byte {
	sizes := []int{16, 32, 48, 256}
	images := make([][]byte, 0, len(sizes))
	for _, size := range sizes {
		images = append(images, makeKeyIconPNG(size))
	}

	var buf bytes.Buffer
	writeIconLE(&buf, uint16(0))          // reserved
	writeIconLE(&buf, uint16(1))          // icon
	writeIconLE(&buf, uint16(len(sizes))) // image count
	imageOffset := 6 + len(sizes)*16      // ICONDIR + ICONDIRENTRY records
	for i, size := range sizes {
		widthByte := byte(size)
		heightByte := byte(size)
		if size >= 256 {
			widthByte = 0
			heightByte = 0
		}
		buf.WriteByte(widthByte)
		buf.WriteByte(heightByte)
		buf.WriteByte(0) // color count
		buf.WriteByte(0) // reserved
		writeIconLE(&buf, uint16(1))
		writeIconLE(&buf, uint16(32))
		writeIconLE(&buf, uint32(len(images[i])))
		writeIconLE(&buf, uint32(imageOffset))
		imageOffset += len(images[i])
	}
	for _, img := range images {
		buf.Write(img)
	}
	return buf.Bytes()
}

func writeIconLE(buf *bytes.Buffer, v any) {
	if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
		panic(err)
	}
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
