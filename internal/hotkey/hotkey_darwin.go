//go:build darwin

package hotkey

/*
#cgo LDFLAGS: -framework Carbon
#include <Carbon/Carbon.h>

extern void goHotkeyPressed(void);

static EventHotKeyRef hotKeyRef = NULL;
static EventHandlerRef handlerRef = NULL;

static OSStatus handleHotKey(EventHandlerCallRef nextHandler, EventRef event, void *userData) {
	goHotkeyPressed();
	return noErr;
}

static int registerCarbonHotKey(UInt32 keyCode, UInt32 modifiers) {
	if (handlerRef == NULL) {
		EventTypeSpec eventType;
		eventType.eventClass = kEventClassKeyboard;
		eventType.eventKind = kEventHotKeyPressed;
		OSStatus handlerStatus = InstallApplicationEventHandler(&handleHotKey, 1, &eventType, NULL, &handlerRef);
		if (handlerStatus != noErr) {
			return (int)handlerStatus;
		}
	}

	if (hotKeyRef != NULL) {
		UnregisterEventHotKey(hotKeyRef);
		hotKeyRef = NULL;
	}

	EventHotKeyID hotKeyID;
	hotKeyID.signature = 'gpas';
	hotKeyID.id = 1;
	OSStatus status = RegisterEventHotKey(keyCode, modifiers, hotKeyID, GetApplicationEventTarget(), 0, &hotKeyRef);
	return (int)status;
}

static void unregisterCarbonHotKey(void) {
	if (hotKeyRef != NULL) {
		UnregisterEventHotKey(hotKeyRef);
		hotKeyRef = NULL;
	}
}
*/
import "C"

import (
	"fmt"
	"sync/atomic"

	"gopass/internal/shortcut"
)

const (
	carbonControl = 1 << 12
	carbonShift   = 1 << 9
	carbonCommand = 1 << 8
)

var hotkeyCallback atomic.Value

func register(value string, callback func()) error {
	parsed, err := shortcut.Parse(value)
	if err != nil {
		return err
	}

	keyCode, ok := macKeyCode(parsed.Key)
	if !ok {
		return fmt.Errorf("unsupported shortcut key %q", parsed.Key)
	}

	var modifiers C.UInt32
	if parsed.Control {
		modifiers |= carbonControl
	}
	if parsed.Shift {
		modifiers |= carbonShift
	}
	if parsed.Command {
		modifiers |= carbonCommand
	}
	if parsed.Windows {
		return fmt.Errorf("the Windows modifier is not available on macOS")
	}

	hotkeyCallback.Store(callback)
	if status := C.registerCarbonHotKey(C.UInt32(keyCode), modifiers); status != 0 {
		return fmt.Errorf("register hotkey failed with status %d", int(status))
	}
	return nil
}

func unregister() {
	C.unregisterCarbonHotKey()
}

//export goHotkeyPressed
func goHotkeyPressed() {
	if callback, ok := hotkeyCallback.Load().(func()); ok && callback != nil {
		go callback()
	}
}

func macKeyCode(key string) (uint32, bool) {
	codes := map[string]uint32{
		"A": 0x00, "S": 0x01, "D": 0x02, "F": 0x03, "H": 0x04, "G": 0x05,
		"Z": 0x06, "X": 0x07, "C": 0x08, "V": 0x09, "B": 0x0B, "Q": 0x0C,
		"W": 0x0D, "E": 0x0E, "R": 0x0F, "Y": 0x10, "T": 0x11, "1": 0x12,
		"2": 0x13, "3": 0x14, "4": 0x15, "6": 0x16, "5": 0x17, "=": 0x18,
		"9": 0x19, "7": 0x1A, "-": 0x1B, "8": 0x1C, "0": 0x1D, "]": 0x1E,
		"O": 0x1F, "U": 0x20, "[": 0x21, "I": 0x22, "P": 0x23, "L": 0x25,
		"J": 0x26, "'": 0x27, "K": 0x28, ";": 0x29, "\\": 0x2A, ",": 0x2B,
		"/": 0x2C, "N": 0x2D, "M": 0x2E, ".": 0x2F,
	}
	code, ok := codes[key]
	return code, ok
}
