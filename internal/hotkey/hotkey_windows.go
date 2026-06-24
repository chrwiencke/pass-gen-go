//go:build windows

package hotkey

import (
	"fmt"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"gopass/internal/shortcut"
)

const (
	hotkeyID   = 1
	modAlt     = 0x0001
	modControl = 0x0002
	modShift   = 0x0004
	modWin     = 0x0008
	wmHotkey   = 0x0312
)

var (
	user32Windows         = syscall.NewLazyDLL("user32.dll")
	registerHotKeyProc    = user32Windows.NewProc("RegisterHotKey")
	unregisterHotKeyProc  = user32Windows.NewProc("UnregisterHotKey")
	getMessageProc        = user32Windows.NewProc("GetMessageW")
	postThreadMessageProc = user32Windows.NewProc("PostThreadMessageW")
	kernel32Windows       = syscall.NewLazyDLL("kernel32.dll")
	getCurrentThreadID    = kernel32Windows.NewProc("GetCurrentThreadId")

	messageThreadID uint32
)

type msg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

type point struct {
	x int32
	y int32
}

func register(value string, callback func()) error {
	parsed, err := shortcut.Parse(value)
	if err != nil {
		return err
	}
	if parsed.Command {
		return fmt.Errorf("the Command modifier is not available on Windows")
	}

	var modifiers uintptr
	if parsed.Control {
		modifiers |= modControl
	}
	if parsed.Windows {
		modifiers |= modWin
	}

	key := uintptr(parsed.Key[0])
	unregister()
	ready := make(chan error, 1)
	go messageLoop(modifiers, key, callback, ready)
	return <-ready
}

func unregister() {
	if messageThreadID != 0 {
		postThreadMessageProc.Call(uintptr(messageThreadID), 0x0012, 0, 0)
		messageThreadID = 0
		time.Sleep(50 * time.Millisecond)
	}
}

func messageLoop(modifiers uintptr, key uintptr, callback func(), ready chan<- error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer unregisterHotKeyProc.Call(0, hotkeyID)

	threadID, _, _ := getCurrentThreadID.Call()
	messageThreadID = uint32(threadID)

	if r, _, err := registerHotKeyProc.Call(0, hotkeyID, modifiers, key); r == 0 {
		ready <- fmt.Errorf("RegisterHotKey failed: %w", windowsErr(err))
		return
	}
	ready <- nil

	var m msg
	for {
		r, _, _ := getMessageProc.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			return
		}
		if m.message == wmHotkey && callback != nil {
			go callback()
		}
	}
}

func windowsErr(err error) error {
	if errno, ok := err.(syscall.Errno); ok && errno == 0 {
		return syscall.EINVAL
	}
	return err
}

var _ = modAlt
var _ = modShift
