package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"
	"time"

	"gopass/internal/clipboard"
	"gopass/internal/password"

	"github.com/gogpu/systray"
)

const appName = "GoPass"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	tray := systray.New()
	menu := systray.NewMenu()

	menu.Add("Copy password", func() {
		copyPassword(tray)
	})
	menu.AddSeparator()
	menu.Add("Quit", func() {
		tray.Remove()
		os.Exit(0)
	})

	iconPNG := makeKeyIconPNG()
	tray.SetIcon(iconPNG).
		SetTemplateIcon(iconPNG).
		SetTooltip(appName + ": left-click to copy a Norwegian password").
		SetMenu(menu)

	// Left-clicking the macOS menu-bar icon or Windows taskbar tray icon copies a new password.
	tray.OnClick(func() {
		copyPassword(tray)
	})

	tray.Show()

	if err := tray.Run(); err != nil {
		log.Fatalf("tray loop failed: %v", err)
	}
}

func copyPassword(tray *systray.SystemTray) {
	pw, err := password.Generate()
	if err != nil {
		tray.SetTooltip(appName + ": could not generate password")
		log.Printf("password generation failed: %v", err)
		return
	}

	if err := clipboard.CopyText(pw); err != nil {
		tray.SetTooltip(appName + ": clipboard is unavailable")
		log.Printf("clipboard copy failed: %v", err)
		return
	}

	tray.SetTooltip(fmt.Sprintf(appName+": password copied at %s", time.Now().Format("15:04:05")))

	// Do not log, display, or notify the actual password. It is only written to the clipboard.
}

func makeKeyIconPNG() []byte {
	const size = 32

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	transparent := color.RGBA{0, 0, 0, 0}
	draw.Draw(img, img.Bounds(), &image.Uniform{transparent}, image.Point{}, draw.Src)

	// Pixel-art key tray icon. PNG is accepted by gogpu/systray on macOS and Windows.
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
