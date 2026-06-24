//go:build windows

package paste

import "syscall"

const (
	vkControl     = 0x11
	vkV           = 0x56
	keyEventKeyUp = 0x0002
)

var (
	user32Paste = syscall.NewLazyDLL("user32.dll")
	keybdEvent  = user32Paste.NewProc("keybd_event")
)

func Send() error {
	keybdEvent.Call(vkControl, 0, 0, 0)
	keybdEvent.Call(vkV, 0, 0, 0)
	keybdEvent.Call(vkV, 0, keyEventKeyUp, 0)
	keybdEvent.Call(vkControl, 0, keyEventKeyUp, 0)
	return nil
}
