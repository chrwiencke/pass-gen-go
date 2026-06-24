//go:build darwin

package paste

/*
#cgo LDFLAGS: -framework ApplicationServices
#include <ApplicationServices/ApplicationServices.h>

static void sendPasteShortcut(void) {
	CGEventSourceRef source = CGEventSourceCreate(kCGEventSourceStateHIDSystemState);
	CGEventRef down = CGEventCreateKeyboardEvent(source, (CGKeyCode)9, true);
	CGEventRef up = CGEventCreateKeyboardEvent(source, (CGKeyCode)9, false);
	CGEventSetFlags(down, kCGEventFlagMaskCommand);
	CGEventSetFlags(up, kCGEventFlagMaskCommand);
	CGEventPost(kCGHIDEventTap, down);
	CGEventPost(kCGHIDEventTap, up);
	CFRelease(down);
	CFRelease(up);
	CFRelease(source);
}
*/
import "C"

func Send() error {
	C.sendPasteShortcut()
	return nil
}
