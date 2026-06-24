package settingsui

import (
	"runtime"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
)

func TestShortcutCaptureRecordsHeldShortcut(t *testing.T) {
	test.NewApp()

	capture := newShortcutCapture()

	capture.KeyDown(&fyne.KeyEvent{Name: desktop.KeyControlLeft})
	capture.KeyDown(&fyne.KeyEvent{Name: desktop.KeyShiftLeft})
	capture.KeyDown(&fyne.KeyEvent{Name: fyne.KeyP})

	if got := capture.Text; got != "Ctrl+Shift+P" {
		t.Fatalf("captured shortcut = %q, want %q", got, "Ctrl+Shift+P")
	}
}

func TestShortcutCaptureUsesPlatformSuperName(t *testing.T) {
	test.NewApp()

	capture := newShortcutCapture()

	capture.KeyDown(&fyne.KeyEvent{Name: desktop.KeySuperLeft})
	capture.KeyDown(&fyne.KeyEvent{Name: fyne.KeyP})

	want := "Windows+P"
	if runtime.GOOS == "darwin" {
		want = "Command+P"
	}
	if got := capture.Text; got != want {
		t.Fatalf("captured shortcut = %q, want %q", got, want)
	}
}
